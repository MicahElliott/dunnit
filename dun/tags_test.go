package dun

import (
	"reflect"
	"testing"
)

func TestExtractTags(t *testing.T) {
	got := extractTags("worked on #foo and #bar-baz, also #foo again #pts:3")
	want := []string{"#foo", "#bar-baz", "#pts:3"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractTags_None(t *testing.T) {
	if got := extractTags("no tags here"); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestMatchingTags(t *testing.T) {
	candidates := []string{"#boss", "#personal", "#ticketno", "#emacs"}

	got := matchingTags(candidates, "os")
	want := []string{"#boss"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := matchingTags(candidates, ""); got != nil {
		t.Errorf("expected nil for empty fragment, got %v", got)
	}

	if got := matchingTags(candidates, "zzz"); got != nil {
		t.Errorf("expected nil for no match, got %v", got)
	}
}

func TestMatchingTags_PrefixPriority(t *testing.T) {
	// "emacs" and "email" both start with "e", "wetware" only
	// contains "e" mid-word -- prefix matches should come first.
	candidates := []string{"#wetware", "#emacs", "#email"}
	got := matchingTags(candidates, "e")
	want := []string{"#emacs", "#email", "#wetware"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCurrentTagFragment(t *testing.T) {
	cases := []struct {
		text      string
		cursor    int
		wantStart int
		wantFrag  string
		wantOK    bool
	}{
		{"hello #wo", 9, 6, "#wo", true},
		{"hello #wo", 7, 6, "#", true}, // cursor right after '#'
		{"hello #wo there", 9, 6, "#wo", true},
		{"hello there", 11, 0, "", false},
		{"#tag", 4, 0, "#tag", true},
		{"", 0, 0, "", false},
	}
	for _, c := range cases {
		start, frag, ok := currentTagFragment(c.text, c.cursor)
		if ok != c.wantOK || (ok && (start != c.wantStart || frag != c.wantFrag)) {
			t.Errorf("currentTagFragment(%q, %d) = (%d, %q, %v), want (%d, %q, %v)",
				c.text, c.cursor, start, frag, ok, c.wantStart, c.wantFrag, c.wantOK)
		}
	}
}
