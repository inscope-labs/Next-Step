# Changelog

All notable changes to Next Step will be documented in this file.

## [1.0.0] - 2026-08-23

Initial release. Feature parity with retired ABX-STEP v1.2.0, rebuilt on
the new architecture (Go engine, versioned protocol content, two-binary
trust split). See `root/protocol/CHANGELOG-PROTOCOL.md` for the protocol
side of this release.

Included:
- Host-side protocol functionality: workspace claim/registry, task
  build/validation, receipt generation, the post-execution ledger, hook
  architecture (`INGRESS`/`PRE_EXECUTE`/`POST_EXECUTE`/`EGRESS`),
  inter-workspace links.
- Onboarding chain (`state-0.md` through `state-3.md`) migrated from
  ABX-STEP v1.2.0, versioned under `root/protocol/v1.0/`, resolved
  through the `protocol/current` symlink.
- Public install flow (`install/install.sh`): platform detection,
  checksum-verified release binary fetch, non-destructive `root/` mirror.
- CI (build+test on every PR), release (cross-compile matrix + checksummed
  GitHub Release assets for `linux/amd64`, `linux/arm64`, `linux/arm`
  GOARM=7), and CodeQL workflows.
- Full engine test suite, including an end-to-end integration test (fresh
  install -> claim workspace -> build/approve/run a task -> generate a
  receipt) driven through the compiled CLI binary, not just internal
  package calls.

Not included in this release — tracked, not silently dropped (see
`docs/architecture-overview.md` and `docs/security-model.md`):
- Sandboxed task execution (`next-step-runner`, the fuller FSM engine,
  session isolation) is scaffolded structurally only.
  `engine/internal/sandbox` has no isolation logic yet; `next-step-runner`
  refuses to run rather than execute a task without that guarantee.
- On-device (Termux/Android) validation of the install flow against this
  real release, on real hardware, is still outstanding — see
  `root/protocol/CHANGELOG-PROTOCOL.md`'s Phase 5.5 addendum, item
  "On-device binary distribution."

## [Unreleased]
