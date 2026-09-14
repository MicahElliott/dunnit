package dun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordTomorrowGoals(t *testing.T) {
	withTempDunnitDir(t)

	recordTomorrowGoals([]string{"learn go", "", "  ship feature  "})

	_, fname := tomorrowLedgerPath()
	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatalf("expected tomorrow's ledger to exist: %v", err)
	}
	want := "[05:00] GOAL learn go\n[05:00] GOAL ship feature\n"
	if got := string(data); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTomorrowLedgerPath_IsTomorrow(t *testing.T) {
	withTempDunnitDir(t)

	_, fname := tomorrowLedgerPath()
	tomorrow := time.Now().AddDate(0, 0, 1).Format("Mon-20060102")
	want := "ledger-" + tomorrow + ".txt"
	if got := fname[len(fname)-len(want):]; got != want {
		t.Errorf("expected filename to end with %q, got %q (full: %q)", want, got, fname)
	}
}

func TestAutoEODReportRequiresThreeDoneEntries(t *testing.T) {
	withTempDunnitDir(t)
	today := time.Now()

	if autoEODReportEligible(today) {
		t.Fatal("empty day should not be eligible for automatic EOD drafting")
	}
	recordActivity("one", "DONE")
	recordActivity("two", "DONE")
	if autoEODReportEligible(today) {
		t.Fatal("two DONE entries should not be eligible for automatic EOD drafting")
	}
	recordActivity("three", "DONE")
	if !autoEODReportEligible(today) {
		t.Fatal("three DONE entries should be eligible for automatic EOD drafting")
	}
}

func TestEODReportPathUsesDescriptorAndWeekday(t *testing.T) {
	date := time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local)
	_, path := eodReportPath(date)
	if got, want := filepath.Base(path), "eod-Mon-20260914.md"; got != want {
		t.Fatalf("eodReportPath() base = %q, want %q", got, want)
	}
}
