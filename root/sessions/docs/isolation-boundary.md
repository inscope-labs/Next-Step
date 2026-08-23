# Isolation boundary

The rule, stated exactly once, in one place, so every other document can
point here instead of restating it:

> **Task execution must never touch the true host environment. Onboarding
> is exempt from that rule.**

This is not enforced by anything in `root/sessions/` today. As of v1.0,
`engine/internal/sandbox` and `cmd/next-step-runner` are structural stubs
(see `engine/internal/sandbox/doc.go`), and the files under
`root/sessions/runtime/` (`Dockerfile.scratch`, `sandbox-profile.json`) are
non-functional placeholders marking where that enforcement will eventually
live. Stating the rule here, ahead of the enforcement existing, is
deliberate: it gives the eventual implementation a single normative
sentence to satisfy rather than leaving isolation intent implicit in the
sandbox package's absence.

## Why onboarding is exempt

Covered in full in `docs/security-model.md` §1 — summarized here only
because this file's job is to be the one-line pointer, not to duplicate the
reasoning: onboarding is how a human establishes that an agent is
authorized to act at all, and there's no way to establish that trust from
inside a sandbox that, by design, has no path back to a human-trusted
context. Execution is what an already-onboarded agent's *unreviewed output*
does — a different question, and this document is only about that one.

## What "never touch the true host environment" means, concretely

Once `next-step-runner` is real, this boundary is expected to mean at
minimum:

- No filesystem access outside the ephemeral session's own scope — not
  read, not write, regardless of what a task manifest's `WRITE_PATHS` or
  `LINKS` fields declare (those remain lexical, trusted-contract checks at
  the `next-step` / manifest-validation layer per
  `root/protocol/current/spec/PROTOCOL-FACTS.md`; they are not, and were
  never intended to be, the enforcement mechanism for *this* boundary).
- No network access back to the host device's own network context. The
  Oracle Always Free VM's MCP relay is a separate, already-isolated
  forwarding path (`docs/security-model.md` §3) and is not a way for a
  sandboxed session to reach the host either.
- No process-level visibility into `next-step`'s own state (registry,
  session/active pointer, credentials, or anything else host-trusted).

## What this document is not

It is not a threat model (see `docs/security-model.md`), not an
implementation spec (that lands in `engine/internal/sandbox` once it stops
being a stub), and not a claim that isolation is live today. Until
`engine/internal/sandbox` has real logic, treat unreviewed task execution
as **not isolated from the host**, full stop — that caveat is carried
verbatim from `engine/internal/sandbox/doc.go` and repeated here because
this is the file most likely to be read in isolation from the rest of the
docs tree.
