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
- `root/protocol/`, `root/workspaces/`, `root/sessions/`, `root/bin/` map
  directly to their installed counterparts.
- Runtime-generated content (live workspace state, `.task-seq`, active session
  state) is git-ignored under `root/workspaces/` and `root/sessions/` but the
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
- `--workspace` is a mandatory flag everywhere a workspace-scoped operation
  occurs — there is no default/global fallback. The old silent `00000`
  workspace fallback is retired permanently; an unspecified workspace is a
  hard `REJECTED`, not a default.
- All paths workspace-scoped from day one — this was a bug class in the old
  implementation (hardcoded `/tmp/` paths, later a `build-task.sh` `SRC` path
  fix) and is treated as a first-class constraint in the Go rewrite rather
  than a patch applied after the fact.

## 7. Engine package breakdown (`engine/internal/`)

| Package | Responsibility |
|---|---|
| `taskseq` | Sequential task counter, workspace-scoped |
| `registry` | Workspace claim/registry, atomic claim semantics |
| `task` | Task zip build and validation |
| `receipt` | Receipt generation, action-plan linkage (locked model: agent sees receipt only) |
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

These are consistent with how sandboxing was already flagged as a
v1.2.2+-class feature under the old ABX-STEP numbering, and their build-out
is intentionally deferred until live multi-agent onboarding feedback informs
the design.
