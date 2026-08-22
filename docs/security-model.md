# Next Step — Security Model

**Status:** v1.0 scaffolding reference.

This document covers the trust and isolation model behind Next Step's
architecture. It is a companion to `architecture-overview.md`, focused
specifically on the "why" behind the trust boundary rather than the package
layout.

## 1. The core boundary: onboarding vs. execution

Next Step draws one hard line: **task execution must never touch the true
host environment. Onboarding is exempt from that rule.**

This looks asymmetric at first, but the asymmetry is the point:

- Onboarding is the mechanism by which a human establishes that a given AI
  agent is authorized to act at all. It runs in the trusted host context
  because there is no meaningful way to *establish* trust from inside an
  untrusted sandbox — the human is present, the state chain is walked
  deliberately, and the outcome (a claimed workspace) is itself the trust
  artifact everything downstream depends on.
- Task execution is what an *already-onboarded* agent does with content it
  produced. That content has not been reviewed yet — the receipt/staging
  pipeline review happens after the task is built, not before. So execution
  of a task's contents runs in a sandbox with zero path back to the host,
  full stop, regardless of how trusted the onboarding step that preceded it
  was.

Put differently: onboarding trust is about *who* is acting. Execution
isolation is about *what* their output is allowed to touch before a human has
reviewed it. These are different questions and get different answers.

## 2. Two-binary split rationale

`next-step` and `next-step-runner` are separate binaries rather than two modes
of one binary, for three reasons:

1. **Blast radius.** A vulnerability in the sandboxed execution path
   (`next-step-runner`) should not automatically imply code-level access to
   the host-trusted registry/receipt/onboarding logic (`next-step`). Separate
   binaries means separate attack surfaces, not just separate code paths
   inside one process.
2. **Independent evolution.** The FSM engine, session isolation model, and
   sandbox internals are all explicitly unfinished / deferred past v1.0. That
   work can proceed on `next-step-runner` without needing to touch, retest, or
   re-release `next-step` itself.
3. **Deployment flexibility.** A host-only install (no local task execution
   ever performed) never needs to fetch or trust `next-step-runner` at all.
   The install flow (`install/install.sh`) only ever fetches `next-step` —
   `next-step-runner` is explicitly excluded from what gets placed at
   `root/bin/` by the public installer.

## 3. Build / runtime / distribution separation

Three environments, kept deliberately independent of one another:

| Environment | Role | Must never |
|---|---|---|
| CI (GitHub Actions) | Builds, tests, cross-compiles, publishes signed release binaries | Depend on the Oracle Always Free VM, or on any live runtime state |
| Oracle Always Free VM | Runtime operating plane; MCP relay (blind WSS forwarder) between remote MCP clients and the host device | Function as a build environment; execute or interpret task contents itself |
| Host device (Termux/Android or Linux) | Runs `next-step`, holds onboarding trust, claims workspaces, submits tasks | Directly expose task execution to the host process/filesystem |

The relay VM is deliberately kept "blind" — it forwards, it does not
interpret. This means a compromise of the relay does not, by itself, grant
the ability to author or approve tasks; it can only intercept or disrupt
traffic already flowing between an already-authorized client and the host.

Whether the relay and the Next Step runtime plane eventually need *additional*
process-level trust isolation from each other (beyond "the relay doesn't
execute anything") is an open question, not yet resolved — noted here rather
than assumed away.

## 4. What "enforcement" means at each layer

- `next-step` **enforces** the protocol: workspace claims are atomic and
  rejected outright (not defaulted) when unspecified; task packaging is
  workspace-scoped throughout; receipt generation is the only path by which
  a task becomes reviewable.
- The **staging pipeline** (human review) enforces scope compliance — this is
  explicitly never the responsibility of the submitting agent. An agent that
  produces a receipt has not thereby validated itself; a human/pipeline step
  downstream does that.
- `next-step-runner` **enforces** isolation: it is the component whose entire
  job is to make sure task execution cannot observe or affect the true host
  environment. In v1.0 this is a structural stub — the isolation guarantee is
  not yet live — so v1.0 deployments must not treat unreviewed task execution
  as safe until the sandbox is functional.

## 5. Known open items (tracked, not resolved)

- Relay ↔ runtime process isolation (see §3) — open.
- Formal sandbox technology choice for `next-step-runner` (container, gVisor,
  seccomp-based, etc.) — deferred until post-v1.0, pending onboarding
  feedback that will inform the FSM/session design it needs to support.
- Whether any future merger of adjacent components (e.g. file-manager-style
  functionality) into a shared execution context would weaken this isolation
  model — treated as a hard constraint against merging trust boundaries,
  not yet a concrete proposal for Next Step itself.
