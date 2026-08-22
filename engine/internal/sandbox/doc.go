// Package sandbox will implement ephemeral sandboxed task execution
// primitives for next-step-runner.
//
// Present structurally, not functionally required for v1.0 — task
// execution isolation is explicitly out of scope for v1.0 per the build
// plan. Until this package is real, v1.0 deployments must not treat
// unreviewed task execution as isolated from the host. See
// docs/security-model.md §5 for tracked open items (sandbox technology
// choice, relay/runtime process isolation) this package will need to
// resolve.
package sandbox
