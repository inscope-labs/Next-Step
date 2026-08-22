// Package taskseq implements the sequential task counter.
//
// Ground truth per root/protocol/v1.0/spec/PROTOCOL-FACTS.md's "Task
// package format" section: a SINGLE SHARED counter at
// $NEXT_STEP_HOME/.task-seq, not a per-workspace counter — "Shared by both
// build paths (build-task.sh and chat-delivery) so numbers never collide."
// An earlier draft of this package was workspace-scoped, based on a
// looser reading of the build plan's "carry forward the workspace-scoped
// path model" instruction (which was actually about avoiding hardcoded
// /tmp/ paths generally, not about per-workspace counter scope
// specifically) — corrected here against the real source once found.
// <NNN> is documented as "human-facing convenience only"; TASK_ID (a full
// UUID) is the real identity, so a global counter does not create any
// correctness risk even under the collision it's explicitly designed to
// avoid.
//
// Locking reuses the same mkdir-atomicity primitive as the registry
// package's workspace claim — no new primitive introduced, consistent
// with PROTOCOL-FACTS.md's stated approach ("same guarantee locks/
// already relies on").
package taskseq

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	seqFilename  = ".task-seq"
	lockFilename = ".task-seq.lock"
	lockTimeout  = 5 * time.Second
	lockRetry    = 20 * time.Millisecond
)

// Next atomically increments and returns the next sequential task number,
// shared across the entire installation at $NEXT_STEP_HOME (home). Numbers
// start at 1.
func Next(home string) (int, error) {
	if err := os.MkdirAll(home, 0o755); err != nil {
		return 0, fmt.Errorf("preparing home dir: %w", err)
	}
	lockPath := filepath.Join(home, lockFilename)

	if err := acquireLock(lockPath); err != nil {
		return 0, err
	}
	defer os.Remove(lockPath)

	seqPath := filepath.Join(home, seqFilename)
	current := 0
	if b, err := os.ReadFile(seqPath); err == nil {
		trimmed := strings.TrimSpace(string(b))
		if trimmed != "" {
			n, err := strconv.Atoi(trimmed)
			if err != nil {
				return 0, fmt.Errorf("%s is corrupt (not an integer): %q", seqFilename, trimmed)
			}
			current = n
		}
	} else if !os.IsNotExist(err) {
		return 0, fmt.Errorf("reading %s: %w", seqFilename, err)
	}

	next := current + 1
	if err := os.WriteFile(seqPath, []byte(strconv.Itoa(next)+"\n"), 0o644); err != nil {
		return 0, fmt.Errorf("writing %s: %w", seqFilename, err)
	}
	return next, nil
}

// acquireLock is a spin-with-backoff mkdir-based lock, matching the
// documented "failed mkdir means try again" pattern used elsewhere in the
// protocol. It is deliberately simple: this is a single-host, human-paced
// protocol (one task built/approved/run at a time in practice for v1.0),
// not a high-contention concurrent system — that's explicitly future work
// (the v1.4.0-class concurrent multi-agent model PROTOCOL-FACTS.md flags
// as not yet designed).
func acquireLock(lockPath string) error {
	deadline := time.Now().Add(lockTimeout)
	for {
		err := os.Mkdir(lockPath, 0o755)
		if err == nil {
			return nil
		}
		if !os.IsExist(err) {
			return fmt.Errorf("acquiring taskseq lock: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("acquiring taskseq lock: timed out after %s (stale lock at %s?)", lockTimeout, lockPath)
		}
		time.Sleep(lockRetry)
	}
}
