# workspace-skeleton

Reference mirror of the substructure `next-step` creates under
`workspace/<ID>/` when a workspace is claimed.

**This directory is documentation only.** It is not read by the engine at
runtime. `engine/internal/registry`'s `createWorkspaceTree` hardcodes the
same seven subdirectory names directly (`registry.go`'s `subdirs` var) and
creates them with `os.MkdirAll` at claim time — it does not copy or template
from this path. If the two ever drift, `registry.go`'s `subdirs` var is
ground truth; update this directory to match, not the other way around.

## Subdirectories

| Dir | Holds |
|---|---|
| `inbox/` | Incoming material not yet built into a task |
| `tasks/` | Built task zips (`task-<NNN>_<TASK_ID>.zip`) awaiting or past submission |
| `approvals/` | Human approval records for `run-task --approve` |
| `receipts/` | Pre-execution receipts (scope + parent action-plan linkage) |
| `logs/` | Execution logs |
| `locks/` | Lock files used by concurrency-sensitive operations |
| `files/` | General workspace-scoped file storage, valid `WRITE_PATHS` target root |

See `root/protocol/current/spec/PROTOCOL-FACTS.md`'s Workspace model section
for the ground-truth description, and `../docs/workspace-lifecycle.md` for
how a workspace moves through claim → active → deletion.
