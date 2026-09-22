package dun

import (
	"strings"
	"testing"
	"time"
)

func TestPeriodSummaryTitleUsesFullCalendarWeek(t *testing.T) {
	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	if got, want := periodSummaryTitle(periodWeek, anchor), "Week Summary (W38 — Sep 14-20)"; got != want {
		t.Fatalf("periodSummaryTitle() = %q, want %q", got, want)
	}
}

func TestNormalizePeriodReportReplacesGeneratedTitle(t *testing.T) {
	got := normalizePeriodReport("**Impact report (Week 38)**\n\n## Summary\nGood week.", "Week Summary (W38 — Sep 14-20)")
	want := "# Week Summary (W38 — Sep 14-20)\n\n## Summary\nGood week.\n"
	if got != want {
		t.Fatalf("normalizePeriodReport() = %q, want %q", got, want)
	}
}

func TestPeriodReportSignalsIncludesMetricsHilitesAndPendingItems(t *testing.T) {
	withTempDunnitDir(t)
	date := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, date, []string{
		"[09:00] TODO follow up with @Brandon #alpha",
		"[10:00] DONE shipped the fix ~30m #alpha",
		"[11:00] WIN shipped the fix #alpha",
		"[17:00] PRODUCTIVITY 4",
		"[17:01] SENTIMENT Positive",
	})
	InvalidateLedgerCaches()

	from := time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, time.September, 20, 23, 59, 59, 0, time.Local)
	got := periodReportSignals(from, to)
	for _, want := range []string{
		"- Entries: 5",
		"- Completed: 1",
		"- Tracked time: 0h 30m",
		"- Average productivity: 4.0/5",
		"WIN: shipped the fix #alpha",
		"TODO: follow up with @Brandon #alpha",
		"Talking points from the ledger:",
		"- **#alpha** — 3 mentions",
		"- **@Brandon** — 1 mention",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("periodReportSignals() missing %q in %q", want, got)
		}
	}
}

func TestPeriodReportSignalsCollapseCarryForwardCopies(t *testing.T) {
	withTempDunnitDir(t)
	first := time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, first, []string{
		"[09:00] TODO follow up with @Brandon #alpha",
	})
	writeLedgerLinesForDate(t, first.AddDate(0, 0, 1), []string{
		"[09:00] TODO follow up with @Brandon #alpha s/2026-09-14",
	})
	InvalidateLedgerCaches()

	got := periodReportSignals(first, first.AddDate(0, 0, 7))
	for _, want := range []string{
		"- Entries: 1",
		"- People mentioned: 1",
		"- Topics mentioned: 1",
		"- **#alpha** — 1 mention",
		"- **@Brandon** — 1 mention",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("periodReportSignals() missing %q in %q", want, got)
		}
	}
}

func TestReportMentionMapsCollapseCarryForwardCopies(t *testing.T) {
	first := time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local)
	entries := []LedgerEntry{
		{Date: first, Category: "TODO", Text: "follow up with @Brandon #alpha", Tags: []string{"#alpha"}, People: []string{"@Brandon"}},
		{Date: first.AddDate(0, 0, 1), Category: "TODO", Text: "follow up with @Brandon #alpha s/2026-09-14", Tags: []string{"#alpha"}, People: []string{"@Brandon"}},
	}
	tags, people := reportMentionMaps(entries)
	if tags["#alpha"].count != 1 || people["@brandon"].count != 1 {
		t.Fatalf("report mention counts = (%d, %d), want (1, 1)", tags["#alpha"].count, people["@brandon"].count)
	}
}
