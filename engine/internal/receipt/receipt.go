// Package receipt implements receipt generation and action-plan linkage.
//
// Enforces the locked task/receipt/action-plan object model (new at Next
// Step v1.0, no ABX-STEP predecessor): a receipt links exactly one zipped
// task to its declared scope and its parent action plan. The submitting
// agent sees only the receipt, never the action plan — this package is
// what generates that receipt; nothing in this codebase hands an agent the
// action plan document itself.
//
// v1.0 scope note: this package generates and persists receipts
// conforming to root/protocol/v1.0/schemas/receipt.schema.json. It does
// not implement the staging pipeline's human scope-compliance review
// (status transitions out of PENDING_REVIEW) — that reviewer-side
// workflow isn't specified yet beyond the schema's status enum, and isn't
// invented here.
package receipt

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

// Status mirrors receipt.schema.json's status enum.
type Status string

const (
	StatusPendingReview Status = "PENDING_REVIEW"
	StatusScopeApproved Status = "SCOPE_APPROVED"
	StatusScopeRejected Status = "SCOPE_REJECTED"
)

// Receipt mirrors root/protocol/v1.0/schemas/receipt.schema.json field for
// field.
type Receipt struct {
	ReceiptID          string `json:"receipt_id"`
	TaskID             string `json:"task_id"`
	TaskContentHash    string `json:"task_content_hash"`
	WorkspaceID        string `json:"workspace_id"`
	ParentActionPlanID string `json:"parent_action_plan_id"`
	Scope              string `json:"scope"`
	CreatedAt          string `json:"created_at"`
	Status             Status `json:"status"`
	ReviewedBy         string `json:"reviewed_by,omitempty"`
	ReviewedAt         string `json:"reviewed_at,omitempty"`
}

func newReceiptID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating receipt id: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// Generate creates a new receipt in PENDING_REVIEW status, linking a
// single task (by ID and content hash) to its parent action plan and
// declared scope, and persists it under workspace/<ID>/receipts/.
func Generate(home, workspaceID, taskID, taskContentHash, parentActionPlanID, scope string) (Receipt, error) {
	if !registry.Exists(home, workspaceID) {
		return Receipt{}, fmt.Errorf("workspace %s is not claimed", workspaceID)
	}
	if taskID == "" || taskContentHash == "" || parentActionPlanID == "" || scope == "" {
		return Receipt{}, fmt.Errorf("task_id, task_content_hash, parent_action_plan_id, and scope are all required")
	}

	id, err := newReceiptID()
	if err != nil {
		return Receipt{}, err
	}

	r := Receipt{
		ReceiptID:          id,
		TaskID:             taskID,
		TaskContentHash:    taskContentHash,
		WorkspaceID:        workspaceID,
		ParentActionPlanID: parentActionPlanID,
		Scope:              scope,
		CreatedAt:          time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		Status:             StatusPendingReview,
	}

	if err := save(home, workspaceID, r); err != nil {
		return Receipt{}, err
	}
	return r, nil
}

func receiptPath(home, workspaceID, receiptID string) string {
	return filepath.Join(registry.Root(home, workspaceID), "receipts", receiptID+".json")
}

func save(home, workspaceID string, r Receipt) error {
	p := receiptPath(home, workspaceID, r.ReceiptID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("preparing receipts dir: %w", err)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding receipt: %w", err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return fmt.Errorf("writing receipt: %w", err)
	}
	return nil
}

// Load reads a previously generated receipt.
func Load(home, workspaceID, receiptID string) (Receipt, error) {
	b, err := os.ReadFile(receiptPath(home, workspaceID, receiptID))
	if err != nil {
		return Receipt{}, fmt.Errorf("reading receipt: %w", err)
	}
	var r Receipt
	if err := json.Unmarshal(b, &r); err != nil {
		return Receipt{}, fmt.Errorf("decoding receipt: %w", err)
	}
	return r, nil
}
