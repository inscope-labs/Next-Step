// Package fsm implements the onboarding state chain.
//
// For v1.0, this implements only the linear onboarding-chain flow
// (state-0 -> state-3, see root/protocol/v1.0/spec/) that the retired
// ABX-STEP v1.2.0 lineage already had. Treat this as scaffolding for a
// fuller state-machine engine (arbitrary state graphs, not just the linear
// chain) planned for a later minor version once live multi-agent
// onboarding feedback is available — not a full FSM build yet.
//
// Phase 4 stub: package skeleton only, no logic yet. Built out in Phase 5
// per the Next Step build plan.
package fsm
