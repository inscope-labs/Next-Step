# PROTOCOL-FACTS.md — Next Step v1.0

Read this in full before acting. This file is the contract. Source files
are for HOW; this file is for WHAT.

## Identity

- `PROTOCOL_VERSION=1.0`
- Home: `$NEXT_STEP_HOME` (default `$HOME/next-step`)
- Next Step succeeds the retired ABX-STEP protocol (last version v1.2.1).
  The manifest fields, workspace model, and authorization model below were
  carried forward from that lineage rather than redesigned from scratch —
  see `docs/architecture-overview.md` for the full rationale.
- Live entry point / versioned protocol directory layout
  (`protocol/current` symlink vs. flat `protocol/spec/`) is an open
  architectural question for this repo — not yet resolved, tracked in
  `docs/architecture-overview.md` §5. This file describes the manifest and
  authorization contract, which hold regardless of how that question
  resolves.

## Task package format

External file: `task-<NNN>_<TASK_ID>.zip`

- `<NNN>` — zero-padded sequence number, drawn from the single shared
  `$NEXT_STEP_HOME/.task-seq` counter, incremented atomically. Shared by
  both build paths (`build-task.sh` and chat-delivery) so numbers never
  collide. Human-facing convenience only.
- `<TASK_ID>` — full UUID. This is the real identity. `TASK_ID` inside
  the manifest, the internal `task-<TASK_ID>/` folder name, receipts,
  approvals, and locks are all keyed on the full UUID, never `<NNN>`.
  `run-task.sh` never validates the outer zip filename against
  anything.
- UUID generation: `/proc/sys/kernel/random/uuid` primary. Falls back
  to a timestamp-based ID only if that path is unavailable.

Manifest mandatory fields (both present from Next Step v1.0; inherited
from legacy ABX-STEP v1.2.0, which added them over ABX-STEP v1.1.0):
- `WORKSPACE_ID`, `WORKSPACE_NAME` — both mandatory from Next Step v1.0
  onward. Historically these were added in legacy ABX-STEP v1.2.0 as a
  minor (not patch) bump, since they changed `TASK_CONTENT_HASH` identity
  for any newly-built task.
- `WRITE_PATHS` — declared relative write paths, or `NONE`.

## Execution entry point

`protocol/current/run-task.sh --show|--approve|--run <zip>`, optionally
with `--workspace <ID>` (if omitted, reads `$NEXT_STEP_HOME/sessions/active`).

`run-task.sh` is a thin orchestrator with this contract header; actual
staging logic lives in `runner-stage.sh`, actual execution logic in
`runner-exec.sh`. Opening `run-task.sh` and this file answers WHAT;
opening the runner-* files is only needed for HOW.

## Verification approach

Functional, not just syntactic. Every file in this protocol is tested
by running real hand-built task packages through the full pipeline
(`build-task.sh` → `--show` → `--approve` → `--run`), because syntax
checks alone (`bash -n`) have already been shown to miss real bugs
(the `LOCK_DIR` scoping regression). Backup-and-verify before any
destructive step; read-only scans before diagnosing anything.

## Authorization model

Unchanged from legacy ABX-STEP v1.1.0 onward, and unchanged in Next
Step v1.0: human-only, hard gate. No task executes without
explicit human `--approve`. This is a fixed-term commitment — the one
place in the whole onboarding chain where exact wording is held
consistent across every state file, not just intent.

## Workspace model

- `workspace/<ID>/` — one directory per workspace, holding its own
  `inbox/tasks/approvals/receipts/logs/locks/files`.
- `workspace/registry/<ID>/` — existence is the claim. Claimed via
  `mkdir` (atomic on POSIX; a failed `mkdir` means "try another ID,"
  same guarantee `locks/` already relies on — no new primitive).
  Holds `name`, `created`, `creator`, `purpose`.
- `workspace/00000/` — legacy workspace. Everything that existed at
  the flat, pre-workspace layout was migrated here. Marked
  non-executable / inert by its own README, but **not exempt** from
  the normal workspace deletion process — it can be deleted like any
  other workspace if the human chooses to.
- `sessions/active` — the current default workspace ID, read by
  `run-task.sh`/`build-task.sh` when `--workspace` is omitted. Set via
  `bin/set-session.sh`.
