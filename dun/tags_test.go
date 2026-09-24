package dun

import (
	"reflect"
	"strings"
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
	if got := tagUsageTooltip("#foo", stat); got != "Used 14 times in the last 30 days; 234 total" {
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

func TestDeduplicateCarryForwardEntriesNormalizesMetadataAndInflections(t *testing.T) {
	first := time.Date(2026, 9, 11, 0, 0, 0, 0, time.Local)
	entries := []LedgerEntry{
		{Date: first, Time: first.Add(6 * time.Hour), Category: "TODO", Text: "Wrap up #foo", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 1), Time: first.AddDate(0, 0, 1).Add(6 * time.Hour), Category: "TODO", Text: "Wrap up #foo s/2026-09-11", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 2), Time: first.AddDate(0, 0, 2).Add(6 * time.Hour), Category: "DOING", Text: "Wrapping up #foo ~1h s/2026-09-11", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 3), Time: first.AddDate(0, 0, 3).Add(6 * time.Hour), Category: "DONE", Text: "Wrapped up #foo ~2h s/2026-09-11 (via DOING)", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 3), Time: first.AddDate(0, 0, 3).Add(5 * time.Hour), Category: "TODO", Text: "Wrap up #foo", Tags: []string{"#foo"}},
		{Date: first.AddDate(0, 0, 4), Time: first.AddDate(0, 0, 4).Add(6 * time.Hour), Category: "DONE", Text: "Wrap up #foo", Tags: []string{"#foo"}},
	}

	got := deduplicateCarryForwardEntries(entries)
	if len(got) != 2 {
		t.Fatalf("deduplicated entries = %d, want one carried lineage plus one independent use: %+v", len(got), got)
	}
	if got[0].Category != "DONE" || got[0].Text != "Wrapped up #foo ~2h s/2026-09-11 (via DOING)" {
		t.Fatalf("deduplicated carried entry = %+v, want newest inflected row", got[0])
	}
	if got[1].Category != "DONE" || got[1].Text != "Wrap up #foo" {
		t.Fatalf("independent entry = %+v, want unmarked later use", got[1])
	}
}

func TestTagUsageTooltipUsesSingularForOneUse(t *testing.T) {
	stat := &tagStat{count: 1, recentCount: 1}
	if got := tagUsageTooltip("#foo", stat); got != "Used 1 time in the last 30 days; 1 total" {
		t.Fatalf("tagUsageTooltip = %q, want singular wording", got)
	}
}

func TestTagInsertionTextKeepsTagAtEnd(t *testing.T) {
	cases := []struct {
		name, text, tag, wantText string
		wantCursor                int
	}{
		{name: "empty entry", tag: "#foo", wantText: " #foo", wantCursor: 0},
		{name: "existing entry", text: "write update", tag: "#foo", wantText: "write update #foo", wantCursor: 13},
		{name: "trailing whitespace", text: "write update  ", tag: "#foo", wantText: "write update #foo", wantCursor: 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotText, gotCursor := tagInsertionText(tc.text, tc.tag)
			if gotText != tc.wantText || gotCursor != tc.wantCursor {
				t.Fatalf("tagInsertionText(%q, %q) = (%q, %d), want (%q, %d)",
					tc.text, tc.tag, gotText, gotCursor, tc.wantText, tc.wantCursor)
			}
		})
	}
}

func TestTagEntriesLast30DaysUsesCalendarWindow(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -29), []string{"[09:00] DONE first #foo"})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -30), []string{"[09:00] DONE too old #foo"})
	writeLedgerLinesForDate(t, now, []string{"[10:00] DONE latest #foo"})
	InvalidateLedgerCaches()

	entries := tagEntriesLast30Days("#foo", now)
	if len(entries) != 2 || !strings.Contains(entries[0].Text, "latest") || !strings.Contains(entries[1].Text, "first") {
		t.Fatalf("tagEntriesLast30Days() = %+v, want newest first with two entries", entries)
	}
}

func TestTagEntriesLast30DaysDeduplicatesLogicalEntries(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Date(2026, time.September, 23, 12, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -2), []string{
		"[09:00] TODO Wrap up #foo",
		"[09:01] TODO Wrap up #foo s/2026-09-21",
	})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -1), []string{
		"[09:02] DOING Wrapping up #foo ~1h s/2026-09-21",
		"[09:03] DONE Wrapped up #foo ~2h s/2026-09-21 (via DOING)",
	})
	InvalidateLedgerCaches()

	entries := tagEntriesLast30Days("#foo", now)
	if len(entries) != 1 || entries[0].Category != "DONE" {
		t.Fatalf("tagEntriesLast30Days() = %+v, want one newest DONE entry", entries)
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
