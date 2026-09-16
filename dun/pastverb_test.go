package dun

import "testing"

func TestPastTense(t *testing.T) {
	cases := map[string]string{
		// regular
		"walk":   "walked",
		"stop":   "stopped",
		"try":    "tried",
		"create": "created",
		"review": "reviewed",
		"fix":    "fixed",
		"ship":   "shipped",
		"plan":   "planned",
		"eat":    "ate", // irregular, sanity check it doesn't fall through to regular rules
		// irregular
		"go":    "went",
		"write": "wrote",
		"buy":   "bought",
		"meet":  "met",
		"do":    "did",
		"make":  "made",
		// case preservation
		"Walk": "Walked",
		"FIX":  "FIXED",
		"Go":   "Went",
	}
	for in, want := range cases {
		if got := PastTense(in); got != want {
			t.Errorf("PastTense(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPastTenseLeadingWord(t *testing.T) {
	cases := map[string]string{
		"Fix the login bug":     "Fixed the login bug",
		"walk the dog @15m":     "walked the dog @15m",
		"Ship it":               "Shipped it",
		"go to the store #home": "went to the store #home",
		"":                      "",
		"Solo":                  "Soloed", // no space at all: whole string treated as the word
	}
	for in, want := range cases {
		if got := PastTenseLeadingWord(in); got != want {
			t.Errorf("PastTenseLeadingWord(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPresentParticiple(t *testing.T) {
	cases := map[string]string{
		"send": "sending", "write": "writing", "make": "making",
		"run": "running", "fix": "fixing", "try": "trying",
		"tie": "tying", "Send": "Sending", "FIX": "FIXING",
	}
	for in, want := range cases {
		if got := PresentParticiple(in); got != want {
			t.Errorf("PresentParticiple(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBaseTenseLeadingWord(t *testing.T) {
	cases := map[string]string{
		"Sent the report":        "Send the report",
		"Sending the report @5m": "Send the report @5m",
		"Shipped the fix":        "Ship the fix",
		"Creating a task":        "Create a task",
		"Walk the dog":           "Walk the dog",
	}
	for in, want := range cases {
		if got := BaseTenseLeadingWord(in); got != want {
			t.Errorf("BaseTenseLeadingWord(%q) = %q, want %q", in, got, want)
		}
	}
}
