package dun

import (
	"os"
	"testing"
)

// withTempDunnitDir points DUNNIT_DIR at a fresh temp directory for the
// duration of a test, restoring the previous value afterward.
func withTempDunnitDir(t *testing.T) {
	t.Helper()
	old := os.Getenv("DUNNIT_DIR")
	dir := t.TempDir()
	os.Setenv("DUNNIT_DIR", dir)
	t.Cleanup(func() { os.Setenv("DUNNIT_DIR", old) })
}

func TestRemoveLastLedgerLine(t *testing.T) {
	withTempDunnitDir(t)

	recordActivity("first", "DONE")
	recordActivity("second", "DONE")

	if err := removeLastLedgerLine(); err != nil {
		t.Fatalf("removeLastLedgerLine: %v", err)
	}

	lines := readLedgerLines()
	if len(lines) != 1 {
		t.Fatalf("expected 1 line after removal, got %d: %v", len(lines), lines)
	}
	if got := lastEntryText(); got != "first" {
		t.Errorf("expected remaining line to be 'first', got %q", got)
	}
}

func TestRemoveLastLedgerLine_Empty(t *testing.T) {
	withTempDunnitDir(t)

	if err := removeLastLedgerLine(); err != nil {
		t.Fatalf("removeLastLedgerLine on empty ledger should be a no-op, got err: %v", err)
	}
}

func TestReplaceLastLedgerLine(t *testing.T) {
	withTempDunnitDir(t)

	recordActivity("first", "DONE")
	recordActivity("second", "DONE")

	newLine := "[12:00:00] DONE edited second"
	if err := replaceLastLedgerLine(newLine); err != nil {
		t.Fatalf("replaceLastLedgerLine: %v", err)
	}

	lines := readLedgerLines()
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines after replace, got %d: %v", len(lines), lines)
	}
	if lines[1] != newLine {
		t.Errorf("expected last line %q, got %q", newLine, lines[1])
	}
	if lines[0] == newLine {
		t.Errorf("first line should be untouched")
	}
}
