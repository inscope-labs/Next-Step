// Package registry implements workspace claim/registry logic.
//
// Replicates the atomic-claim semantics the retired ABX-STEP lineage
// achieved with a shell mkdir race-check, but as a Go-native operation
// rather than relying on filesystem race semantics as the sole safety
// mechanism. An unspecified workspace is a hard REJECTED, never a silent
// default.
//
// Phase 4 stub: package skeleton only, no logic yet. Built out in Phase 5
// per the Next Step build plan.
package registry
