package dun

import (
	"strings"
	"testing"
	"time"
)

func TestGatherTrendPointsUsesLatestValidReadingAndCountsDuplicates(t *testing.T) {
	withTempDunnitDir(t)
	day := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, day, []string{
		"[09:00] PRODUCTIVITY 2",
		"[10:00] PRODUCTIVITY 4",
		"[09:30] PRODUCTIVITY 9",
		"[09:00] SENTIMENT Negative",
		"[11:00] SENTIMENT positive",
		"[12:00] SENTIMENT Positive #home",
	})
	InvalidateLedgerCaches()

	points := gatherTrendPointsForRange(day, day.AddDate(0, 0, 1).Add(-time.Second))
	if len(points) != 1 {
		t.Fatalf("gatherTrendPointsForRange returned %d points, want 1", len(points))
	}
	point := points[0]
	if !point.productivitySet || point.productivity != 4 || point.productivityCount != 2 {
		t.Fatalf("productivity = %d, set %v, count %d; want latest 4, set, count 2", point.productivity, point.productivitySet, point.productivityCount)
	}
	if !point.sentimentSet || point.sentiment != 1 || point.sentimentCount != 2 {
		t.Fatalf("sentiment = %d, set %v, count %d; want positive, set, count 2", point.sentiment, point.sentimentSet, point.sentimentCount)
	}
}

func TestTrendTableRowsShowsMissingDays(t *testing.T) {
	from := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 0, 2).Add(-time.Second)
	points := []trendPoint{{date: from.AddDate(0, 0, 1), productivity: 3, productivitySet: true, sentiment: -1, sentimentSet: true}}
	rows := trendTableRows(points, from, to)
	if len(rows) != 2 {
		t.Fatalf("trendTableRows returned %d rows, want 2", len(rows))
	}
	if rows[0].productivity != "—" || rows[0].sentiment != "—" {
		t.Fatalf("first missing-day row = %#v, want dashes", rows[0])
	}
	if rows[1].productivity != "3/5" || rows[1].sentiment != "Negative" {
		t.Fatalf("logged-day row = %#v", rows[1])
	}
	if !strings.Contains(trendSummary(points, from, to), "1 of 2 days logged") {
		t.Fatalf("trend summary did not report missing day: %q", trendSummary(points, from, to))
	}
}

func TestGatherTrendPointsRejectsNonPositiveProductivity(t *testing.T) {
	withTempDunnitDir(t)
	day := time.Date(2026, time.September, 24, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, day, []string{
		"[09:00] PRODUCTIVITY 0",
		"[10:00] PRODUCTIVITY 6",
	})
	InvalidateLedgerCaches()
	if got := gatherTrendPointsForRange(day, day); len(got) != 0 {
		t.Fatalf("invalid productivity produced %d trend points, want none", len(got))
	}
}
