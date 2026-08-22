package registry

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestClaim_Basic verifies a claim creates the registry entry and the
// full per-workspace subdir tree.
func TestClaim_Basic(t *testing.T) {
	home := t.TempDir()
	info, err := Claim(home, "my-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if !Exists(home, info.ID) {
		t.Fatalf("Exists(%s) = false after Claim", info.ID)
	}
	for _, d := range subdirs {
		p := filepath.Join(Root(home, info.ID), d)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected subdir %s to exist: %v", d, err)
		}
	}
}

// TestClaim_ConcurrentNoCollision drives many concurrent claims and
// asserts every one gets a distinct ID with no lost or duplicate claims —
// this is the atomic mkdir-based claim/collision-retry guarantee
// PROTOCOL-FACTS.md documents ("a failed mkdir means try another ID").
func TestClaim_ConcurrentNoCollision(t *testing.T) {
	home := t.TempDir()
	const n = 25

	var wg sync.WaitGroup
	ids := make([]string, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			info, err := Claim(home, "concurrent-ws", "testing", "tester")
			ids[i] = info.ID
			errs[i] = err
		}(i)
	}
	wg.Wait()

	seen := map[string]bool{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("claim %d failed: %v", i, err)
		}
		if seen[ids[i]] {
			t.Fatalf("duplicate workspace ID claimed: %s", ids[i])
		}
		seen[ids[i]] = true
		if !Exists(home, ids[i]) {
			t.Errorf("claimed ID %s does not report Exists", ids[i])
		}
	}
	if len(seen) != n {
		t.Fatalf("expected %d distinct claims, got %d", n, len(seen))
	}
}

func TestResolve_ExplicitAndActiveAndHardReject(t *testing.T) {
	home := t.TempDir()
	info, err := Claim(home, "ws", "p", "c")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	// No explicit ID, no active session -> hard reject, never a silent
	// guess (the retired 00000 fallback bug must not resurface).
	if _, err := Resolve(home, ""); err != ErrWorkspaceRequired {
		t.Fatalf("Resolve with nothing set: want ErrWorkspaceRequired, got %v", err)
	}

	// Explicit ID works regardless of active session.
	got, err := Resolve(home, info.ID)
	if err != nil || got != info.ID {
		t.Fatalf("Resolve(explicit): got (%q, %v), want (%q, nil)", got, err, info.ID)
	}

	// Explicit unclaimed ID is rejected.
	if _, err := Resolve(home, "not-a-real-id"); err == nil {
		t.Fatalf("Resolve(unclaimed explicit ID): want error, got nil")
	}

	// Active session fallback works once set.
	if err := SetActive(home, info.ID); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	got, err = Resolve(home, "")
	if err != nil || got != info.ID {
		t.Fatalf("Resolve(active fallback): got (%q, %v), want (%q, nil)", got, err, info.ID)
	}
}
