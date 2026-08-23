package receipt

import (
	"testing"

	"github.com/inscope-labs/next-step/engine/internal/registry"
)

func TestGenerate_RequiresClaimedWorkspace(t *testing.T) {
	home := t.TempDir()
	if _, err := Generate(home, "not-a-real-workspace", "task-1", "sha256:abc", "plan-1", "scope text"); err == nil {
		t.Error("Generate against an unclaimed workspace returned no error")
	}
}

func TestGenerate_RequiresAllFields(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "receipt-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	cases := []struct {
		name                                               string
		taskID, taskContentHash, parentActionPlanID, scope string
	}{
		{"missing taskID", "", "sha256:abc", "plan-1", "scope"},
		{"missing hash", "task-1", "", "plan-1", "scope"},
		{"missing plan", "task-1", "sha256:abc", "", "scope"},
		{"missing scope", "task-1", "sha256:abc", "plan-1", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := Generate(home, info.ID, c.taskID, c.taskContentHash, c.parentActionPlanID, c.scope); err == nil {
				t.Errorf("Generate with %s: expected error, got nil", c.name)
			}
		})
	}
}

func TestGenerate_Basic(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "receipt-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	r, err := Generate(home, info.ID, "task-1", "sha256:abc", "plan-1", "write files/x.txt only")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if r.ReceiptID == "" {
		t.Error("ReceiptID is empty")
	}
	if r.Status != StatusPendingReview {
		t.Errorf("Status = %q, want %q", r.Status, StatusPendingReview)
	}
	if r.WorkspaceID != info.ID {
		t.Errorf("WorkspaceID = %q, want %q", r.WorkspaceID, info.ID)
	}
	if r.TaskID != "task-1" || r.TaskContentHash != "sha256:abc" || r.ParentActionPlanID != "plan-1" || r.Scope != "write files/x.txt only" {
		t.Errorf("Generate did not preserve all input fields: %+v", r)
	}
	if r.CreatedAt == "" {
		t.Error("CreatedAt is empty")
	}
	if r.ReviewedBy != "" || r.ReviewedAt != "" {
		t.Errorf("a freshly generated receipt should have no reviewer set, got ReviewedBy=%q ReviewedAt=%q", r.ReviewedBy, r.ReviewedAt)
	}
}

func TestGenerate_DistinctReceiptIDs(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "receipt-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}

	r1, err := Generate(home, info.ID, "task-1", "sha256:abc", "plan-1", "scope")
	if err != nil {
		t.Fatalf("Generate (1): %v", err)
	}
	r2, err := Generate(home, info.ID, "task-2", "sha256:def", "plan-1", "scope")
	if err != nil {
		t.Fatalf("Generate (2): %v", err)
	}
	if r1.ReceiptID == r2.ReceiptID {
		t.Error("two receipts got the same ReceiptID")
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	home := t.TempDir()
	info, err := registry.Claim(home, "receipt-ws", "testing", "tester")
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	generated, err := Generate(home, info.ID, "task-1", "sha256:abc", "plan-1", "scope text")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	loaded, err := Load(home, info.ID, generated.ReceiptID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded != generated {
		t.Errorf("Load returned %+v, want %+v", loaded, generated)
	}
}

func TestLoad_MissingReceipt(t *testing.T) {
	home := t.TempDir()
	if _, err := Load(home, "any-workspace", "no-such-receipt"); err == nil {
		t.Error("Load on a nonexistent receipt returned no error")
	}
}
