package task

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/inscope-labs/next-step/engine/internal/registry"
	"github.com/inscope-labs/next-step/engine/internal/taskseq"
)

// requiredInboxFiles are the three files a staged task directory must
// contain before it can be built, per demo/README.md's target layout.
var requiredInboxFiles = []string{"task.manifest", "start.sh", "verify.sh"}

// BuildResult is what a successful Build produces.
type BuildResult struct {
	ZipPath     string
	SeqNumber   int
	ContentHash string
	WorkspaceID string
	TaskID      string
}

// Build replicates build-task.sh: validates a staged
// workspace/<ID>/inbox/task-<TaskID>/ directory, computes and injects the
// real TASK_CONTENT_HASH (replacing the PENDING placeholder), assigns the
// next sequential number via taskseq, and zips the result into
// workspace/<ID>/tasks/task-<NNN>_<TaskID>.zip.
func Build(home, workspaceID, taskID string) (BuildResult, error) {
	if !registry.Exists(home, workspaceID) {
		return BuildResult{}, fmt.Errorf("workspace %s is not claimed", workspaceID)
	}
	wsRoot := registry.Root(home, workspaceID)
	inboxDir := filepath.Join(wsRoot, "inbox", "task-"+taskID)

	for _, name := range requiredInboxFiles {
		p := filepath.Join(inboxDir, name)
		if _, err := os.Stat(p); err != nil {
			return BuildResult{}, fmt.Errorf("staged task is missing %s (expected at %s): %w", name, p, err)
		}
	}

	manifestPath := filepath.Join(inboxDir, "task.manifest")
	startPath := filepath.Join(inboxDir, "start.sh")
	verifyPath := filepath.Join(inboxDir, "verify.sh")

	// WORKSPACE_NAME is looked up and injected here, before full manifest
	// validation — it is legitimately absent from the hand-filled staged
	// manifest (never hand-filled, per demo/README.md), so validating the
	// full required-field set before this injection would incorrectly
	// reject every correctly-staged task.
	realName, err := lookupWorkspaceName(home, workspaceID)
	if err != nil {
		return BuildResult{}, err
	}
	if err := setManifestField(manifestPath, "WORKSPACE_NAME", realName); err != nil {
		return BuildResult{}, fmt.Errorf("injecting WORKSPACE_NAME: %w", err)
	}

	m, err := ParseManifest(manifestPath)
	if err != nil {
		return BuildResult{}, fmt.Errorf("invalid manifest: %w", err)
	}
	if m.TaskID != taskID {
		return BuildResult{}, fmt.Errorf("manifest TASK_ID %q does not match the requested task %q", m.TaskID, taskID)
	}
	if m.WorkspaceID != workspaceID {
		return BuildResult{}, fmt.Errorf("manifest WORKSPACE_ID %q does not match --workspace %q", m.WorkspaceID, workspaceID)
	}
	if err := ValidateWritePaths(m.WritePaths); err != nil {
		return BuildResult{}, fmt.Errorf("WRITE_PATHS validation failed: %w", err)
	}

	hash, err := contentHash(manifestPath, startPath, verifyPath)
	if err != nil {
		return BuildResult{}, err
	}
	if err := setManifestField(manifestPath, "TASK_CONTENT_HASH", hash); err != nil {
		return BuildResult{}, fmt.Errorf("injecting TASK_CONTENT_HASH: %w", err)
	}

	seq, err := taskseq.Next(home)
	if err != nil {
		return BuildResult{}, fmt.Errorf("assigning sequence number: %w", err)
	}

	zipName := fmt.Sprintf("task-%03d_%s.zip", seq, taskID)
	tasksDir := filepath.Join(wsRoot, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		return BuildResult{}, fmt.Errorf("preparing tasks dir: %w", err)
	}
	zipPath := filepath.Join(tasksDir, zipName)

	if err := zipDir(inboxDir, zipPath); err != nil {
		return BuildResult{}, fmt.Errorf("building zip: %w", err)
	}

	return BuildResult{
		ZipPath:     zipPath,
		SeqNumber:   seq,
		ContentHash: hash,
		WorkspaceID: workspaceID,
		TaskID:      taskID,
	}, nil
}

// setManifestField rewrites a single KEY=VALUE line in place, preserving
// every other line exactly. Used for the two build-time injections
// (WORKSPACE_NAME, TASK_CONTENT_HASH) that are never hand-filled.
func setManifestField(path, key, value string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := splitLinesKeepEmpty(string(b))
	found := false
	for i, line := range lines {
		if hasKey(line, key) {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, key+"="+value)
	}
	return os.WriteFile(path, []byte(joinLines(lines)), 0o644)
}

func hasKey(line, key string) bool {
	return len(line) > len(key) && line[:len(key)] == key && line[len(key)] == '='
}

func splitLinesKeepEmpty(s string) []string {
	var lines []string
	start := 0
	for i, c := range s {
		if c == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// zipDir zips the contents of srcDir (task.manifest, start.sh, verify.sh —
// flat, no subdirectories expected per the documented layout) into
// destZip.
func zipDir(srcDir, destZip string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	f, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, e := range entries {
		if e.IsDir() {
			continue // flat structure only, per documented layout
		}
		srcPath := filepath.Join(srcDir, e.Name())
		if err := addFileToZip(zw, srcPath, e.Name()); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, srcPath, nameInZip string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	w, err := zw.Create(nameInZip)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, src)
	return err
}
