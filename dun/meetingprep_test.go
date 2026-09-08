package dun

import (
	"os"
	"testing"
	"time"
)

func TestNormalizeTag(t *testing.T) {
	cases := []struct{ in, want string }{
		{"jeff", "#jeff"},
		{"#jeff", "#jeff"},
		{"  boss  ", "#boss"},
		{"", ""},
		{"  ", ""},
	}
	for _, c := range cases {
		if got := normalizeTag(c.in); got != c.want {
			t.Errorf("normalizeTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLastN(t *testing.T) {
	entries := []taggedEntry{{line: "a"}, {line: "b"}, {line: "c"}}

	got := lastN(entries, 2)
	if len(got) != 2 || got[0].line != "b" || got[1].line != "c" {
		t.Errorf("lastN(3 items, 2) = %+v", got)
	}

	got = lastN(entries, 10)
	if len(got) != 3 {
		t.Errorf("lastN with n > len should return all items, got %+v", got)
	}
}

// writeLedgerFileForDate creates a ledger file for the given date with
// the given raw lines (test helper, mirrors DunnitDir's naming scheme).
func writeLedgerFileForDate(t *testing.T, date time.Time, lines []string) {
	t.Helper()
	dir, fname := ledgerPathFor(date)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(fname)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
}

func TestPullTaggedEntries(t *testing.T) {
	withTempDunnitDir(t)

	today := time.Now()
	oldDate := today.AddDate(0, 0, -30) // outside a 2-week lookback

	writeLedgerFileForDate(t, today, []string{
		"[09:00:00] MEETING #boss discussed roadmap",
		"[10:00:00] TODO unrelated item",
		"[11:00:00] DONE mentioned #boss in passing",
	})
	writeLedgerFileForDate(t, oldDate, []string{
		"[09:00:00] MEETING #boss old agenda item",
	})

	since := today.AddDate(0, 0, -14)
	got := pullTaggedEntries("#boss", since, nil)
	if len(got) != 2 {
		t.Fatalf("expected 2 matching entries within lookback, got %d: %+v", len(got), got)
	}
	for _, e := range got {
		if e.date.Before(since) {
			t.Errorf("entry %+v is before the since cutoff", e)
		}
	}
}

func TestPullTaggedEntries_CategoryFilter(t *testing.T) {
	withTempDunnitDir(t)

	today := time.Now()
	writeLedgerFileForDate(t, today, []string{
		"[09:00:00] MEETING #boss discussed roadmap",
		"[10:00:00] DONE mentioned #boss in passing",
		"[11:00:00] IDEA #boss maybe try this",
	})

	since := today.AddDate(0, 0, -14)

	got := pullTaggedEntries("#boss", since, categoryFilterSet("MEETING"))
	if len(got) != 1 {
		t.Fatalf("MEETING filter: expected 1 entry, got %d: %+v", len(got), got)
	}

	got = pullTaggedEntries("#boss", since, categoryFilterSet("Related"))
	if len(got) != 2 {
		t.Fatalf("Related filter: expected 2 entries (MEETING+IDEA), got %d: %+v", len(got), got)
	}

	got = pullTaggedEntries("#boss", since, categoryFilterSet("All"))
	if len(got) != 3 {
		t.Fatalf("All filter: expected 3 entries, got %d: %+v", len(got), got)
	}
}
