package dun

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestRunCarryForwardIfNeeded_CopiesUnresolvedItem(t *testing.T) {
	withTempDunnitDir(t)

	// Simulate a TODO logged "yesterday" by writing directly to a
	// backdated ledger file, since recordActivity always writes to
	// today's file.
	yesterday := time.Now().AddDate(0, 0, -1)
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report",
	})
	InvalidateLedgerCaches()

	runCarryForwardIfNeeded()

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 carried-forward line in today's ledger, got %d: %v", len(lines), lines)
	}
	cat, text, ok := parseLedgerLine(lines[0])
	if !ok || cat != "TODO" {
		t.Fatalf("expected a TODO line, got %q", lines[0])
	}
	if !strings.HasPrefix(text, "finish the report") || !strings.Contains(text, "(since ") {
		t.Errorf("expected carried-forward text to keep original text and add a (since ...) suffix, got %q", text)
	}
}

func TestRunCarryForwardIfNeeded_SkipsResolvedItem(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := time.Now().AddDate(0, 0, -1)
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report",
		"[10:00:00] DONE finish the report (via TODO)",
	})
	InvalidateLedgerCaches()

	runCarryForwardIfNeeded()

	lines := readLedgerLines()
	if len(lines) != 0 {
		t.Fatalf("expected resolved item NOT to carry forward, got %v", lines)
	}
}

func TestRunCarryForwardIfNeeded_IdempotentPerDay(t *testing.T) {
	withTempDunnitDir(t)

	yesterday := time.Now().AddDate(0, 0, -1)
	writeLedgerLinesForDate(t, yesterday, []string{
		"[09:00:00] TODO finish the report",
	})
	InvalidateLedgerCaches()

	runCarryForwardIfNeeded()
	runCarryForwardIfNeeded() // should be a no-op the second time

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected carry-forward to run exactly once per day, got %d lines: %v", len(lines), lines)
	}
}

func TestRunCarryForwardIfNeeded_PreservesOriginalSinceDate(t *testing.T) {
	withTempDunnitDir(t)

	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	writeLedgerLinesForDate(t, twoDaysAgo, []string{
		"[09:00:00] TODO finish the report",
	})
	InvalidateLedgerCaches()

	runCarryForwardIfNeeded()

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %v", len(lines), lines)
	}
	_, text, _ := parseLedgerLine(lines[0])
	wantSince := twoDaysAgo.Format("2006-01-02")
	if !strings.Contains(text, "(since "+wantSince+")") {
		t.Errorf("expected since date %q preserved, got text %q", wantSince, text)
	}
}

func TestStaleBadge(t *testing.T) {
	fresh := "do a thing (since " + time.Now().Format("2006-01-02") + ")"
	if got := staleBadge(fresh); got != "" {
		t.Errorf("expected no stale badge for a fresh item, got %q", got)
	}

	old := "do a thing (since " + time.Now().AddDate(0, 0, -10).Format("2006-01-02") + ")"
	if got := staleBadge(old); got == "" {
		t.Errorf("expected a stale badge for a 10-day-old item, got none")
	}

	noSuffix := "do a thing"
	if got := staleBadge(noSuffix); got != "" {
		t.Errorf("expected no stale badge for text with no since-suffix, got %q", got)
	}
}

func TestStripCarryForwardSince(t *testing.T) {
	in := "finish the report (since 2026-08-28)"
	want := "finish the report"
	if got := stripCarryForwardSince(in); got != want {
		t.Errorf("stripCarryForwardSince(%q) = %q, want %q", in, got, want)
	}
	if got := stripCarryForwardSince(want); got != want {
		t.Errorf("stripCarryForwardSince(%q) = %q, want unchanged %q", want, got, want)
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
