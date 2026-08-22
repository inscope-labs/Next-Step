// Package fsm implements the onboarding state chain.
//
// For v1.0, this implements only the linear onboarding-chain flow
// (state-0 -> state-3, see root/protocol/v1.0/spec/) that the retired
// ABX-STEP v1.2.0 lineage already had. Treat this as scaffolding for a
// fuller state-machine engine (arbitrary state graphs, not just the linear
// chain) planned for a later minor version once live multi-agent
// onboarding feedback is available — not a full FSM build yet.
//
// Onboarding itself is walked by an AI agent reading and following the
// markdown state files interactively; this package does not "run"
// onboarding. What it provides for v1.0 is a small, honest piece of
// scaffolding: naming the valid states, the one legal linear transition
// order, and where each state's spec file lives — useful for a future
// session-progress tracker, without inventing state-tracking behavior
// that isn't asked for yet.
package fsm

import (
	"fmt"
	"path/filepath"
)

// State is one step in the linear onboarding chain.
type State int

const (
	State0Discovery State = iota
	State1Facts
	State2Interaction
	State3 // see state-3.md for what this covers; named generically here
	// since this package doesn't need to interpret content, only order.
	numStates
)

var stateFiles = map[State]string{
	State0Discovery:   "state-0.md",
	State1Facts:       "state-1.md",
	State2Interaction: "state-2.md",
	State3:            "state-3.md",
}

var stateNames = map[State]string{
	State0Discovery:   "STATE 0 — Discovery & Inventory",
	State1Facts:       "STATE 1 — Facts",
	State2Interaction: "STATE 2 — Live Interaction",
	State3:            "STATE 3",
}

// IsValid reports whether s is one of the defined linear states.
func IsValid(s State) bool {
	return s >= State0Discovery && s < numStates
}

// Next returns the next state in the fixed linear order, and false if s is
// already the last state (there is no branching in v1.0 — see package doc).
func Next(s State) (State, bool) {
	if !IsValid(s) || s == numStates-1 {
		return 0, false
	}
	return s + 1, true
}

// Name returns a human-readable label for a state.
func Name(s State) string {
	if n, ok := stateNames[s]; ok {
		return n
	}
	return fmt.Sprintf("unknown state %d", s)
}

// SpecPath returns the path to a state's spec file, relative to the
// protocol/current/ directory (e.g. "spec/state-0.md"). Resolving this
// against a real $NEXT_STEP_HOME is the caller's responsibility, since
// this package has no filesystem dependency by design.
func SpecPath(s State) (string, error) {
	f, ok := stateFiles[s]
	if !ok {
		return "", fmt.Errorf("no spec file mapping for state %d", s)
	}
	return filepath.Join("spec", f), nil
}
