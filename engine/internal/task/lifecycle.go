package task

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

// Show extracts and returns a human-readable summary of a built task zip's
// manifest and contents — what a human reviews before --approve. This is
// read-only; it never executes anything from the zip.
func Show(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	var manifestContent string
	var fileNames []string
	for _, f := range r.File {
		fileNames = append(fileNames, f.Name)
		if f.Name == "task.manifest" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("reading task.manifest from zip: %w", err)
			}
			b, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("reading task.manifest from zip: %w", err)
			}
			manifestContent = string(b)
		}
	}
	if manifestContent == "" {
		return "", fmt.Errorf("zip has no task.manifest entry")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "=== %s ===\n", filepath.Base(zipPath))
	fmt.Fprintf(&sb, "Files: %s\n\n", strings.Join(fileNames, ", "))
	sb.WriteString("--- task.manifest ---\n")
	sb.WriteString(manifestContent)
	return sb.String(), nil
}

func approvalPath(home, workspaceID, zipPath string) string {
	base := filepath.Base(zipPath)
	return filepath.Join(registry.Root(home, workspaceID), "approvals", base+".approved")
}

// WorkspaceIDFromZipPath derives the workspace ID from a task zip's own
// path (workspace/<ID>/tasks/<file>.zip), rather than trusting a
// separately-typed --workspace flag that could mismatch the zip's actual
// location. Validates the derived ID is actually claimed.
func WorkspaceIDFromZipPath(home, zipPath string) (string, error) {
	abs, err := filepath.Abs(zipPath)
	if err != nil {
		return "", fmt.Errorf("resolving zip path: %w", err)
	}
	tasksDir := filepath.Dir(abs)
	if filepath.Base(tasksDir) != "tasks" {
		return "", fmt.Errorf("zip path %q is not under a workspace's tasks/ directory as expected", zipPath)
	}
	workspaceID := filepath.Base(filepath.Dir(tasksDir))
	if !registry.Exists(home, workspaceID) {
		return "", fmt.Errorf("derived workspace %q from zip path is not claimed", workspaceID)
	}
	return workspaceID, nil
}

// Approve records human authorization for a built task zip. This is the
// mechanism, not the gate itself — the binary cannot verify who physically
// invoked it. The actual gate is procedural: the onboarding chain
// (state-2.md, Step 5) instructs the agent to hand this exact command to
// the human and wait, never to run it itself. See
// docs/security-model.md §1 and root/protocol/v1.0/spec/PROTOCOL-FACTS.md's
// Authorization model.
func Approve(home, workspaceID, zipPath, approver string) error {
	if _, err := os.Stat(zipPath); err != nil {
		return fmt.Errorf("zip not found: %w", err)
	}
	p := approvalPath(home, workspaceID, zipPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("preparing approvals dir: %w", err)
	}
	content := fmt.Sprintf("approved_at=%s\napproved_by=%s\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"), approver)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing approval marker: %w", err)
	}
	return nil
}

func isApproved(home, workspaceID, zipPath string) bool {
	_, err := os.Stat(approvalPath(home, workspaceID, zipPath))
	return err == nil
}

// Report is the printed result of a Run, matching the field contract
// documented in PROTOCOL-FACTS.md's Report field contract and
// task-acceptance-criteria.md §4: EXECUTION, IDEMPOTENCY, EXECUTION_COUNT,
// ATTEMPT_COUNT must agree with LOG_TAIL. VERIFICATION is the field that
// actually determines acceptance — a zero exit code alone is not
// sufficient (task-acceptance-criteria.md §5).
type Report struct {
	Execution       string // APPLIED | NOOP, self-reported by start.sh
	IdempotencyMode string // declared mode from the manifest (v1.0 does not
	// verify actual idempotency by re-running and diffing; that would need
	// a two-run comparison mechanism not yet specified — flagged here as a
	// known v1.0 simplification, not a silent omission.
	ExecutionCount int
	AttemptCount   int
	LogTail        string
	Verification   string // PASS | FAIL
}

func (r Report) String() string {
	return fmt.Sprintf(
		"EXECUTION: %s\nIDEMPOTENCY_MODE: %s\nEXECUTION_COUNT: %d\nATTEMPT_COUNT: %d\nVERIFICATION: %s\nLOG_TAIL:\n%s",
		r.Execution, r.IdempotencyMode, r.ExecutionCount, r.AttemptCount, r.Verification, r.LogTail,
	)
}

