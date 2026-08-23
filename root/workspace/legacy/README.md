# Legacy workspace migration policy

**Status for this repo's v1.0 releases: dormant.** Next Step v1.0 ships to
fresh installs only — there is no shipped upgrade path from a pre-workspace
flat layout, so nothing in this document is exercised by `install/install.sh`
or by `engine/internal/registry` today. This is policy for *if and when* a
future migration scenario exists, documented now for completeness rather
than left to be reconstructed under pressure later.

## Why this exists at all

The retired ABX-STEP lineage went through exactly this transition once
already — a flat, pre-workspace layout that was later migrated wholesale
into a single reserved workspace. `PROTOCOL-FACTS.md` carries that outcome
forward as a documented ground-truth concept (`workspace/00000/`) even
though v1.0's own install flow never produces it. This document exists so
that if Next Step itself ever needs an analogous migration — a future
protocol version changing the workspace model in a way that pre-existing
installs can't just adopt in place — there's a precedent to follow rather
than a decision to make from scratch under time pressure.

## The policy, if invoked

1. **Reserved ID.** Migrated flat-layout content lands in a single
   workspace with a fixed, reserved ID — `00000` in the precedent this
   follows. A reserved ID (rather than a freshly generated UUID) makes the
   migration outcome deterministic and inspectable ahead of time, and avoids
   colliding with the normal claim path's UUID generation.
2. **Inert, not privileged.** The migrated workspace is marked non-executable
   / inert by its own README at claim time. Inert means: nothing treats it
   as an implicit target (see `../docs/workspace-lifecycle.md` §2 — there is
   no implicit target, ever, migrated or not). It does **not** mean exempt
   from anything else a normal workspace can have done to it.
3. **Not exempt from deletion.** A migrated legacy workspace can be deleted
   through the same manual process as any other workspace (see
   `../docs/workspace-lifecycle.md` §3). Migration status is not a
   protection — it's provenance metadata, nothing more.
4. **One-time, one-directional.** Migration is a one-time action performed
   by whatever installer or upgrade tooling triggers it, not an ongoing sync.
   Once migrated content is inside `workspace/00000/`, it is an ordinary
   workspace going forward and follows the ordinary lifecycle.

## What's still open

- No migration tooling exists in this repo (no script, no `next-step`
  subcommand). This document describes the *outcome contract* a future
  migration path must satisfy, not an implementation.
- Whether a future Next Step protocol version bump would ever need this
  path (as opposed to it staying purely a carried-forward historical
  artifact from the ABX-STEP lineage) is genuinely undecided.
