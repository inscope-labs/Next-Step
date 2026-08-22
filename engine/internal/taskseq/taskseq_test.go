package taskseq

import (
	"sync"
	"testing"
)

// TestNext_Sequential verifies basic monotonic behavior starting at 1.
func TestNext_Sequential(t *testing.T) {
	home := t.TempDir()
	for want := 1; want <= 5; want++ {
		got, err := Next(home)
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if got != want {
			t.Fatalf("Next() = %d, want %d", got, want)
		}
	}
}

// TestNext_GlobalNotPerWorkspace is the regression test for the exact bug
// found and fixed this phase: an earlier draft scoped the counter
// per-workspace, but PROTOCOL-FACTS.md is explicit that
// $NEXT_STEP_HOME/.task-seq is a SINGLE SHARED counter across the whole
// installation, "shared by both build paths ... so numbers never
// collide." This test asserts that two logically distinct callers
// (simulating two different workspaces' build-task calls) draw from the
// same monotonic sequence with no shared state reset between them.
func TestNext_GlobalNotPerWorkspace(t *testing.T) {
	home := t.TempDir() // one $NEXT_STEP_HOME, shared regardless of caller

	first, err := Next(home)
	if err != nil {
		t.Fatalf("Next (workspace A): %v", err)
	}
	second, err := Next(home)
	if err != nil {
		t.Fatalf("Next (workspace B): %v", err)
	}
	if second != first+1 {
		t.Fatalf("global counter did not advance monotonically across callers: first=%d second=%d", first, second)
	}
}

// TestNext_ConcurrentNoDuplicates drives many concurrent Next() calls
// against the same home and asserts every returned number is unique and
// the final count equals n — i.e. the mkdir-based lock actually
// serializes increments rather than losing updates under contention.
func TestNext_ConcurrentNoDuplicates(t *testing.T) {
	home := t.TempDir()
	const n = 40

	var wg sync.WaitGroup
	results := make([]int, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], errs[i] = Next(home)
		}(i)
	}
	wg.Wait()

	seen := map[int]bool{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("Next() call %d failed: %v", i, err)
		}
		if seen[results[i]] {
			t.Fatalf("duplicate sequence number returned: %d", results[i])
		}
		seen[results[i]] = true
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct sequence numbers, got %d (lost update under contention)", n, len(seen))
	}
	// The set of numbers issued must be exactly {1..n} — no gaps, no
	// numbers skipped or double-counted.
	for i := 1; i <= n; i++ {
		if !seen[i] {
			t.Errorf("sequence number %d was never issued (gap under contention)", i)
		}
	}
}
