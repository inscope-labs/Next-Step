package task

import "testing"

func TestValidateWritePaths_AcceptsConfinedAndNone(t *testing.T) {
	cases := []string{"NONE", "files/a.txt", "a/b/c.txt,files/x.txt"}
	for _, c := range cases {
		if err := ValidateWritePaths(c); err != nil {
			t.Errorf("ValidateWritePaths(%q): unexpected error: %v", c, err)
		}
	}
}

// TestValidateWritePaths_RejectsEscapes is the escape-rejection path
// called out explicitly in Phase 5.5.7: every documented escape vector
// (absolute path, "..", "~") must be rejected lexically, before staging.
func TestValidateWritePaths_RejectsEscapes(t *testing.T) {
	cases := []string{
		"/etc/passwd",
		"../outside.txt",
		"files/../../outside.txt",
		"~/outside.txt",
		"files/a.txt,../escape.txt", // one good entry hiding one bad one
		"",
	}
	for _, c := range cases {
		if err := ValidateWritePaths(c); err == nil {
			t.Errorf("ValidateWritePaths(%q): expected rejection, got nil error", c)
		}
	}
}

func TestValidateLinks_NoneAndAbsentAreFine(t *testing.T) {
	exists := func(string) bool { return true }
	for _, c := range []string{"", "NONE"} {
		if err := ValidateLinks("ws-a", c, exists); err != nil {
			t.Errorf("ValidateLinks(%q): unexpected error: %v", c, err)
		}
	}
}

func TestValidateLinks_AcceptsWellFormedEntry(t *testing.T) {
	exists := func(id string) bool { return id == "ws-b" }
	if err := ValidateLinks("ws-a", "ws-b:read-only", exists); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateLinks_RejectsUnknownMode(t *testing.T) {
	exists := func(string) bool { return true }
	if err := ValidateLinks("ws-a", "ws-b:read-write", exists); err == nil {
		t.Error("expected rejection of unrecognized link mode, got nil")
	}
}

func TestValidateLinks_RejectsSelfLink(t *testing.T) {
	exists := func(string) bool { return true }
	if err := ValidateLinks("ws-a", "ws-a:read-only", exists); err == nil {
		t.Error("expected rejection of a link targeting the declaring task's own workspace, got nil")
	}
}

func TestValidateLinks_RejectsUnclaimedTarget(t *testing.T) {
	exists := func(string) bool { return false }
	if err := ValidateLinks("ws-a", "ws-b:read-only", exists); err == nil {
		t.Error("expected rejection of a link targeting an unclaimed workspace, got nil")
	}
}

func TestValidateLinks_RejectsMalformedEntry(t *testing.T) {
	exists := func(string) bool { return true }
	cases := []string{"ws-b", "ws-b:", ":read-only", "ws-b:read-only,"}
	for _, c := range cases {
		if err := ValidateLinks("ws-a", c, exists); err == nil {
			t.Errorf("ValidateLinks(%q): expected rejection, got nil", c)
		}
	}
}
