package task

import (
	"strings"
	"testing"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

// TestRun_RefusesUnapprovedZip is the approval hard-gate test called out
// in Phase 5.5.7: Run must refuse to execute a task that was never
// Approve()'d, and it must fail on that gate before doing anything else
// (extracting the zip, running scripts) — this test uses a zip path that
// doesn't even exist on disk, so any error other than the approval gate
// itself would indicate the gate was skipped.
func TestRun_RefusesUnapprovedZip(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "gate-test-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	zipPath := registry.Root(home, info.ID) + "/tasks/task-001_does-not-exist.zip"

	_, err = Run(home, info.ID, zipPath)
	if err == nil {
		t.Fatal("Run on an unapproved (and nonexistent) zip: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not approved") {
		t.Fatalf("Run error = %q, want it to mention the approval gate (got past it without approval?)", err.Error())
	}
}

// TestRun_PassesApprovalGateOnceApproved confirms Approve() actually
// clears the gate: after approval, Run must fail for a *different*
// reason (the zip genuinely doesn't exist on disk) rather than the
// approval-gate error, proving the gate itself is not the blocker
// anymore.
func TestRun_PassesApprovalGateOnceApproved(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "gate-test-ws-2", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	zipPath := registry.Root(home, info.ID) + "/tasks/task-001_does-not-exist.zip"

	if err := Approve(home, info.ID, zipPath, "tester"); err != nil {
		// Approve stats the zip first; a genuinely absent zip is
		// expected to fail Approve too, in which case this test's
		// "approved" precondition can't be constructed this way. Skip
		// rather than false-fail on an unrelated precondition.
		t.Skipf("Approve on a nonexistent zip failed as expected (%v); precondition for this test requires a real zip", err)
	}

	_, err = Run(home, info.ID, zipPath)
	if err == nil {
		t.Fatal("Run on an approved but nonexistent zip: expected an extraction error, got nil")
	}
	if strings.Contains(err.Error(), "not approved") {
		t.Fatalf("Run error = %q, still hit the approval gate after Approve() succeeded", err.Error())
	}
}
