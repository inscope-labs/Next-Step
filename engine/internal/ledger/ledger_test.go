package ledger

import (
	"testing"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

func TestCommit_RequiresClaimedWorkspace(t *testing.T) {
	home := t.TempDir()
	if _, err := Commit(home, "not-a-real-workspace", "task-1", "sha256:abc", StateApplied, VerificationPass, true); err == nil {
		t.Error("Commit against an unclaimed workspace returned no error")
	}
}

func TestCommit_FirstAttempt(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "ledger-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	e, err := Commit(home, info.ID, "task-1", "sha256:abc", StateApplied, VerificationPass, true)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if e.AttemptCount != 1 {
		t.Errorf("AttemptCount = %d, want 1", e.AttemptCount)
	}
	if e.ExecutionCount != 1 {
		t.Errorf("ExecutionCount = %d, want 1 (applied=true)", e.ExecutionCount)
	}
	if e.FirstAttempt == "" || e.FirstAttempt != e.LastAttempt {
		t.Errorf("FirstAttempt (%q) should equal LastAttempt (%q) on the first commit", e.FirstAttempt, e.LastAttempt)
	}
	if e.LastFinalStatus != "SUCCESS" {
		t.Errorf("LastFinalStatus = %q, want SUCCESS for a PASS verification", e.LastFinalStatus)
	}
	if e.EntryHash == "" {
		t.Error("EntryHash is empty")
	}
}

func TestCommit_NoopRerunDoesNotIncrementExecutionCount(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "ledger-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	first, err := Commit(home, info.ID, "task-1", "sha256:abc", StateApplied, VerificationPass, true)
	if err != nil {
		t.Fatalf("Commit (first): %v", err)
	}

	second, err := Commit(home, info.ID, "task-1", "sha256:abc", StateNoop, VerificationPass, false)
	if err != nil {
		t.Fatalf("Commit (second, noop): %v", err)
	}

	if second.AttemptCount != first.AttemptCount+1 {
		t.Errorf("AttemptCount did not increment on rerun: first=%d second=%d", first.AttemptCount, second.AttemptCount)
	}
	if second.ExecutionCount != first.ExecutionCount {
		t.Errorf("ExecutionCount changed on a NOOP rerun (applied=false): first=%d second=%d", first.ExecutionCount, second.ExecutionCount)
	}
	if second.FirstAttempt != first.FirstAttempt {
		t.Errorf("FirstAttempt changed across commits: first=%q second=%q", first.FirstAttempt, second.FirstAttempt)
	}
}

func TestCommit_FailedVerification(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "ledger-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	e, err := Commit(home, info.ID, "task-1", "sha256:abc", StateUnknown, VerificationFail, false)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if e.LastFinalStatus != "FAILURE" {
		t.Errorf("LastFinalStatus = %q, want FAILURE for a FAIL verification", e.LastFinalStatus)
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "ledger-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	committed, err := Commit(home, info.ID, "task-1", "sha256:abc", StateApplied, VerificationPass, true)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	loaded, err := Load(home, info.ID, "task-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != committed {
		t.Errorf("Load returned %+v, want %+v", loaded, committed)
	}
}

func TestLoad_MissingEntry(t *testing.T) {
	home := t.TempDir()
	if _, err := Load(home, "any-workspace", "no-such-task"); err == nil {
		t.Error("Load on a nonexistent entry returned no error")
	}
}
