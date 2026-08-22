// Package registry implements workspace claim/registry logic.
//
// Replicates the atomic-claim semantics the retired ABX-STEP lineage
// achieved with a shell mkdir race-check, but as a Go-native operation
// (os.Mkdir, which is itself atomic on POSIX — same underlying guarantee,
// no new primitive introduced) rather than shelling out.
//
// Ground-truth paths, per root/protocol/v1.0/spec/PROTOCOL-FACTS.md:
//   - workspace/<ID>/                one directory per claimed workspace
//   - workspace/registry/<ID>/       existence is the claim
//   - sessions/active                default-workspace pointer, human-set
//
// These are top-level under $NEXT_STEP_HOME, distinct from root/workspaces/
// (plural, templates/docs/legacy reference material only).
package registry

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrWorkspaceRequired is returned when an operation needed a workspace ID
// (either explicit or via sessions/active) and neither was available. This
// is always a hard rejection, never a silent guess — see
// docs/architecture-overview.md §6.
var ErrWorkspaceRequired = errors.New("workspace required: no --workspace given and sessions/active is not set")

// subdirs is the full per-workspace substructure, per PROTOCOL-FACTS.md's
// "holding its own inbox/tasks/approvals/receipts/logs/locks/files".
var subdirs = []string{"inbox", "tasks", "approvals", "receipts", "logs", "locks", "files"}

// Info holds a claimed workspace's registry metadata.
type Info struct {
	ID      string
	Name    string
	Created string // UTC, YYYY-MM-DDTHH:MM:SSZ
	Creator string
	Purpose string
}

// newID generates a fresh RFC 4122 v4 UUID using only crypto/rand — no
// external module, since this environment's egress allowlist has no
// proxy.golang.org/sum.golang.org and every other Go package here is
// standard-library-only for the same reason.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating workspace id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Claim atomically claims a new workspace. Retries on UUID collision
// (astronomically unlikely, but this is the documented "a failed mkdir
// means try another ID" behavior, not a new guarantee invented here).
func Claim(home, name, purpose, creator string) (Info, error) {
	if strings.TrimSpace(name) == "" {
		return Info{}, errors.New("workspace name is required")
	}
	registryRoot := filepath.Join(home, "workspace", "registry")
	if err := os.MkdirAll(registryRoot, 0o755); err != nil {
		return Info{}, fmt.Errorf("preparing registry root: %w", err)
	}

	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := newID()
		if err != nil {
			return Info{}, err
		}
		claimPath := filepath.Join(registryRoot, id)
		if err := os.Mkdir(claimPath, 0o755); err != nil {
			if os.IsExist(err) {
				continue // try another ID, per documented behavior
			}
			return Info{}, fmt.Errorf("claiming workspace: %w", err)
		}

		info := Info{
			ID:      id,
			Name:    name,
			Created: time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			Creator: creator,
			Purpose: purpose,
		}
		if err := writeRegistryFiles(claimPath, info); err != nil {
			return Info{}, err
		}
		if err := createWorkspaceTree(home, id); err != nil {
			return Info{}, err
		}
		return info, nil
	}
	return Info{}, fmt.Errorf("claiming workspace: exhausted %d attempts on ID collision", maxAttempts)
}

func writeRegistryFiles(claimPath string, info Info) error {
	files := map[string]string{
		"name":    info.Name,
		"created": info.Created,
		"creator": info.Creator,
		"purpose": info.Purpose,
	}
	for filename, content := range files {
		p := filepath.Join(claimPath, filename)
		if err := os.WriteFile(p, []byte(content+"\n"), 0o644); err != nil {
			return fmt.Errorf("writing registry/%s: %w", filename, err)
		}
	}
	return nil
}

func createWorkspaceTree(home, id string) error {
	root := filepath.Join(home, "workspace", id)
	for _, d := range subdirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return fmt.Errorf("creating workspace/%s/%s: %w", id, d, err)
		}
	}
	return nil
}

// Exists reports whether a workspace ID is claimed (registry entry present).
func Exists(home, id string) bool {
	_, err := os.Stat(filepath.Join(home, "workspace", "registry", id))
	return err == nil
}

// Load reads a claimed workspace's registry metadata.
func Load(home, id string) (Info, error) {
	if !Exists(home, id) {
		return Info{}, fmt.Errorf("workspace %s is not claimed (no registry entry)", id)
	}
	claimPath := filepath.Join(home, "workspace", "registry", id)
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(claimPath, name))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	return Info{
		ID:      id,
		Name:    read("name"),
		Created: read("created"),
		Creator: read("creator"),
		Purpose: read("purpose"),
	}, nil
}

// Root returns the workspace's live instance directory: workspace/<ID>/.
func Root(home, id string) string {
	return filepath.Join(home, "workspace", id)
}

// ActiveSessionPath is sessions/active — the default-workspace pointer.
func ActiveSessionPath(home string) string {
	return filepath.Join(home, "sessions", "active")
}

// SetActive writes sessions/active, the human-set default-workspace
// pointer read by task operations when --workspace is omitted. Refuses to
// point at an unclaimed workspace.
func SetActive(home, id string) error {
	if !Exists(home, id) {
		return fmt.Errorf("cannot set active session: workspace %s is not claimed", id)
	}
	p := ActiveSessionPath(home)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("preparing sessions dir: %w", err)
	}
	if err := os.WriteFile(p, []byte(id+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing sessions/active: %w", err)
	}
	return nil
}

// GetActive reads sessions/active. Returns "" (not an error) if unset —
// callers combine this with ErrWorkspaceRequired logic in Resolve.
func GetActive(home string) string {
	b, err := os.ReadFile(ActiveSessionPath(home))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// Resolve implements the documented --workspace resolution rule: use the
// explicit ID if given; otherwise fall back to sessions/active; otherwise
// hard-reject. Never silently guesses (no revival of the old 00000
// fallback bug).
func Resolve(home, explicitID string) (string, error) {
	if explicitID != "" {
		if !Exists(home, explicitID) {
			return "", fmt.Errorf("workspace %s is not claimed", explicitID)
		}
		return explicitID, nil
	}
	active := GetActive(home)
	if active == "" {
		return "", ErrWorkspaceRequired
	}
	if !Exists(home, active) {
		return "", fmt.Errorf("sessions/active points at %s, which is not claimed", active)
	}
	return active, nil
}
