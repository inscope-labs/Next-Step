# Next Step — Architecture Overview

**Status:** v1.0 scaffolding reference.
**Supersedes:** ABX-STEP (retired, last protocol version v1.2.1).

## 1. What Next Step is

Next Step is a human-authorized, hash-bound task execution protocol for AI coding
agents. An agent is onboarded through a fixed state chain, claims a workspace,
builds a task as a zip artifact, and submits it for review. A receipt links that
one task to its scope and to a parent action plan — the agent never sees the
action plan itself, only the receipt.

This document describes the *target* architecture for the rebuild onto Go. It is
the companion reference for `next-step-build-plan.md`.

## 2. Why the rebuild

The prior implementation (ABX-STEP, bash-based, v1.1.0 → v1.2.1) proved the
protocol shape correctly but carried structural debt inherent to a shell
implementation: hardcoded paths, fragile heredoc delivery, `mkdir`-based
registry races handled by convention rather than the runtime, and no clean
separation between "code that runs with the user's full trust" and "code that
executes a task's contents."

Next Step keeps the protocol shape (workspace model, onboarding chain, task →
receipt flow) and rebuilds the engine underneath it in Go, with an explicit
two-binary trust split.

## 3. Two-binary trust split

| Binary | Runs where | Purpose | Trust level |
|---|---|---|---|
| `next-step` | Host (trusted Termux/Linux environment) | Onboarding, workspace claim, task packaging, receipt generation, registry | Full host trust — same trust level as the user invoking it |
| `next-step-runner` | Ephemeral sandboxed session | Executes a claimed task's contents | Zero host trust — must never reach the true host environment |

Onboarding is explicitly exempt from sandboxing: it runs in the trusted host
context because it is the mechanism that *establishes* trust for everything
downstream. Task execution is the boundary that must never touch the host —
see `security-model.md` for the full rationale.

`next-step-runner` is a separate binary, not a mode flag on `next-step`, so
that the install flow can choose not to ship it at all in host-only deployments,
and so the sandboxed execution path can evolve independently (FSM engine,
session isolation) without touching the host-trusted binary's surface area.

## 4. Repo ↔ install mirror

The repository's `root/` directory is a path-for-path mirror of what gets
installed at `$NEXT_STEP_HOME` on a target machine. This means:

- What you see in `root/` in the repo is exactly the payload the installer
  places on disk — no separate "packaging" transform between repo layout and
  install layout.
- `root/protocol/`, `root/workspace/`, `root/sessions/`, `root/bin/` map
  directly to their installed counterparts.
- Runtime-generated content (live workspace state, `.task-seq`, active session
  state) is git-ignored under `root/workspace/` and `root/sessions/` but the
  *directory structure* itself is committed, so a fresh install has the correct
  empty skeleton to write into.

## 5. Versioned protocol directories

Protocol content lives under `protocol/vX.Y.Z/` directories. An atomic
`protocol/current` symlink flip is how a version becomes active — this avoids
any window where a partially-updated protocol version could be read mid-flip.
`root/protocol/CHANGELOG-PROTOCOL.md` tracks protocol-level (not engine-level)
version history, starting at v1.0 as the successor to ABX-STEP protocol v1.2.1.

**Implemented as of v1.0** (this was an open question through Phase 3; resolved
before Phase 4): `root/protocol/v1.0/{spec,schemas}/` holds the actual content,
and `root/protocol/current` is a symlink to `v1.0`. Every cross-reference inside
the onboarding chain (`next-step-onboarding.md`, `state-0.md` through
`state-3.md`, `PROTOCOL-FACTS.md`) resolves through `protocol/current/`, never
a hardcoded version number — so a future version bump only requires adding
`protocol/v1.1.0/` (or similar) and re-pointing the symlink, with no edits
required to the chain's own path references.

## 6. Workspace model

- A workspace claim is atomic (Go-native equivalent of the old `mkdir`-based
  atomic registry, but without relying on filesystem race semantics as the
  sole safety mechanism).
- `--workspace` is accepted everywhere a workspace-scoped operation occurs.
  If omitted, the operation reads `sessions/active` (an explicit,
  human-set default pointer — set via a `session set-active` operation,
  not an implicit/silent choice made by the system). If `--workspace` is
  omitted *and* `sessions/active` doesn't exist or is empty, the operation
  is a hard `REJECTED`, never a silent guess. This is distinct from the old
  `00000` legacy-workspace bug (a hardcoded, un-chosen fallback target),
  which is retired permanently — `00000` (where it exists at all, for
  migrated-from-flat-layout installs) is inert and deletable like any other
  workspace, never an implicit target.
- All paths workspace-scoped from day one — this was a bug class in the old
  implementation (hardcoded `/tmp/` paths, later a `build-task.sh` `SRC` path
  fix) and is treated as a first-class constraint in the Go rewrite rather
  than a patch applied after the fact.
