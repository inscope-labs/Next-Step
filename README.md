# Next Step

Next Step is a human-authorized, hash-bound task execution protocol for AI coding agents.

> Status: scaffolding in progress. This repository is being rebuilt from the retired
> ABX-STEP v1.2.0 protocol onto a new architecture (Go engine, repo↔install mirror,
> two-binary trust split). See `docs/architecture-overview.md` for full design rationale.

## Install

```bash
git clone https://github.com/inscope-labs/Next-Step.git
cd Next-Step
install/install.sh
```

Detects your platform (`linux/amd64`, `linux/arm64`, or `linux/arm` —
Termux/Android — v7 only), fetches and checksum-verifies the matching
`next-step` binary from the latest GitHub Release, and installs it plus the
`root/` payload to `$NEXT_STEP_HOME` (default `$HOME/next-step`).

Must be run from inside a checked-out copy of this repo (as above), not via
curl-pipe — it mirrors this checkout's own `root/` directory onto disk. No
release has been published yet as of Phase 8; the script will fail clearly
(exit 5) rather than silently until Phase 11's first tagged release.

## Documentation

- [Architecture Overview](docs/architecture-overview.md)
- [Security Model](docs/security-model.md)
- [Protocol Spec](root/protocol/current/spec/)
