package dun

import (
	"os"
	"strings"
	"testing"
	"time"
)

func previousCarryWorkday(now time.Time) time.Time {
	date := now.AddDate(0, 0, -1)
	for date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
		date = date.AddDate(0, 0, -1)
	}
	return date
}

func TestCarryForwardDailyPlan_CopiesUnresolvedItem(t *testing.T) {
	tests := []struct {
		name     string
		category string
		text     string
	}{
		{name: "TODO", category: "TODO", text: "finish the report"},
		{name: "DOING", category: "DOING", text: "finishing the report"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTempDunnitDir(t)

			// Simulate a prior-day item by writing directly to a backdated
			// ledger file, since recordActivity always writes to today.
			yesterday := previousCarryWorkday(time.Now())
			writeLedgerLinesForDate(t, yesterday, []string{
				"[09:00:00] " + tt.category + " " + tt.text,
			})
			InvalidateLedgerCaches()

			carryForwardDailyPlan(time.Now())

			lines := readLedgerLines()
			if len(lines) != 1 {
				t.Fatalf("expected 1 carried-forward line in today's ledger, got %d: %v", len(lines), lines)
			}
			cat, text, ok := parseLedgerLine(lines[0])
			if !ok || cat != tt.category {
				t.Fatalf("expected a %s line, got %q", tt.category, lines[0])
			}
			if !strings.HasPrefix(text, tt.text) || !strings.Contains(text, " s/") {
				t.Errorf("expected carried-forward text to keep original text and add an s/YYYY-MM-DD suffix, got %q", text)
			}
		})
	}
}

func TestCarryForwardDailyPlan_SkipsResolvedItem(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := previousCarryWorkday(time.Now())
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report",
		"[10:00:00] DONE finish the report (via TODO)",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())

	lines := readLedgerLines()
	if len(lines) != 0 {
		t.Fatalf("expected resolved item NOT to carry forward, got %v", lines)
	}
}

func TestCarryForwardDailyPlan_DeduplicatesHistoricalAndTodayItems(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := previousCarryWorkday(time.Now())
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO repeated task s/2026-09-07",
		"[09:01:00] TODO repeated task s/2026-09-07",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())
	carryForwardDailyPlan(time.Now())

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected one deduplicated carry-forward line, got %d: %v", len(lines), lines)
	}

	// The existing today's item itself suppresses a duplicate.
	carryForwardDailyPlan(time.Now())
	if lines = readLedgerLines(); len(lines) != 1 {
		t.Fatalf("expected existing today's item to suppress duplicate, got %d: %v", len(lines), lines)
	}
}

func TestCarryForwardDailyPlan_IdempotentPerDay(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := previousCarryWorkday(time.Now())
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())
	carryForwardDailyPlan(time.Now()) // should be a no-op the second time

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected carry-forward to run exactly once per day, got %d lines: %v", len(lines), lines)
	}
}

func TestCarryForwardDailyPlan_PreservesOriginalSinceDate(t *testing.T) {
	withTempDunnitDir(t)

	twoDaysAgo := previousCarryWorkday(time.Now()).AddDate(0, 0, -2)
	writeLedgerLinesForDate(t, twoDaysAgo, []string{
		"[09:00:00] TODO finish the report",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	_, text, _ := parseLedgerLine(lines[0])
	wantSince := twoDaysAgo.Format("2006-01-02")
	if !strings.Contains(text, " s/"+wantSince) {
		t.Errorf("expected since date %q preserved, got text %q", wantSince, text)
	}
}

func TestCarryForwardSinceSuffixUsesFullDate(t *testing.T) {
	date := time.Date(2026, time.September, 11, 9, 0, 0, 0, time.Local)
	if got := carryForwardSinceSuffix(date); got != " s/2026-09-11" {
		t.Fatalf("carryForwardSinceSuffix(%v) = %q, want %q", date, got, " s/2026-09-11")
	}
}

func TestParseCarryForwardSinceAcceptsCurrentAndLegacyMarkers(t *testing.T) {
	since, ok := parseCarryForwardSince("task s/2026-09-11")
	if !ok || since.Format("2006-01-02") != "2026-09-11" {
		t.Fatalf("carry date = %v, %v", since, ok)
	}
	since, ok = parseCarryForwardSince("task (since 2026-09-11)")
	if !ok || since.Format("2006-01-02") != "2026-09-11" {
		t.Fatalf("legacy carry date = %v, %v", since, ok)
	}
	if _, ok := parseCarryForwardSince("task s/2026-99-99"); ok {
		t.Fatal("invalid carry date parsed successfully")
	}
}

func TestDailyCarryForwardUsesNewestPlanDayOnly(t *testing.T) {
	withTempDunnitDir(t)

	now := time.Now()
	older := now.AddDate(0, 0, -2)
	yesterday := previousCarryWorkday(now)
	writeLedgerLinesForDate(t, older, []string{
		"[09:00:00] TODO older task",
	})
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO newer task",
		"[09:01:00] DOING newer active task",
		"[09:02:00] RISK deployment concern",
	})
	InvalidateLedgerCaches()

	source, items, _ := dailyCarryForwardItems(now)
	if source.Format("2006-01-02") != yesterday.Format("2006-01-02") {
		t.Fatalf("source date = %v, want yesterday %v", source, yesterday)
	}
	if len(items) != 2 {
		t.Fatalf("expected only yesterday's two plan items, got %+v", items)
	}
	if items[0].Text != "newer task" || items[1].Text != "newer active task" {
		t.Fatalf("unexpected carry-forward items: %+v", items)
	}
}

