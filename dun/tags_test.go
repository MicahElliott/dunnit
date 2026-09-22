package dun

import (
	"reflect"
	"testing"
	"time"
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

func TestFormatTagWithCountIsCompact(t *testing.T) {
	if got := formatTagWithCount("#snap", &tagStat{count: 42}); got != "#snap(42)" {
		t.Fatalf("formatTagWithCount = %q, want %q", got, "#snap(42)")
	}
	tag, count := splitTagCount("#snap(42)")
	if tag != "#snap" || count != "(42)" {
		t.Fatalf("splitTagCount = (%q, %q), want (%q, %q)", tag, count, "#snap", "(42)")
	}
}

func TestTagUsageTooltipUsesRecentCount(t *testing.T) {
	stat := &tagStat{count: 234, recentCount: 14}
	if got := tagUsageTooltip("#foo", stat); got != "Used 14 times in the last 30 days" {
		t.Fatalf("tagUsageTooltip = %q, want recent-count tooltip", got)
	}
}

func TestTagStatsCollapseCarryForwardCopies(t *testing.T) {
	first := time.Date(2026, 9, 1, 0, 0, 0, 0, time.Local)
	entries := []LedgerEntry{
		{Date: first, Category: "TODO", Text: "ship it #foo", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 1), Category: "TODO", Text: "ship it #foo s/2026-09-01", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 2), Category: "DOING", Text: "shipping it #foo s/2026-09-01", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 2), Category: "DONE", Text: "shipped it #foo", Tags: []string{"#foo"}},
	}

	stats := gatherTagStatsFromEntries(deduplicateCarryForwardEntries(entries), first.AddDate(0, 0, 2))
	if stats["#foo"].count != 2 {
		t.Fatalf("tag count = %d, want original lineage plus independent DONE use", stats["#foo"].count)
	}
	if stats["#foo"].recentCount != 2 {
		t.Fatalf("recent tag count = %d, want 2", stats["#foo"].recentCount)
	}
}

func TestTagUsageTooltipUsesSingularForOneUse(t *testing.T) {
	stat := &tagStat{recentCount: 1}
	if got := tagUsageTooltip("#foo", stat); got != "Used 1 time in the last 30 days" {
		t.Fatalf("tagUsageTooltip = %q, want singular wording", got)
	}
}

func TestFinalizeTagStatsFavorsRecentUse(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	stats := map[string]*tagStat{
		"#old":    {score: 100, lastSeen: now.AddDate(0, 0, -30)},
		"#recent": {score: 1, lastSeen: now},
	}
	finalizeTagStats(stats, now)
	if stats["#recent"].score <= stats["#old"].score {
		t.Fatalf("recent tag score %v did not outrank old heavy tag score %v",
			stats["#recent"].score, stats["#old"].score)
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
