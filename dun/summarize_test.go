package dun

import "testing"

func TestLineHasExcludedTag(t *testing.T) {
	cases := []struct {
		name        string
		line        string
		excludeTags []string
		want        bool
	}{
		{"exact match", "bought groceries #home", []string{"#home"}, true},
		{"no match", "fixed the bug", []string{"#home"}, false},
		{"no exclude tags configured", "bought groceries #home", nil, false},
		// Regression: Settings' exclude-tags field and ledger lines
		// can easily end up with differently-cased "same" tag (e.g.
		// "#Home" logged vs "#home" configured) -- matching must be
		// case-insensitive, see lineHasExcludedTag's doc comment.
		{"case-insensitive match, tag uppercase", "bought groceries #Home", []string{"#home"}, true},
		{"case-insensitive match, config uppercase", "bought groceries #home", []string{"#Home"}, true},
		{"distinct tag with shared prefix not matched", "cleaned #home-office", []string{"#home"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := lineHasExcludedTag(c.line, c.excludeTags); got != c.want {
				t.Errorf("lineHasExcludedTag(%q, %v) = %v, want %v", c.line, c.excludeTags, got, c.want)
			}
		})
	}
}
