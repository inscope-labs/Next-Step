package fsm

import "testing"

func TestIsValid(t *testing.T) {
	cases := []struct {
		s    State
		want bool
	}{
		{State0Discovery, true},
		{State1Facts, true},
		{State2Interaction, true},
		{State3, true},
		{numStates, false},
		{State(-1), false},
		{State(99), false},
	}
	for _, c := range cases {
		if got := IsValid(c.s); got != c.want {
			t.Errorf("IsValid(%d) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestNext_LinearChainNoBranching(t *testing.T) {
	order := []State{State0Discovery, State1Facts, State2Interaction, State3}
	s := order[0]
	for i := 1; i < len(order); i++ {
		next, ok := Next(s)
		if !ok {
			t.Fatalf("Next(%s) reported no next state before reaching the end of the chain", Name(s))
		}
		if next != order[i] {
			t.Errorf("Next(%s) = %s, want %s", Name(s), Name(next), Name(order[i]))
		}
		s = next
	}
	// State3 is the last state -- there is no branching in v1.0.
	if _, ok := Next(State3); ok {
		t.Error("Next(State3) reported a next state, but State3 is the last state in the linear chain")
	}
}

func TestNext_InvalidState(t *testing.T) {
	if _, ok := Next(State(99)); ok {
		t.Error("Next on an invalid state reported a next state")
	}
}

func TestName_KnownAndUnknown(t *testing.T) {
	if n := Name(State0Discovery); n == "" {
		t.Error("Name(State0Discovery) is empty")
	}
	if n := Name(State(99)); n == "" || n == Name(State0Discovery) {
		t.Errorf("Name for an unknown state should be a distinct non-empty placeholder, got %q", n)
	}
}

func TestSpecPath_AllValidStatesResolve(t *testing.T) {
	want := map[State]string{
		State0Discovery:   "spec/state-0.md",
		State1Facts:       "spec/state-1.md",
		State2Interaction: "spec/state-2.md",
		State3:            "spec/state-3.md",
	}
	for s, w := range want {
		got, err := SpecPath(s)
		if err != nil {
			t.Errorf("SpecPath(%s): %v", Name(s), err)
			continue
		}
		if got != w {
			t.Errorf("SpecPath(%s) = %q, want %q", Name(s), got, w)
		}
	}
}

func TestSpecPath_InvalidStateErrors(t *testing.T) {
	if _, err := SpecPath(State(99)); err == nil {
		t.Error("SpecPath on an invalid state returned no error")
	}
}
