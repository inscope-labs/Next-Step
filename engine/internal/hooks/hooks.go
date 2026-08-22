// Package hooks implements the four-point extension mechanism documented
// in root/protocol/v1.0/spec/PROTOCOL-FACTS.md's "Hook / gate
// architecture" section.
//
// This is the lookup/exec mechanism only, per the Phase 5.5.0.3 decision:
// implement the mechanism now (small, well-specified, and already fully
// documented), defer writing any actual hook scripts. No hook files ship
// with this package; every call point is a true no-op until a human
// installs a script at the documented path.
//
// Scope for v1.4.0: this is explicitly the intended extension surface for
// a future concurrent multi-agent execution model — see the package doc
// in engine/internal/task for why no new enforcement logic should grow
// into the task lifecycle itself instead of through these points.
package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Point names the four named extension points, exactly as documented in
// PROTOCOL-FACTS.md's hook table.
type Point string

const (
	Ingress     Point = "INGRESS"      // gate, fires before staging, can block
	PreExecute  Point = "PRE_EXECUTE"  // gate, fires before start.sh runs, can block
	PostExecute Point = "POST_EXECUTE" // hook, fires after start.sh runs, observational only
	Egress      Point = "EGRESS"       // hook, fires after the receipt/ledger is committed, observational only
)

// CanBlock reports whether a non-zero exit from this point's hook script
// should block the operation it wraps. This is not a new decision
// invented here — it is a direct read of PROTOCOL-FACTS.md's "Can block?"
// column: INGRESS and PRE_EXECUTE are gates (yes); POST_EXECUTE and
// EGRESS are hooks, observational only (no).
func (p Point) CanBlock() bool {
	return p == Ingress || p == PreExecute
}

// Lookup implements the documented lookup order:
// $WORKSPACE_ROOT/hooks/<name>, then $NEXT_STEP_HOME/hooks/<name>. If
// neither exists and is executable, returns ("", false) — the true no-op
// case.
func Lookup(home, workspaceRoot string, point Point) (path string, found bool) {
	candidates := []string{
		filepath.Join(workspaceRoot, "hooks", string(point)),
		filepath.Join(home, "hooks", string(point)),
	}
	for _, c := range candidates {
		if isExecutable(c) {
			return c, true
		}
	}
	return "", false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

// Result is what Fire returns: whether a hook actually ran, its combined
// output, and its exit error (nil on success).
type Result struct {
	Ran    bool
	Output string
	Err    error
}

// Fire looks up and, if found, executes the hook for the given point.
// True no-op behavior when absent: returns Result{Ran: false} with a nil
// error — writes nothing, blocks nothing. When present, the hook is
// exec'd as a separate process (never sourced), with context passed via
// NEXT_STEP_* environment variables per PROTOCOL-FACTS.md.
//
// extraEnv supplies point-specific context (e.g. NEXT_STEP_TASK_ID,
// NEXT_STEP_ZIP_PATH) on top of the base NEXT_STEP_HOME /
// NEXT_STEP_WORKSPACE_ROOT / NEXT_STEP_HOOK_NAME set here.
func Fire(home, workspaceRoot string, point Point, extraEnv map[string]string) Result {
	path, found := Lookup(home, workspaceRoot, point)
	if !found {
		return Result{Ran: false}
	}

	env := append(os.Environ(),
		"NEXT_STEP_HOME="+home,
		"NEXT_STEP_WORKSPACE_ROOT="+workspaceRoot,
		"NEXT_STEP_HOOK_NAME="+string(point),
	)
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}

	cmd := exec.Command(path)
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()

	return Result{Ran: true, Output: buf.String(), Err: err}
}

// FireBlocking is Fire, but converts a CanBlock point's non-zero exit into
// a returned error the caller must treat as a hard stop. Non-blocking
// points (POST_EXECUTE, EGRESS) never return an error here — a failing
// observational hook is logged by the caller via Result.Err, not treated
// as a gate failure.
func FireBlocking(home, workspaceRoot string, point Point, extraEnv map[string]string) (Result, error) {
	r := Fire(home, workspaceRoot, point, extraEnv)
	if r.Ran && r.Err != nil && point.CanBlock() {
		return r, fmt.Errorf("hook %s at %s blocked the operation: %w\noutput:\n%s",
			point, home, r.Err, r.Output)
	}
	return r, nil
}
