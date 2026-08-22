// Package task implements task zip build and validation, plus the
// show/approve/run lifecycle that operates on a built zip.
//
// Replicates build-task.sh and run-task.sh's behavior from the retired
// ABX-STEP lineage. The build plan's Phase 5 lists "task" as owning
// "build and validation" specifically; show/approve/run aren't split into
// a separate named package in the plan, but they operate on the same
// artifact (the task zip) and the same lifecycle, so this package owns
// them too rather than inventing an unlisted fifth package.
//
// IMPORTANT — trust boundary: Run() executes a task's start.sh/verify.sh
// directly on the host, post human-approval. This is NOT a regression from
// docs/security-model.md's "task execution must never touch the host"
// principle — that principle describes the target state once
// next-step-runner (sandboxed execution) is functional, which is
// explicitly out of scope for v1.0. v1.0's job is feature parity with
// ABX-STEP v1.2.0, which also executed directly on the host with no
// sandbox, gated only by the human --approve step. This package matches
// that existing real behavior; it does not introduce a new gap.
package task

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

// Manifest mirrors root/protocol/v1.0/schemas/task-manifest.schema.json.
// That schema formalizes the logical field set/types; the on-disk file
// format is plain KEY=VALUE lines (matching demo/task.manifest.demo), not
// literal JSON — this struct's parser reads that real format.
type Manifest struct {
	ProtocolVersion        string
	TaskID                 string
	TaskVersion            int
	CreatedAt              string
	CreatedBy              string
	WorkspaceID            string
	WorkspaceName          string
	TaskIntent             string
	TaskContentHash        string
	WritePaths             string // comma-separated relative paths, or "NONE"
	IdempotencyMode        string
	ExecutionContext       string
	AllowNetwork           bool
	AllowDestructive       bool
	AllowRemoteExec        bool
	AllowPrivileged        bool
	AllowCredentialAccess  bool
	AllowProcessControl    bool
	RequireCleanWorkingDir bool
	InterruptionPolicy     string
}

var requiredFields = []string{
	"PROTOCOL_VERSION", "TASK_ID", "TASK_VERSION", "CREATED_AT", "CREATED_BY",
	"WORKSPACE_ID", "WORKSPACE_NAME", "TASK_INTENT", "TASK_CONTENT_HASH",
	"WRITE_PATHS", "IDEMPOTENCY_MODE", "EXECUTION_CONTEXT", "ALLOW_NETWORK",
	"ALLOW_DESTRUCTIVE", "ALLOW_REMOTE_EXEC", "ALLOW_PRIVILEGED",
	"ALLOW_CREDENTIAL_ACCESS", "ALLOW_PROCESS_CONTROL",
	"REQUIRE_CLEAN_WORKING_DIR", "INTERRUPTION_POLICY",
}

// ParseManifest reads the KEY=VALUE manifest format.
func ParseManifest(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("opening manifest: %w", err)
	}
	defer f.Close()

	fields := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			return Manifest{}, fmt.Errorf("manifest line is not KEY=VALUE: %q", line)
		}
		fields[line[:idx]] = line[idx+1:]
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, fmt.Errorf("reading manifest: %w", err)
	}

	var missing []string
	for _, k := range requiredFields {
		if _, ok := fields[k]; !ok {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return Manifest{}, fmt.Errorf("manifest missing required field(s): %s", strings.Join(missing, ", "))
	}

	taskVersion, err := strconv.Atoi(fields["TASK_VERSION"])
	if err != nil {
		return Manifest{}, fmt.Errorf("TASK_VERSION is not an integer: %q", fields["TASK_VERSION"])
	}
	parseBool := func(key string) (bool, error) {
		v := fields[key]
		switch v {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return false, fmt.Errorf("%s must be true or false, got %q", key, v)
		}
	}
	boolFields := map[string]*bool{}
	m := Manifest{
		ProtocolVersion:    fields["PROTOCOL_VERSION"],
		TaskID:             fields["TASK_ID"],
		TaskVersion:        taskVersion,
		CreatedAt:          fields["CREATED_AT"],
		CreatedBy:          fields["CREATED_BY"],
		WorkspaceID:        fields["WORKSPACE_ID"],
		WorkspaceName:      fields["WORKSPACE_NAME"],
		TaskIntent:         fields["TASK_INTENT"],
		TaskContentHash:    fields["TASK_CONTENT_HASH"],
		WritePaths:         fields["WRITE_PATHS"],
		IdempotencyMode:    fields["IDEMPOTENCY_MODE"],
		ExecutionContext:   fields["EXECUTION_CONTEXT"],
		InterruptionPolicy: fields["INTERRUPTION_POLICY"],
	}
	boolFields["ALLOW_NETWORK"] = &m.AllowNetwork
	boolFields["ALLOW_DESTRUCTIVE"] = &m.AllowDestructive
	boolFields["ALLOW_REMOTE_EXEC"] = &m.AllowRemoteExec
	boolFields["ALLOW_PRIVILEGED"] = &m.AllowPrivileged
	boolFields["ALLOW_CREDENTIAL_ACCESS"] = &m.AllowCredentialAccess
	boolFields["ALLOW_PROCESS_CONTROL"] = &m.AllowProcessControl
	boolFields["REQUIRE_CLEAN_WORKING_DIR"] = &m.RequireCleanWorkingDir
	for key, dst := range boolFields {
		v, err := parseBool(key)
		if err != nil {
			return Manifest{}, err
		}
		*dst = v
	}
	return m, nil
}

