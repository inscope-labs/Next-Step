package hooks

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFire_TrueNoOpWhenAbsent(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	r := Fire(home, wsRoot, Ingress, nil)
	if r.Ran {
		t.Fatalf("Fire with no hook installed: Ran = true, want false (true no-op)")
	}
	if r.Err != nil {
		t.Fatalf("Fire with no hook installed: Err = %v, want nil", r.Err)
	}
}

func TestFireBlocking_TrueNoOpWhenAbsent(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	if _, err := FireBlocking(home, wsRoot, Ingress, nil); err != nil {
		t.Fatalf("FireBlocking with no hook installed: %v, want nil", err)
	}
}

func writeHook(t *testing.T, dir, name, script string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestLookup_WorkspaceRootBeforeHome(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	writeHook(t, filepath.Join(home, "hooks"), "INGRESS", "#!/bin/sh\nexit 0\n")
	writeHook(t, filepath.Join(wsRoot, "hooks"), "INGRESS", "#!/bin/sh\nexit 0\n")

	path, found := Lookup(home, wsRoot, Ingress)
	if !found {
		t.Fatal("Lookup: found = false, want true")
	}
	want := filepath.Join(wsRoot, "hooks", "INGRESS")
	if path != want {
		t.Errorf("Lookup returned %q, want workspace-root hook %q to take precedence over home's", path, want)
	}
}

func TestFireBlocking_GateBlocksOnNonZeroExit(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	writeHook(t, filepath.Join(home, "hooks"), "PRE_EXECUTE", "#!/bin/sh\nexit 1\n")

	if _, err := FireBlocking(home, wsRoot, PreExecute, nil); err == nil {
		t.Fatal("FireBlocking(PRE_EXECUTE) with a failing installed gate: expected an error, got nil")
	}
}

func TestFireBlocking_ObservationalHookNeverBlocks(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	writeHook(t, filepath.Join(home, "hooks"), "EGRESS", "#!/bin/sh\nexit 1\n")

	if _, err := FireBlocking(home, wsRoot, Egress, nil); err != nil {
		t.Fatalf("FireBlocking(EGRESS) with a failing installed hook: got error %v, want nil (observational, non-blocking)", err)
	}
}

func TestFire_ExtraEnvIsPassed(t *testing.T) {
	home := t.TempDir()
	wsRoot := t.TempDir()
	out := filepath.Join(home, "out.txt")
	script := "#!/bin/sh\necho \"$NEXT_STEP_TASK_ID\" > " + out + "\nexit 0\n"
	writeHook(t, filepath.Join(home, "hooks"), "INGRESS", script)

	r := Fire(home, wsRoot, Ingress, map[string]string{"NEXT_STEP_TASK_ID": "abc-123"})
	if !r.Ran {
		t.Fatal("Fire: Ran = false, want true (hook was installed)")
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading hook output file: %v", err)
	}
	if got := string(b); got != "abc-123\n" {
		t.Errorf("hook did not receive NEXT_STEP_TASK_ID via env: got %q", got)
	}
}