- Ground-truth runtime paths (from `PROTOCOL-FACTS.md`, not this repo's own
  earlier scaffold guess): `workspace/<ID>/` for live claimed instances
  (holding `inbox/tasks/approvals/receipts/logs/locks/files`),
  `workspace/registry/<ID>/` for the claim registry, `sessions/active` for
  the default-workspace pointer. All of this lives under one top-level
  `root/workspace/` (singular) directory, alongside the committed
  `templates/docs/legacy` reference material (see
  `root/workspace/templates/workspace-skeleton/README.md`,
  `root/workspace/docs/workspace-lifecycle.md`,
  `root/workspace/legacy/README.md`). An earlier build-plan draft called
  for a second, separate `root/workspaces/` (plural) directory to hold that
  reference material — it briefly existed in this repo before Phase 6 was
  corrected; there was never a real reason for the split, and it added a
  second top-level dir to keep in sync with the repo↔install mirror in §4
  for no benefit.

## 7. Engine package breakdown (`engine/internal/`)

| Package | Responsibility |
|---|---|
| `taskseq` | Sequential task counter, global (single shared counter at `$NEXT_STEP_HOME/.task-seq`) — not per-workspace, corrected from an earlier draft; see package doc for why |
| `registry` | Workspace claim/registry, atomic claim semantics |
| `task` | Task zip build and validation |
| `receipt` | Pre-execution receipt generation, action-plan linkage (locked model: agent sees receipt only) |
| `ledger` | Added Phase 5.5.4. Post-execution execution-state record per task (attempt/execution counts, last verification, content hash over that state) — the artifact the retired ABX-STEP lineage also called a "receipt." Distinct from `receipt`; see §9 |
| `hooks` | Added Phase 5.5.2. Four-point (`INGRESS`/`PRE_EXECUTE`/`POST_EXECUTE`/`EGRESS`) lookup/exec mechanism, wired into `task.Run`. No bundled scripts — true no-op until a human installs one |
| `fsm` | Onboarding state chain (state-0 → state-3 for v1.0; fuller state-machine engine is a later minor version) |
| `sandbox` | Ephemeral sandboxed execution primitives (stubbed structurally in v1.0, not functional) |

## 8. Onboarding chain

A linear, four-state flow (`state-0.md` → `state-3.md`) that an AI agent walks
through before it can claim a workspace. v1.0 implements this as a fixed linear
flow; a fuller FSM engine (arbitrary state graphs, not just the linear chain)
is planned for a later minor version once live multi-agent onboarding feedback
is available.

## 9. Task → receipt → action plan model (locked)

- **Action plan**: a broad, reusable JSON roadmap document. Contains one or
  more tasks. Not scoped to a single agent or session.
- **Task receipt**: a narrower artifact. Links one zipped task to its scope
  and to its parent action plan.
- **The AI agent only ever sees the receipt** — never the action plan. The
  agent zips the task; `next-step` generates the receipt; a staging pipeline
  with human review validates scope compliance. The submitting agent never
  validates its own scope compliance.
- **This "receipt" is a pre-execution artifact, distinct from the
  execution ledger.** Phase 5.5.0.2/5.5.4 resolved a naming collision
  inherited loosely from the ABX-STEP lineage: that lineage's own
  "receipt" was a *post-execution* execution-state record (attempt/
  execution counts, last verification, a content hash over that state),
  written after every `run-task` attempt — a genuinely different artifact
  from the pre-execution scope-authorization document described above.
  The old concept is carried forward as `engine/internal/ledger`, kept
  deliberately separate rather than merged into `receipt.schema.json`, so
  "receipt" in this document always means the scope/action-plan artifact.
  `PROTOCOL-FACTS.md`'s `EGRESS` hook — documented as firing "after the
  receipt is committed" — resolves to the ledger entry being committed,
  since that is the artifact tied to a specific `run-task` execution; the
  pre-execution receipt is generated earlier in the flow and isn't
  re-committed at `run-task` time.

## 10. Distribution and runtime separation

- **CI** (`ci.yml`, `release.yml`, `codeql.yml`) builds and tests the
  `engine/` Go module and cross-compiles release binaries. CI has no
  dependency on the Oracle Always Free VM.
- **The Oracle Always Free VM** is a runtime operating plane and MCP relay
  (blind WSS forwarder) only — it is never a build environment. This
  separation is intentional: the relay's trust boundary and the CI/release
  pipeline's trust boundary are independent, and collapsing them would let a
  compromise of one become a compromise of the other.
- **Public install** (`install/install.sh`) fetches a release binary (never a
  source build) and verifies its checksum before placing it on disk, so the
  installed binary's provenance is always a signed/checksummed release
  artifact, never a local compile.

## 11. What v1.0 explicitly does not include

Per the build plan's scope note, v1.0 is host-side protocol functionality
only. The following are scaffolded structurally (packages/dirs/stub files
exist) but are not functionally required for v1.0:

- Ephemeral sandboxed task execution (`next-step-runner` real logic)
- The fuller FSM engine (only the linear onboarding chain ships in v1.0)
- Session isolation enforcement
- Real write-time enforcement of `WRITE_PATHS` and `LINKS` (both are
  lexical, trusted-contract checks only — see PROTOCOL-FACTS.md's
  Link/capability caveat)
- Actual hook scripts for any of the four extension points (the
  lookup/exec mechanism is implemented and wired in as of Phase 5.5.2;
  no scripts ship with this repo)
- The "chat-delivery" build path named in PROTOCOL-FACTS.md's Task
  package format section — undocumented beyond that one line, open per
  Phase 5.5.0.1

These are consistent with how sandboxing was already flagged as a
v1.2.2+-class feature under the old ABX-STEP numbering, and their build-out
is intentionally deferred until live multi-agent onboarding feedback informs
the design.
