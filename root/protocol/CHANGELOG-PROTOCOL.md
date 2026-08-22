# Protocol Changelog

This tracks the **protocol** (onboarding chain, task/receipt/action-plan
model, manifest contract) — not the `engine/` Go module's own version,
which is tracked separately.

## [1.0] — Unreleased

Next Step protocol v1.0 succeeds the retired ABX-STEP protocol, whose last
version was v1.2.1. The onboarding chain (`next-step-onboarding.md`,
`state-0.md` through `state-3.md`), `PROTOCOL-FACTS.md`, and the `demo/`
scaffold were migrated from ABX-STEP v1.2.0 with naming updated throughout
(no ABX-STEP references remain) and paths updated to this repo's versioned
`protocol/v1.0/spec/` layout, resolved through the `protocol/current`
symlink.

Carried forward unchanged from the ABX-STEP lineage:
- Human-only, hard-gate authorization model.
- Workspace model: atomic claim via existence-as-claim registry pattern,
  mandatory `--workspace` scoping, hard `REJECTED` (not a silent default)
  when unspecified.
- Manifest mandatory fields including `WORKSPACE_ID`, `WORKSPACE_NAME`,
  `WRITE_PATHS`.
- Functional (not syntactic) verification approach; the
  `NEXT_STEP_RESULT=APPLIED|NOOP` self-report contract.
- Hook/gate architecture (`INGRESS`, `PRE_EXECUTE`, `POST_EXECUTE`,
  `EGRESS`) as the intended extension surface for future concurrent
  multi-agent execution — not yet built.

New at v1.0, not present in the ABX-STEP lineage:
- `task-acceptance-criteria.md` — consolidates verification/acceptance
  criteria that were previously scattered across `PROTOCOL-FACTS.md` and
  the state files into one document.
- Formalized JSON Schemas (`task-manifest.schema.json`,
  `receipt.schema.json`, `action-plan.schema.json`) for data shapes that
  were previously implicit in shell script behavior only.
- The locked task → receipt → action-plan object model: an action plan is
  a broad, reusable roadmap containing one or more tasks; a receipt links
  one zipped task to its scope and parent action plan; the submitting
  agent sees only the receipt, never the action plan.

Decided at v1.0, not left open:
- Protocol content is versioned from the start:
  `root/protocol/v1.0/{spec,schemas}/`, with `root/protocol/current` as an
  atomic symlink to the active version directory. All cross-references
  within the onboarding chain resolve through `protocol/current/`, never a
  hardcoded version number.

Open items carried forward, still not resolved:
- CLI command syntax for what were ABX-STEP's `run-task.sh`/`build-task.sh`
  shell entry points, pending the Go engine's actual subcommand design in
  Phase 4-5 of the build plan. The onboarding chain currently references
  the legacy script names verbatim as placeholders.

## [1.0] — Phase 5.5 addendum (gap closure before Phase 6)

Triggered by a direct v1.2.0-parity review after Phase 5, which found
eight gaps between the Go rewrite and actual documented ABX-STEP v1.2.0
behavior. Decisions recorded per Phase 5.5.0, verified against the real
v1.2.0 backup (not assumed) where the backup was available.

Resolved and implemented:
- **Hook architecture** (5.5.0.3 → 5.5.2): implemented now, not deferred
  to v1.4.0. `engine/internal/hooks` provides the lookup/exec mechanism
  (`$WORKSPACE_ROOT/hooks/<name>`, then `$NEXT_STEP_HOME/hooks/<name>`,
  true no-op if neither exists/is executable), wired into all four points
  in `task.Run`. Blocking semantics were not left undecided — they follow
  directly from PROTOCOL-FACTS.md's own "Can block?" column: `INGRESS`/
  `PRE_EXECUTE` (gates) block on non-zero exit; `POST_EXECUTE`/`EGRESS`
  (hooks) are logged, never block. No hook scripts ship with this repo.
- **Inter-workspace links** (5.5.0.4 → 5.5.3): implemented now, full
  lexical validation to the same standard as `WRITE_PATHS` — the
  documented rule already gave everything needed (mode enum, "same
  honesty category as `WRITE_PATHS`"). New optional `LINKS` manifest
  field, `task.ValidateLinks`, called from `Build` alongside
  `ValidateWritePaths`.
- **Receipts semantics conflict** (5.5.0.2 → 5.5.4): resolved as two
  genuinely different artifacts (option b), not a merge or a retirement.
  The pre-execution task/receipt/action-plan document
  (`engine/internal/receipt`) is unchanged. A new `engine/internal/ledger`
  package formalizes the post-execution execution-state record the old
  ABX-STEP lineage also called a "receipt" (verified against a real
  receipt file recovered from the v1.2.0 backup). `PROTOCOL-FACTS.md`'s
  `EGRESS` hook now explicitly documents which of the two it fires after
  (the ledger entry).
- **`protocol/current/templates/`** (5.5.1): documentation-accuracy fix,
  not new content. The real v1.2.0 backup's `templates/` directory is
  present but empty; `demo/` was always the whole of the migrated
  reference-asset set. `PROTOCOL-FACTS.md`'s path-enforcement section
  corrected accordingly.
- **`next-step-clipboard.sh`** (5.5.5): pulled forward from its originally
  planned Phase 7 slot, since the onboarding chain's documented command
  flow assumes it exists. Ported from `abx-clipboard.sh`
  (`ABX_*` → `NEXT_STEP_*`, `abx-step` → `next-step` home default) to
  `root/bin/next-step-clipboard.sh`. Still trusted-host-context only.
- **Minimal safety net** (5.5.7): `_test.go` coverage added for
  `registry`'s atomic claim/collision-retry behavior, `taskseq`'s
  global-counter correctness under concurrency, `task`'s `WRITE_PATHS`/
  `LINKS` escape-rejection paths, the approval hard-gate, the hooks
  lookup/exec mechanism itself, and an end-to-end smoke test confirming
  true no-op hook behavior with nothing installed. Not the full Phase 10
  scope — just enough that the bug classes already found once can't
  silently regress.

Still open, explicitly not resolved this phase:
- **"chat-delivery" build path** (5.5.0.1): the real v1.2.0 backup has
  exactly one reference to this (`PROTOCOL-FACTS.md`'s Task package
  format section) and no implementation anywhere. Needs the architect's
  recollection of what it actually was before it can be built or
  formally retired. `PROTOCOL-FACTS.md` now flags this inline rather than
  silently treating it as settled.
- **On-device binary distribution** (5.5.6): requires the real Termux/
  Android target device; not something a remote build environment can
  complete. The Phase 5 smoke-test sequence has been re-verified in the
  sandboxed environment (extended to also check true no-op hook behavior
  end-to-end and full ledger correctness across a run/rerun pair) but
  still needs to be run on actual hardware before this item closes.
