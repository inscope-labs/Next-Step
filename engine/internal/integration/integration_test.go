// Package integration holds the Phase 10 end-to-end test: the closest
// automated equivalent of the "blind onboarding" validation that gated
// the retired ABX-STEP v1.2.1 promotion, adapted to what a Go test can
// honestly automate.
//
// Scope, stated plainly rather than implied:
//
//   - "Fresh install" is simulated by performing exactly the file
//     operations install/install.sh's step 4 performs (mirror this
//     checkout's root/ tree into a temp $NEXT_STEP_HOME, place a binary at
//     bin/next-step) — not by running install.sh itself. install.sh's own
//     network fetch, asset selection, and checksum verification were
//     exercised manually against a local mock server during Phase 8 (see
//     that commit's message); this test does not re-fetch from a real
//     GitHub Release, because doing so would make CI's pass/fail depend on
//     a release existing (it doesn't yet — see Phase 11) and on outbound
//     network access being available in the test environment.
//   - "Onboard an agent through state-0 through state-3" cannot be
//     literally automated — onboarding is an AI agent reading and
//     following markdown prose interactively (see
//     engine/internal/fsm's package doc), and there is no agent in a `go
//     test` run. What this test verifies instead, as the honest
//     structural proxy: every state the fsm package names resolves (via
//     fsm.SpecPath) to a real, non-empty file in the installed
//     protocol/current/spec/ tree, in the fixed linear order fsm.Next
//     defines, with no gaps. That is: the chain an agent would actually
//     walk is complete and consistent as installed. It does not, and
//     cannot, verify that an agent walking it succeeds.
//   - Claim a workspace, build a task, submit it for approval, run it, and
//     generate a receipt are all real — driven through the actual
//     compiled next-step binary via subprocess, exactly as a human or
//     agent would invoke it, not through direct internal-package calls.
//     This is deliberately black-box (unlike engine/internal/task's
//     existing smoke_test.go, which calls Build/Approve/Run directly) so
//     that a bug in cmd/next-step/main.go's flag wiring — not just in the
//     internal packages — would be caught here.
package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inscope-labs/next-step/engine/internal/fsm"
)

// buildCLI compiles cmd/next-step for the current host and returns the
// path to the resulting binary.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "next-step")
	repoEngineDir, err := filepath.Abs(filepath.Join("..", "..", "..", "engine"))
	if err != nil {
		t.Fatalf("resolving engine dir: %v", err)
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/next-step")
	cmd.Dir = repoEngineDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./cmd/next-step: %v\n%s", err, out)
	}
	return bin
}

// freshInstall performs the same file operations install/install.sh's
// step 4 performs (see package doc for why the network-fetch portion is
// not exercised here) and returns the resulting $NEXT_STEP_HOME.
func freshInstall(t *testing.T, cliBin string) string {
	t.Helper()
	home := t.TempDir()

	repoRootDir, err := filepath.Abs(filepath.Join("..", "..", "..", "root"))
	if err != nil {
		t.Fatalf("resolving root/ dir: %v", err)
	}
	if err := copyTree(repoRootDir, home); err != nil {
		t.Fatalf("mirroring root/ into fresh $NEXT_STEP_HOME: %v", err)
	}

	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", binDir, err)
	}
	installedBin := filepath.Join(binDir, "next-step")
	data, err := os.ReadFile(cliBin)
	if err != nil {
		t.Fatalf("reading built binary: %v", err)
	}
	if err := os.WriteFile(installedBin, data, 0o755); err != nil {
		t.Fatalf("placing binary at %s: %v", installedBin, err)
	}
	return home
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// runCLI runs the installed next-step binary with NEXT_STEP_HOME set, and
// fails the test on a non-zero exit.
func runCLI(t *testing.T, home, bin string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "NEXT_STEP_HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("next-step %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// field extracts the value of a "KEY=value" line from CLI stdout, the
// convention every subcommand in cmd/next-step/main.go uses for
// machine-readable output.
func field(t *testing.T, out, key string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, key+"=") {
			return strings.TrimPrefix(line, key+"=")
		}
	}
	t.Fatalf("no %s= line found in output:\n%s", key, out)
	return ""
}

// TestOnboardingChain_StructurallyComplete walks every state fsm defines,
// in the fixed linear order, and confirms each resolves to a real,
// non-empty spec file in the installed tree. See package doc for what
// this does and does not verify about "onboarding."
func TestOnboardingChain_StructurallyComplete(t *testing.T) {
	cliBin := buildCLI(t)
	home := freshInstall(t, cliBin)
	specDir := filepath.Join(home, "protocol", "current", "spec")

	s := fsm.State0Discovery
	seen := 0
	for {
		if !fsm.IsValid(s) {
			t.Fatalf("state %d is not valid per fsm.IsValid", s)
		}
		rel, err := fsm.SpecPath(s)
		if err != nil {
			t.Fatalf("fsm.SpecPath(%s): %v", fsm.Name(s), err)
		}
		full := filepath.Join(home, "protocol", "current", rel)
		if !strings.HasPrefix(full, specDir) {
			t.Fatalf("SpecPath for %s resolved outside spec/: %s", fsm.Name(s), rel)
		}
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatalf("state %s spec file missing at %s: %v", fsm.Name(s), full, err)
		}
		if len(strings.TrimSpace(string(data))) == 0 {
			t.Errorf("state %s spec file at %s is empty", fsm.Name(s), full)
		}
		seen++

		next, ok := fsm.Next(s)
		if !ok {
			break
		}
		s = next
	}
	if seen != 4 {
		t.Errorf("walked %d states, want 4 (state-0 through state-3)", seen)
	}
}

