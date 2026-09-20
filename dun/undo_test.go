package dun

import (
	"os"
	"strings"
	"testing"
	"time"
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

func TestRecordActivityOmitsSeconds(t *testing.T) {
	withTempDunnitDir(t)
	if err := recordActivity("new format", "DONE"); err != nil {
		t.Fatalf("recordActivity: %v", err)
	}
	lines := readLedgerLines()
	if len(lines) != 1 || len(lines[0]) < 7 || lines[0][3] != ':' || lines[0][6] != ']' {
		t.Fatalf("recordActivity wrote unexpected timestamp: %q", lines)
	}
	if strings.Contains(lines[0][:7], ":00") {
		t.Fatalf("recordActivity still wrote seconds: %q", lines[0])
	}
}

func TestHistoricalItemEditUsesSourceLedger(t *testing.T) {
	withTempDunnitDir(t)
	yesterday := time.Now().AddDate(0, 0, -1)
	writeLedgerLinesForDate(t, yesterday, []string{"[09:00] WAITING waiting on review"})
	path := ledgerFileForDate(yesterday)
	item := OpenItem{Category: "WAITING", Text: "waiting on review", LineIndex: 0, Source: path}
	if err := replaceLedgerItemAt(item, "DOING", "continue the review"); err != nil {
		t.Fatalf("replace historical item: %v", err)
	}
	got := readLedgerLinesFrom(path)
	if len(got) != 1 || got[0] != "[09:00] DOING continue the review" {
		t.Fatalf("historical ledger = %v", got)
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

func TestRecordActivityFlattensNewlines(t *testing.T) {
	withTempDunnitDir(t)

	recordActivity("first line\r\nsecond line\nthird line", "DONE")

	_, fname := getLedger()
	data, err := os.ReadFile(fname)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if strings.Count(string(data), "\n") != 1 {
		t.Fatalf("expected one physical ledger line, got %q", data)
	}
	if !strings.Contains(string(data), "first line second line third line") {
		t.Fatalf("expected flattened text, got %q", data)
	}
}