// Run replicates run-task.sh's default (no-flag) execution path: hard-gates
// on approval, extracts the zip, executes start.sh then verify.sh directly
// on the host (see this file's package doc for why that's correct v1.0
// scope, not a sandbox regression), and produces the report.
//
// ALLOW_NETWORK / ALLOW_DESTRUCTIVE / etc. manifest flags are declarative
// only in v1.0 — shown to the human at --show time for review, but not
// physically enforced at runtime (there is no sandbox to enforce them
// against yet). Do not treat a task as safe to run unattended on the
// strength of these flags alone.
func Run(home, workspaceID, zipPath string) (Report, error) {
	if !isApproved(home, workspaceID, zipPath) {
		return Report{}, fmt.Errorf("task is not approved: run Approve first (human-authorized step)")
	}

	wsRoot := registry.Root(home, workspaceID)
	tmpDir, err := os.MkdirTemp("", "next-step-run-*")
	if err != nil {
		return Report{}, fmt.Errorf("creating scratch dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZip(zipPath, tmpDir); err != nil {
		return Report{}, fmt.Errorf("extracting zip: %w", err)
	}

	m, err := ParseManifest(filepath.Join(tmpDir, "task.manifest"))
	if err != nil {
		return Report{}, fmt.Errorf("re-validating manifest at run time: %w", err)
	}

	env := append(os.Environ(),
		"NEXT_STEP_HOME="+home,
		"NEXT_STEP_WORKSPACE_ROOT="+wsRoot,
	)

	startOut, startErr := runScript(filepath.Join(tmpDir, "start.sh"), tmpDir, env)
	execution := parseResultLine(startOut)
	if execution == "" {
		execution = "UNKNOWN"
	}

	verifyOut, verifyErr := runScript(filepath.Join(tmpDir, "verify.sh"), tmpDir, env)
	verification := "FAIL"
	if verifyErr == nil {
		verification = "PASS"
	}

	logTail := strings.Join([]string{
		"--- start.sh ---", startOut,
		"--- verify.sh ---", verifyOut,
	}, "\n")
	if startErr != nil {
		logTail += fmt.Sprintf("\n[start.sh exited non-zero: %v]", startErr)
	}
	if verifyErr != nil {
		logTail += fmt.Sprintf("\n[verify.sh exited non-zero: %v]", verifyErr)
	}

	attemptCount, executionCount, err := bumpCounters(wsRoot, filepath.Base(zipPath), execution == "APPLIED")
	if err != nil {
		return Report{}, fmt.Errorf("updating attempt/execution counters: %w", err)
	}

	report := Report{
		Execution:       execution,
		IdempotencyMode: m.IdempotencyMode,
		ExecutionCount:  executionCount,
		AttemptCount:    attemptCount,
		LogTail:         logTail,
		Verification:    verification,
	}

	if err := appendLog(wsRoot, filepath.Base(zipPath), report); err != nil {
		return report, fmt.Errorf("run completed but writing log failed: %w", err)
	}
	return report, nil
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		destPath := filepath.Join(destDir, filepath.Base(f.Name))
		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func runScript(path, dir string, env []string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("script not found: %w", err)
	}
	cmd := exec.Command("/bin/bash", path)
	cmd.Dir = dir
	cmd.Env = env
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// parseResultLine finds the single NEXT_STEP_RESULT=APPLIED|NOOP line in
// start.sh's output, per the self-reported-result contract
// (task-acceptance-criteria.md §3).
func parseResultLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NEXT_STEP_RESULT=") {
			v := strings.TrimPrefix(line, "NEXT_STEP_RESULT=")
			if v == "APPLIED" || v == "NOOP" {
				return v
			}
		}
	}
	return ""
}

func counterPath(wsRoot, zipBase string) string {
	return filepath.Join(wsRoot, "logs", zipBase+".counters")
}

// bumpCounters increments ATTEMPT_COUNT on every call and EXECUTION_COUNT
// only when this attempt actually applied a change, and returns the new
// totals. Simple flat-file counter, consistent with this package's other
// plain-text state files.
func bumpCounters(wsRoot, zipBase string, applied bool) (attempt, execution int, err error) {
	p := counterPath(wsRoot, zipBase)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, 0, err
	}
	if b, readErr := os.ReadFile(p); readErr == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(line, "attempt_count="); ok {
				attempt, _ = strconv.Atoi(strings.TrimSpace(v))
			}
			if v, ok := strings.CutPrefix(line, "execution_count="); ok {
				execution, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
	}
	attempt++
	if applied {
		execution++
	}
	content := fmt.Sprintf("attempt_count=%d\nexecution_count=%d\n", attempt, execution)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		return 0, 0, err
	}
	return attempt, execution, nil
}

func appendLog(wsRoot, zipBase string, r Report) error {
	logPath := filepath.Join(wsRoot, "logs", zipBase+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	entry := fmt.Sprintf("=== run at %s ===\n%s\n\n", time.Now().UTC().Format("2006-01-02T15:04:05Z"), r.String())
	_, err = f.WriteString(entry)
	return err
}