// ValidateWritePaths enforces path confinement: every declared write path
// must resolve inside the claiming workspace's root. Rejects absolute
// paths, "..", and "~" lexically, at manifest-validation time — before
// staging, per task-acceptance-criteria.md §6. This is a lexical check,
// not a kernel-level enforcement guarantee; see PROTOCOL-FACTS.md's
// Path enforcement rule and Link/capability caveat for the documented
// limits of what this actually guarantees.
func ValidateWritePaths(writePaths string) error {
	if writePaths == "NONE" {
		return nil
	}
	for _, p := range strings.Split(writePaths, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			return fmt.Errorf("WRITE_PATHS contains an empty entry")
		}
		if filepath.IsAbs(p) {
			return fmt.Errorf("WRITE_PATHS entry %q is absolute; must be relative to the workspace root", p)
		}
		if strings.HasPrefix(p, "~") {
			return fmt.Errorf("WRITE_PATHS entry %q uses ~; must be relative to the workspace root", p)
		}
		cleaned := filepath.Clean(p)
		if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			return fmt.Errorf("WRITE_PATHS entry %q escapes the workspace root", p)
		}
	}
	return nil
}

// contentHash computes TASK_CONTENT_HASH: sha256 over the manifest (with
// TASK_CONTENT_HASH itself blanked, so the hash doesn't depend on its own
// prior value) concatenated with start.sh and verify.sh, in that fixed
// order, so the hash is deterministic and reproducible from the same
// inputs.
func contentHash(manifestPath, startPath, verifyPath string) (string, error) {
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("reading manifest for hash: %w", err)
	}
	// Blank the TASK_CONTENT_HASH line so hashing is stable regardless of
	// its current value (PENDING, a stale hash from a prior build, etc).
	var blanked strings.Builder
	for _, line := range strings.Split(string(manifestBytes), "\n") {
		if strings.HasPrefix(line, "TASK_CONTENT_HASH=") {
			blanked.WriteString("TASK_CONTENT_HASH=\n")
			continue
		}
		blanked.WriteString(line)
		blanked.WriteString("\n")
	}
	startBytes, err := os.ReadFile(startPath)
	if err != nil {
		return "", fmt.Errorf("reading start.sh for hash: %w", err)
	}
	verifyBytes, err := os.ReadFile(verifyPath)
	if err != nil {
		return "", fmt.Errorf("reading verify.sh for hash: %w", err)
	}

	h := sha256.New()
	h.Write([]byte(blanked.String()))
	h.Write(startBytes)
	h.Write(verifyBytes)
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// lookupWorkspaceName reads workspace/registry/<ID>/name directly, since
// WORKSPACE_NAME is looked up and injected at build time, never
// hand-filled — see demo/README.md.
func lookupWorkspaceName(home, workspaceID string) (string, error) {
	info, err := registry.Load(home, workspaceID)
	if err != nil {
		return "", err
	}
	if info.Name == "" {
		return "", fmt.Errorf("workspace %s has no registered name", workspaceID)
	}
	return info.Name, nil
}
