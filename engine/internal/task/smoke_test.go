package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inscope-labs/next-step/engine/internal/ledger"
	"github.com/inscope-labs/next-step/engine/internal/registry"
)

const smokeStart = `#!/usr/bin/env bash
set -euo pipefail
TARGET="$NEXT_STEP_WORKSPACE_ROOT/files/smoke.txt"
if [ -f "$TARGET" ] && [ "$(cat "$TARGET")" = "hello" ]; then
  echo "[start] already satisfied"
  echo "NEXT_STEP_RESULT=NOOP"
  exit 0
fi
printf '%s' "hello" > "$TARGET"
echo "[start] wrote $TARGET"
echo "NEXT_STEP_RESULT=APPLIED"
`

const smokeVerify = `#!/usr/bin/env bash
set -euo pipefail
TARGET="$NEXT_STEP_WORKSPACE_ROOT/files/smoke.txt"
[ -f "$TARGET" ] || { echo "[FAIL] missing"; exit 1; }
if [ "$(cat "$TARGET")" = "hello" ]; then
  echo "[PASS] content matches"
else
  echo "[FAIL] content mismatch"; exit 1
fi
`

// TestSmoke_ClaimBuildShowApproveRun_NoHooksInstalled is the Phase 5
// smoke-test sequence (claim -> build -> show -> reject-before-approval
// -> approve -> run -> idempotent rerun), extended per Phase 5.5.2.8 to
// confirm zero behavior change with no hook files present — every one of
// the four hook points must be a true no-op here.
func TestSmoke_ClaimBuildShowApproveRun_NoHooksInstalled(t *testing.T) {
	home := t.TempDir()

	info, err := registry.Claim(home, "smoke-ws", "smoke test", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	taskID := "11111111-1111-4111-8111-111111111111"
	inboxDir := filepath.Join(registry.Root(home, info.ID), "inbox", "task-"+taskID)
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	manifest := strings.Join([]string{
		"PROTOCOL_VERSION=1.0",
		"TASK_ID=" + taskID,
		"TASK_VERSION=1",
		"CREATED_AT=2026-01-01T00:00:00Z",
		"CREATED_BY=AI",
		"WORKSPACE_ID=" + info.ID,
		"TASK_INTENT=smoke test",
		"TASK_CONTENT_HASH=PENDING",
		"WRITE_PATHS=files/smoke.txt",
		"IDEMPOTENCY_MODE=SAFE",
		"EXECUTION_CONTEXT=ANDROID_TERMUX_HOME",
		"ALLOW_NETWORK=false",
		"ALLOW_DESTRUCTIVE=false",
		"ALLOW_REMOTE_EXEC=false",
		"ALLOW_PRIVILEGED=false",
		"ALLOW_CREDENTIAL_ACCESS=false",
		"ALLOW_PROCESS_CONTROL=false",
		"REQUIRE_CLEAN_WORKING_DIR=false",
		"INTERRUPTION_POLICY=SAFE_TO_INTERRUPT_ANY_POINT",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(inboxDir, "task.manifest"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "start.sh"), []byte(smokeStart), 0o755); err != nil {
		t.Fatalf("write start.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "verify.sh"), []byte(smokeVerify), 0o755); err != nil {
		t.Fatalf("write verify.sh: %v", err)
	}

	built, err := Build(home, info.ID, taskID)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if _, err := Show(built.ZipPath); err != nil {
		t.Fatalf("Show: %v", err)
	}

	// reject-before-approval: Run must refuse before any approval exists.
	if _, err := Run(home, info.ID, built.ZipPath); err == nil {
		t.Fatal("Run before Approve: expected rejection, got nil")
	}

	if err := Approve(home, info.ID, built.ZipPath, "tester"); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	report, err := Run(home, info.ID, built.ZipPath)
	if err != nil {
		t.Fatalf("Run (first, expect APPLIED): %v", err)
	}
	if report.Execution != "APPLIED" {
		t.Errorf("first run Execution = %q, want APPLIED", report.Execution)
	}
	if report.Verification != "PASS" {
		t.Errorf("first run Verification = %q, want PASS", report.Verification)
	}
	// True no-op hook check: no hook files exist anywhere in this test's
	// $NEXT_STEP_HOME or workspace root, so nothing in the log tail
	// should mention a hook running or blocking.
	for _, marker := range []string{"INGRESS", "PRE_EXECUTE", "POST_EXECUTE", "EGRESS", "hook"} {
		if strings.Contains(report.LogTail, marker) {
			t.Errorf("log tail unexpectedly mentions %q with no hooks installed (no-op behavior broken): %s", marker, report.LogTail)
		}
	}

	// idempotent rerun: second run must report NOOP + still PASS.
	report2, err := Run(home, info.ID, built.ZipPath)
	if err != nil {
		t.Fatalf("Run (second, expect NOOP): %v", err)
	}
	if report2.Execution != "NOOP" {
		t.Errorf("second run Execution = %q, want NOOP", report2.Execution)
	}
	if report2.AttemptCount != report.AttemptCount+1 {
		t.Errorf("AttemptCount did not increment across reruns: first=%d second=%d", report.AttemptCount, report2.AttemptCount)
	}
	if report2.ExecutionCount != report.ExecutionCount {
		t.Errorf("ExecutionCount changed on a NOOP rerun: first=%d second=%d", report.ExecutionCount, report2.ExecutionCount)
	}

	// The execution ledger (what EGRESS fires after) must reflect both
	// attempts.
	entry, err := ledger.Load(home, info.ID, taskID)
	if err != nil {
		t.Fatalf("ledger.Load: %v", err)
	}
	if entry.AttemptCount != 2 {
		t.Errorf("ledger AttemptCount = %d, want 2", entry.AttemptCount)
	}
	if entry.ExecutionCount != 1 {
		t.Errorf("ledger ExecutionCount = %d, want 1", entry.ExecutionCount)
	}
	if entry.LastExecution != ledger.StateNoop {
		t.Errorf("ledger LastExecution = %q, want NO-OP", entry.LastExecution)
	}
	if entry.LastFinalStatus != "SUCCESS" {
		t.Errorf("ledger LastFinalStatus = %q, want SUCCESS", entry.LastFinalStatus)
	}
}