- A workspace-escape **snapshot-based** detection approach was
  prototyped and rejected: it doesn't scale (`find`+`stat` over the
  whole `workspace/` tree per run), races under the planned v1.4.0
  concurrent multi-agent model, and isn't shell/agent-agnostic. Do not
  reintroduce it.

## Path enforcement rule

- Read access: allowed globally against shared/global assets, e.g.
  `protocol/current/templates/*`.
- Write access: confined to `workspace/<ID>/` root. Any `WRITE_PATHS`
  entry that resolves outside the claiming workspace's root — absolute
  path, `..`, `~` — is rejected at manifest-validation time, before
  `--show` even runs.

## Link / capability caveat — read this before assuming enforcement

Inter-workspace links (read-only / write-once / time-bounded) and
`WRITE_PATHS` are **documented, statically-enforced contracts that the
runner trusts** — the same honesty category as `ALLOW_*` capability
auditing. They are validated by pure lexical string-checking at
manifest-validation time. This is NOT kernel-level or filesystem-level
enforcement, and it does NOT stop a determined script from writing
outside its declared paths at runtime. Real write-time confinement is
explicitly out of scope for Next Step v1.0 (as it was for legacy
ABX-STEP v1.2.0) and remains future work.

## Hook / gate architecture

Four named extension points, all no-op unless the corresponding file
exists and is executable:

| Point | Kind | Fires | Can block? |
|---|---|---|---|
| `INGRESS` | gate | before staging | yes |
| `PRE_EXECUTE` | gate | before `start.sh` runs | yes |
| `POST_EXECUTE` | hook | after `start.sh` runs | no (observational) |
| `EGRESS` | hook | after the receipt is committed | no — receipt is already truth on disk |

Lookup order: `$WORKSPACE_ROOT/hooks/<name>`, then
`$NEXT_STEP_HOME/hooks/<name>`. If neither exists and is executable,
the point is a true no-op: returns 0, writes nothing. Confirmed via
isolated test. When present, hooks are exec'd as a separate process —
never sourced — with context passed via `ABX_*` env vars.

This is the intended extension surface for the v1.4.0 multi-agent /
multi-pipeline concurrent execution model. Do not grow new enforcement
logic into `run-task.sh`, `runner-stage.sh`, or `runner-exec.sh`
themselves — real sandboxed enforcement is future scope, delivered via
system config and these hook points, not core-file logic.

## Report field contract

`EXECUTION`, `IDEMPOTENCY`, `EXECUTION_COUNT`, `ATTEMPT_COUNT` must
agree with `LOG_TAIL` in every case, including edge cases — not just
the common path. (Legacy ABX-STEP v1.1.0 had a live bug here: a run reported
`EXECUTION: NO-OP` / `IDEMPOTENCY: ALREADY_SATISFIED` while its own
`LOG_TAIL` showed the write branch of `start.sh` actually ran.) The fix,
carried forward from legacy ABX-STEP v1.2.0, resolves this by making `start.sh` self-report via a single
`NEXT_STEP_RESULT=APPLIED|NOOP` line on stdout, rather than inferring
execution state from the presence or content of a separate state
file. The runner treats this line as authoritative and does not
reconstruct execution state by parsing `start.sh`'s other output.

## Clipboard convention

All output shown to the human goes through
`"$NEXT_STEP_HOME/bin/next-step-clipboard.sh"` — never
`termux-clipboard-set` directly. `next-step-clipboard.sh` is the only place
in the entire protocol with platform-specific code: one dispatch line.
Everything else in the protocol core is platform-agnostic. Has its own
`PRE_CLIPBOARD` hook (no-op unless installed), fired before that one
platform line.

Multi-file reads concatenate into a single temp file first, one
clipboard write at the end — chaining multiple
`cat ... | tee >(termux-clipboard-set)` calls in one block overwrites
itself and was a real bug caught live (stage_8a).

## File size constraint

12KB hard ceiling per protocol file — rooted in the clipboard-transport
constraint (every file must be pasteable/reviewable via clipboard). If
a file needs more, split at a natural seam; don't cram.

## Explicitly out of scope for Next Step v1.0

`--dry-run`/`--check-verify` mode, report field merging, cleanup-after-
demo prompting. Rated low/medium in prior test analysis; not part of
the locked decisions for this version. Candidates for a future pass —
their absence here is deliberate, not an oversight.