func TestDailyCarryForwardSkipsWeekendDays(t *testing.T) {
	withTempDunnitDir(t)

	// Monday with an unresolved DOING left on Friday and a weekend plan
	// entry. SOD should skip the weekend and use Friday as the source day.
	now := time.Date(2026, time.September, 21, 9, 0, 0, 0, time.Local)
	friday := now.AddDate(0, 0, -3)
	saturday := now.AddDate(0, 0, -2)
	writeLedgerLinesForDate(t, friday, []string{
		"[09:00:00] DOING finish the report",
	})
	writeLedgerLinesForDate(t, saturday, []string{
		"[10:00:00] TODO weekend-only task",
	})
	InvalidateLedgerCaches()

	source, items, _ := dailyCarryForwardItems(now)
	if !sameCalendarDate(source, friday) {
		t.Fatalf("source date = %v, want Friday %v", source, friday)
	}
	if len(items) != 1 || items[0].Category != "DOING" || items[0].Text != "finish the report" {
		t.Fatalf("expected Friday DOING to carry across weekend, got %+v", items)
	}
}

func TestDailyCarryForwardSkipsResolvedAndOlderThanLookback(t *testing.T) {
	withTempDunnitDir(t)

	now := time.Now()
	older := now.AddDate(0, 0, -4)
	yesterday := previousCarryWorkday(now)
	writeLedgerLinesForDate(t, older, []string{
		"[09:00:00] TODO older unresolved task",
		"[09:01:00] TODO resolved task",
	})
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO resolved task",
		"[10:00:00] DONE resolved task (via TODO)",
	})
	InvalidateLedgerCaches()

	_, items, _ := dailyCarryForwardItems(now)
	if len(items) != 1 || items[0].Text != "older unresolved task" {
		t.Fatalf("expected the unresolved item from the older qualifying day, got %+v", items)
	}

	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -dailyCarryLookbackDays-1), []string{
		"[09:00:00] TODO too old",
	})
	InvalidateLedgerCaches()
	if source, items, _ := dailyCarryForwardItems(now); !source.IsZero() || len(items) != 0 {
		t.Fatalf("item outside lookback should not carry: source=%v items=%+v", source, items)
	}
}

func TestStaleDailyPlanItemsLookBeyondCarryWindow(t *testing.T) {
	withTempDunnitDir(t)

	now := time.Now()
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -8), []string{
		"[09:00:00] TODO stale task",
	})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -3), []string{
		"[09:00:00] TODO fresh task",
	})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -staleReviewLookbackDays-1), []string{
		"[09:00:00] TODO too old for daily review",
	})
	InvalidateLedgerCaches()

	items := staleDailyPlanItems(now)
	if len(items) != 1 || items[0].Text != "stale task" {
		t.Fatalf("expected only the seven-day-old item in stale review, got %+v", items)
	}
}

func TestStaleDailyPlanItemsDeduplicatesRepeatedCopies(t *testing.T) {
	withTempDunnitDir(t)

	now := time.Now()
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -10), []string{
		"[09:00:00] TODO repeated task",
	})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -9), []string{
		"[06:00:00] TODO repeated task s/" + now.AddDate(0, 0, -10).Format("2006-01-02"),
		"[06:01:00] TODO repeated task s/" + now.AddDate(0, 0, -10).Format("2006-01-02"),
	})
	InvalidateLedgerCaches()

	items := staleDailyPlanItems(now)
	if len(items) != 1 || items[0].Text != "repeated task" {
		t.Fatalf("expected one deduplicated stale item, got %+v", items)
	}
}

