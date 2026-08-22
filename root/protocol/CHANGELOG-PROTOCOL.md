# Protocol Changelog

This tracks the **protocol** (onboarding chain, task/receipt/action-plan
model, manifest contract) — not the `engine/` Go module's own version,
which is tracked separately.

## [1.0] — Unreleased

Next Step protocol v1.0 succeeds the retired ABX-STEP protocol, whose last
version was v1.2.1. The onboarding chain (`next-step-onboarding.md`,
`state-0.md` through `state-3.md`), `PROTOCOL-FACTS.md`, and the `demo/`
scaffold were migrated from ABX-STEP v1.2.0 with naming updated throughout
(no ABX-STEP references remain) and paths updated to this repo's flat
`protocol/spec/` layout.

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

Open items carried forward, not yet resolved:
- Whether protocol content should live under versioned `protocol/vX.Y.Z/`
  directories with an atomic `protocol/current` symlink (as ABX-STEP did),
  or the current flat `protocol/spec/` layout. See
  `docs/architecture-overview.md` §5.
- CLI command syntax for what were ABX-STEP's `run-task.sh`/`build-task.sh`
  shell entry points, pending the Go engine's actual subcommand design in
  Phase 4-5 of the build plan. The onboarding chain currently references
  the legacy script names verbatim as placeholders.