// TestFullLifecycle_ClaimBuildApproveRunReceipt is the actual v1.0
// go/no-go gate: fresh install -> claim a workspace -> build a task ->
// approve it -> run it -> generate a receipt, driven entirely through the
// compiled next-step binary, matching how v1.2.1 was validated before
// promotion (root/protocol/CHANGELOG-PROTOCOL.md).
func TestFullLifecycle_ClaimBuildApproveRunReceipt(t *testing.T) {
	cliBin := buildCLI(t)
	home := freshInstall(t, cliBin)
	bin := filepath.Join(home, "bin", "next-step")

	// -- claim a workspace --
	out := runCLI(t, home, bin, "create-workspace",
		"--name", "integration-test-ws",
		"--purpose", "Phase 10 go/no-go gate",
		"--creator", "integration-test")
	workspaceID := field(t, out, "WORKSPACE_ID")
	if workspaceID == "" {
		t.Fatal("empty WORKSPACE_ID")
	}

	runCLI(t, home, bin, "session", "set-active", "--workspace", workspaceID)
	activeOut := runCLI(t, home, bin, "session", "show-active")
	if field(t, activeOut, "ACTIVE_WORKSPACE") != workspaceID {
		t.Errorf("session show-active = %q, want %q", activeOut, workspaceID)
	}

	// -- author a task directly in the workspace inbox --
	// There is no CLI subcommand for this: task authoring is an AI
	// agent's job per protocol, not the host CLI's (see
	// root/protocol/v1.0/spec/task-acceptance-criteria.md). Writing the
	// inbox files directly is the correct stand-in, matching
	// engine/internal/task's own smoke_test.go.
	taskID := "22222222-2222-4222-8222-222222222222"
	inboxDir := filepath.Join(home, "workspace", workspaceID, "inbox", "task-"+taskID)
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		t.Fatalf("mkdir inbox: %v", err)
	}
	manifest := strings.Join([]string{
		"PROTOCOL_VERSION=1.0",
		"TASK_ID=" + taskID,
		"TASK_VERSION=1",
		"CREATED_AT=2026-01-01T00:00:00Z",
		"CREATED_BY=AI",
		"WORKSPACE_ID=" + workspaceID,
		"TASK_INTENT=integration test",
		"TASK_CONTENT_HASH=PENDING",
		"WRITE_PATHS=files/integration.txt",
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
	start := "#!/usr/bin/env bash\nset -euo pipefail\nprintf 'ok' > \"$NEXT_STEP_WORKSPACE_ROOT/files/integration.txt\"\necho NEXT_STEP_RESULT=APPLIED\n"
	verify := "#!/usr/bin/env bash\nset -euo pipefail\n[ \"$(cat \"$NEXT_STEP_WORKSPACE_ROOT/files/integration.txt\")\" = ok ] && echo '[PASS]' || { echo '[FAIL]'; exit 1; }\n"
	if err := os.WriteFile(filepath.Join(inboxDir, "start.sh"), []byte(start), 0o755); err != nil {
		t.Fatalf("write start.sh: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inboxDir, "verify.sh"), []byte(verify), 0o755); err != nil {
		t.Fatalf("write verify.sh: %v", err)
	}

	// -- build the task --
	buildOut := runCLI(t, home, bin, "build-task", "--workspace", workspaceID, taskID)
	zipPath := field(t, buildOut, "ZIP")
	contentHash := field(t, buildOut, "TASK_CONTENT_HASH")
	if zipPath == "" || contentHash == "" {
		t.Fatalf("incomplete build-task output: %s", buildOut)
	}

	// -- reject-before-approval, matching smoke_test.go's guarantee --
	rejectCmd := exec.Command(bin, "run-task", zipPath)
	rejectCmd.Env = append(os.Environ(), "NEXT_STEP_HOME="+home)
	if out, err := rejectCmd.CombinedOutput(); err == nil {
		t.Fatalf("run-task before approval: expected failure, got success: %s", out)
	}

	// -- submit for approval, then run --
	runCLI(t, home, bin, "run-task", "--approve", "--approver", "integration-test", zipPath)
	runOut := runCLI(t, home, bin, "run-task", zipPath)
	if !strings.Contains(runOut, "APPLIED") {
		t.Errorf("run-task output does not show APPLIED: %s", runOut)
	}

	// -- generate a receipt --
	planID := "33333333-3333-4333-8333-333333333333"
	receiptOut := runCLI(t, home, bin, "receipt", "generate",
		"--workspace", workspaceID,
		"--task", taskID,
		"--hash", contentHash,
		"--plan", planID,
		"--scope", "write files/integration.txt only")
	receiptID := field(t, receiptOut, "RECEIPT_ID")
	status := field(t, receiptOut, "STATUS")
	if receiptID == "" {
		t.Fatal("empty RECEIPT_ID")
	}
	if status != "PENDING_REVIEW" {
		t.Errorf("receipt STATUS = %q, want PENDING_REVIEW", status)
	}

	receiptFile := filepath.Join(home, "workspace", workspaceID, "receipts", receiptID+".json")
	if _, err := os.Stat(receiptFile); err != nil {
		t.Errorf("receipt file not found on disk at %s: %v", receiptFile, err)
	}

	fmt.Printf("integration test complete: workspace=%s task=%s receipt=%s\n", workspaceID, taskID, receiptID)
}