func TestResolvedCarryForwardCopiesDoNotResurface(t *testing.T) {
	withTempDunnitDir(t)

	now := time.Now()
	original := now.AddDate(0, 0, -10)
	resolved := now.AddDate(0, 0, -9)
	writeLedgerLinesForDate(t, original, []string{
		"[09:00:00] TODO repeated task",
	})
	writeLedgerLinesForDate(t, resolved, []string{
		"[06:00:00] SOMEDAY repeated task s/" + original.Format("2006-01-02") + " (via TODO)",
	})
	writeLedgerLinesForDate(t, now.AddDate(0, 0, -8), []string{
		"[06:00:00] TODO repeated task s/" + original.Format("2006-01-02"),
		"[06:01:00] TODO repeated task s/" + original.Format("2006-01-02"),
	})
	InvalidateLedgerCaches()

	if items := staleDailyPlanItems(now); len(items) != 0 {
		t.Fatalf("resolved carried-forward copies resurfaced: %+v", items)
	}
}

func TestCarryForwardStartDoneCollapsesLifecycleAcrossDays(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := previousCarryWorkday(time.Now())
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report ~20m",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())
	item := getOpenItems()[0]
	if item.Category != "TODO" {
		t.Fatalf("carried item category = %q, want TODO", item.Category)
	}
	if err := startPlannedItem(item); err != nil {
		t.Fatalf("start carried item: %v", err)
	}
	item = getOpenItems()[0]
	if err := completePlannedItem(item); err != nil {
		t.Fatalf("complete carried item: %v", err)
	}

	if items, _ := priorOpenItems(); len(items) != 0 {
		t.Fatalf("completed carried item should not reappear tomorrow: %+v", items)
	}
	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected one current lifecycle row, got %v", lines)
	}
	if cat, _, ok := parseLedgerLine(lines[0]); !ok || cat != "DONE" {
		t.Fatalf("current row = %q, want DONE", lines[0])
	}
}

func TestLegacyOngoingIsNotActiveOrCarried(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := previousCarryWorkday(time.Now())
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] ONGOING old ditto record",
	})
	InvalidateLedgerCaches()

	carryForwardDailyPlan(time.Now())
	if lines := readLedgerLines(); len(lines) != 0 {
		t.Fatalf("legacy ONGOING should not be carried: %v", lines)
	}
	if category, text, ok := parseLedgerLine("[09:00:00] ONGOING old ditto record"); !ok || category != legacyOngoingCategory || text != "old ditto record" {
		t.Fatal("legacy ONGOING should remain parseable as raw history")
	}
}

func TestStripCarryForwardSince(t *testing.T) {
	in := "finish the report s/2026-08-28"
	want := "finish the report"
	if got := stripCarryForwardSince(in); got != want {
		t.Errorf("stripCarryForwardSince(%q) = %q, want %q", in, got, want)
	}
	if got := stripCarryForwardSince(want); got != want {
		t.Errorf("stripCarryForwardSince(%q) = %q, want unchanged %q", want, got, want)
	}
	legacy := "finish the report (since 2026-08-28)"
	if got := stripCarryForwardSince(legacy); got != "finish the report" {
		t.Errorf("stripCarryForwardSince(%q) = %q, want core text", legacy, got)
	}
	duplicates := "finish the report (since 2026-08-28) (since 2026-08-28)"
	if got := stripCarryForwardSince(duplicates); got != "finish the report" {
		t.Errorf("stripCarryForwardSince(%q) = %q, want core text", duplicates, got)
	}
}

// writeLedgerLinesForDate writes lines directly to the ledger file
// for the given date, bypassing recordActivity (which always targets
// today) -- used to simulate "an item logged on a prior day" in
// tests.
func writeLedgerLinesForDate(t *testing.T, date time.Time, lines []string) {
	t.Helper()
	dir, path := ledgerPathFor(date)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		t.Fatalf("writeLedgerLinesForDate mkdir: %v", err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("writeLedgerLinesForDate create: %v", err)
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			t.Fatalf("writeLedgerLinesForDate write: %v", err)
		}
	}
}
