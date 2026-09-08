package dun

import (
	"os"
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
	tomorrow := time.Now().AddDate(0, 0, 1).Format("20060102")
	want := "ledger-" + tomorrow + ".txt"
	if got := fname[len(fname)-len(want):]; got != want {
		t.Errorf("expected filename to end with %q, got %q (full: %q)", want, got, fname)
	}
}
