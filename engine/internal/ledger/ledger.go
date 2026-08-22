// Package ledger implements the per-task execution-state record that the
// retired ABX-STEP lineage called a "receipt" — an artifact written by
// run-task.sh after every run attempt, keyed on TASK_ID, tracking
// ATTEMPT_COUNT / EXECUTION_COUNT / LAST_EXECUTION_STATE /
// LAST_VERIFICATION / LAST_FINAL_STATUS and a RECEIPT_HASH over that
// state.
//
// Phase 5.5.0.2 resolution: this is genuinely a different artifact from
// engine/internal/receipt's task/receipt/action-plan object (a
// pre-execution scope-authorization document the submitting agent sees
// before a task ever runs). The two share a name in the old lineage by
// coincidence, not by design — one is "proof this task was authorized to
// exist," the other is "a ledger of what actually happened when it ran."
// Keeping them as separate packages/artifacts (rather than merging their
// schemas) avoids conflating a pre-execution authorization concept with a
// post-execution audit-log concept. See docs/architecture-overview.md §9
// for the reconciled model and root/protocol/CHANGELOG-PROTOCOL.md's
// Phase 5.5 addendum for the decision record.
//
// This package formalizes, rather than replaces, what
// engine/internal/task's bumpCounters/appendLog already tracked — it adds
// the single committed-object shape (with a content hash over its own
// state) that PROTOCOL-FACTS.md's EGRESS hook fires "after," matching the
// real v1.2.0 receipt.txt format found in the ABX-STEP backup.
package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

// ExecutionState mirrors the legacy receipt's LAST_EXECUTION_STATE values.
type ExecutionState string

const (
	StateApplied ExecutionState = "APPLIED"
	StateNoop    ExecutionState = "NO-OP"
	StateUnknown ExecutionState = "UNKNOWN"
)

// Verification mirrors the legacy receipt's LAST_VERIFICATION values.
type Verification string

const (
	VerificationPass Verification = "PASS"
	VerificationFail Verification = "FAIL"
)

// Entry is the on-disk execution-ledger record for one task, keyed on
// TaskID. Field names are the Go-idiomatic equivalent of the legacy
// KEY=VALUE receipt file's fields (TASK_ID, TASK_VERSION,
// TASK_CONTENT_HASH, FIRST_ATTEMPT, LAST_ATTEMPT, ATTEMPT_COUNT,
// EXECUTION_COUNT, LAST_EXECUTION_STATE, LAST_VERIFICATION,
// LAST_FINAL_STATUS, RECEIPT_VERSION, RECEIPT_HASH).
type Entry struct {
	TaskID           string         `json:"task_id"`
	TaskContentHash  string         `json:"task_content_hash"`
	FirstAttempt     string         `json:"first_attempt"`
	LastAttempt      string         `json:"last_attempt"`
	AttemptCount     int            `json:"attempt_count"`
	ExecutionCount   int            `json:"execution_count"`
	LastExecution    ExecutionState `json:"last_execution_state"`
	LastVerification Verification   `json:"last_verification"`
	LastFinalStatus  string         `json:"last_final_status"` // SUCCESS | FAILURE
	LedgerVersion    int            `json:"ledger_version"`
	EntryHash        string         `json:"entry_hash"`
}

func entryPath(home, workspaceID, taskID string) string {
	return filepath.Join(registry.Root(home, workspaceID), "receipts", taskID+".ledger.json")
}

// hash computes a stable hash over the entry's substantive fields
// (everything but the hash field itself), matching the legacy
// RECEIPT_HASH's role of binding the record to its own content.
func hash(e Entry) string {
	e.EntryHash = ""
	b, _ := json.Marshal(e)
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

// Commit records one execution attempt for a task, creating a new entry
// on first attempt or updating an existing one, and persists it under
// workspace/<ID>/receipts/<TASK_ID>.ledger.json. This is the artifact the
// EGRESS hook fires after, per PROTOCOL-FACTS.md's "after the receipt is
// committed" — here, "the receipt" resolves to this ledger entry, not
// engine/internal/receipt's pre-execution scope document (see package
// doc).
func Commit(home, workspaceID, taskID, taskContentHash string, execution ExecutionState, verification Verification, applied bool) (Entry, error) {
	if !registry.Exists(home, workspaceID) {
		return Entry{}, fmt.Errorf("workspace %s is not claimed", workspaceID)
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	p := entryPath(home, workspaceID, taskID)

	var e Entry
	if b, err := os.ReadFile(p); err == nil {
		if jsonErr := json.Unmarshal(b, &e); jsonErr != nil {
			return Entry{}, fmt.Errorf("existing ledger entry at %s is corrupt: %w", p, jsonErr)
		}
	} else if !os.IsNotExist(err) {
		return Entry{}, fmt.Errorf("reading ledger entry: %w", err)
	}

	if e.FirstAttempt == "" {
		e.FirstAttempt = now
	}
	e.TaskID = taskID
	e.TaskContentHash = taskContentHash
	e.LastAttempt = now
	e.AttemptCount++
	if applied {
		e.ExecutionCount++
	}
	e.LastExecution = execution
	e.LastVerification = verification
	if verification == VerificationPass {
		e.LastFinalStatus = "SUCCESS"
	} else {
		e.LastFinalStatus = "FAILURE"
	}
	e.LedgerVersion = 1
	e.EntryHash = hash(e)

	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return Entry{}, fmt.Errorf("preparing receipts dir: %w", err)
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return Entry{}, fmt.Errorf("encoding ledger entry: %w", err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return Entry{}, fmt.Errorf("writing ledger entry: %w", err)
	}
	return e, nil
}

// Load reads a task's committed ledger entry.
func Load(home, workspaceID, taskID string) (Entry, error) {
	b, err := os.ReadFile(entryPath(home, workspaceID, taskID))
	if err != nil {
		return Entry{}, fmt.Errorf("reading ledger entry: %w", err)
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		return Entry{}, fmt.Errorf("decoding ledger entry: %w", err)
	}
	return e, nil
}
