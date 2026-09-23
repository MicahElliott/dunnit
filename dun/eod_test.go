package dun

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestEndOfDayAlreadyRunUsesCompletionMarker(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Now()
	cfg := LoadConfig()
	cfg.LastEndOfDayDate = now.Format("2006-01-02")
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if !endOfDayAlreadyRun(now) {
		t.Fatal("expected EOD completion marker to block another run")
	}
}

func TestWriteReportFileIfAbsentDoesNotOverwrite(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Now()
	_, path := eodReportPath(now)
	if err := writeReportFile(path, "# Existing report"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := writeReportFileIfAbsent(path, "# Replacement report"); !errors.Is(err, os.ErrExist) {
		t.Fatalf("writeReportFileIfAbsent() error = %v, want os.ErrExist", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if string(contents) != "# Existing report" {
		t.Fatalf("existing report changed to %q", contents)
	}
}

func TestEndOfDayAlreadyRunUsesExistingReport(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Now()
	_, path := eodReportPath(now)
	if err := writeReportFile(path, "# Existing report"); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if !endOfDayAlreadyRun(now) {
		t.Fatal("expected existing EOD report to block another run")
	}
}

func TestMarkEndOfDayRunPreservesConfig(t *testing.T) {
	withTempDunnitDir(t)
	cfg := LoadConfig()
	cfg.DayStart = "09:15"
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}

	now := time.Now()
	markEndOfDayRun(now)
	marked := LoadConfig()
	if marked.LastEndOfDayDate != now.Format("2006-01-02") || marked.DayStart != "09:15" {
		t.Fatalf("unexpected marked config: %+v", marked)
	}
}

func TestEODReportPathUsesDescriptorAndWeekday(t *testing.T) {
	date := time.Date(2026, time.September, 14, 0, 0, 0, 0, time.Local)
	_, path := eodReportPath(date)
	if got, want := filepath.Base(path), "eod-Mon-20260914.md"; got != want {
		t.Fatalf("eodReportPath() base = %q, want %q", got, want)
	}
}

func TestMarkdownToPlainTextRemovesFormatting(t *testing.T) {
	tests := []struct {
		name string
		md   string
		want string
	}{
		{name: "bold and heading", md: "# Report\n\n**Shipped** the fix", want: "Report\n\nShipped the fix"},
		{name: "link", md: "See [the report](https://example.com)", want: "See the report"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := markdownToPlainText(tt.md); got != tt.want {
				t.Fatalf("markdownToPlainText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEODReportForDisplayUsesCoveredDateAndStats(t *testing.T) {
	withTempDunnitDir(t)
	date := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, date, []string{
		"[09:00:00] DONE shipped the fix",
		"[17:00:00] PRODUCTIVITY 4",
	})
	InvalidateLedgerCaches()

	got := eodReportForDisplay("# End-of-Day Recap — Tue Sep 15\n\nThe work\n\nStats: old", date)
	if !strings.Contains(got, "# End-of-Day Recap — Wed Sep 16") {
		t.Fatalf("report heading did not use covered date: %q", got)
	}
	if !strings.Contains(got, "*Stats: 2 entries · 1 done · productivity 4/5*") {
		t.Fatalf("report stats missing or misplaced: %q", got)
	}
	if strings.Contains(got, "Tue Sep 15") {
		t.Fatalf("stale report heading survived: %q", got)
	}
	if strings.Count(got, "Stats:") != 1 {
		t.Fatalf("expected one stats line below the heading: %q", got)
	}
}

func TestEODReportFactsDeduplicatePeopleAndTopics(t *testing.T) {
	withTempDunnitDir(t)
	date := time.Date(2026, time.September, 17, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, date, []string{
		"[09:00] DONE shipped #alpha with @Brandon",
		"[10:00] TIL learned #alpha with @brandon",
		"[11:00] DONE reviewed #beta with @Surbhi",
	})
	InvalidateLedgerCaches()

	if got, want := eodReportFacts(date), "Worked with 2 people across 2 topics."; got != want {
		t.Fatalf("eodReportFacts() = %q, want %q", got, want)
	}
	got := appendEODReportFacts("Report body", date)
	if !strings.HasSuffix(got, "\n\n"+"Worked with 2 people across 2 topics.\n") {
		t.Fatalf("augmented report = %q", got)
	}
	if !strings.Contains(got, "\n\nWorked with 2 people across 2 topics.") {
		t.Fatalf("facts line needs a blank line above it: %q", got)
	}
}

func TestAugmentEODReportKeepsEveryDoneAndTIL(t *testing.T) {
	withTempDunnitDir(t)
	date := time.Date(2026, time.September, 17, 0, 0, 0, 0, time.Local)
	writeLedgerLinesForDate(t, date, []string{
		"[09:00] DONE first outcome",
		"[10:00] DONE second outcome",
		"[11:00] DONE third outcome",
		"[12:00] TIL learned the useful thing",
	})
	InvalidateLedgerCaches()

	got := augmentEODReport("AI summary", date)
	for _, want := range []string{
		"## Ledger entries by category",
		"### DONE",
		"- first outcome",
		"- second outcome",
		"- third outcome",
		"### TIL",
		"- learned the useful thing",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("augmented report missing %q: %q", want, got)
		}
	}
	if strings.Count(got, "- first outcome") != 1 || strings.Count(got, "- second outcome") != 1 || strings.Count(got, "- third outcome") != 1 {
		t.Fatalf("completed entries were not preserved exactly once: %q", got)
	}
	if strings.Index(got, "## Ledger entries by category") > strings.Index(got, "AI summary") {
		t.Fatalf("category grouping should be the first report section: %q", got)
	}
}

func TestReportHeadingAndBodySeparatesH1(t *testing.T) {
	heading, body := reportHeadingAndBody("# Report\n\n*Stats: 2 entries*\n\nBody")
	if heading != "Report" || body != "*Stats: 2 entries*\n\nBody" {
		t.Fatalf("reportHeadingAndBody() = %q, %q", heading, body)
	}
}

func TestLastEODReportUsesNewestExistingReport(t *testing.T) {
	withTempDunnitDir(t)
	now := time.Now()
	older := now.AddDate(0, 0, -2)
	yesterday := now.AddDate(0, 0, -1)
	_, olderPath := eodReportPath(older)
	_, yesterdayPath := eodReportPath(yesterday)
	if err := writeReportFile(olderPath, "# older"); err != nil {
		t.Fatalf("write older report: %v", err)
	}
	if err := writeReportFile(yesterdayPath, "# yesterday"); err != nil {
		t.Fatalf("write yesterday report: %v", err)
	}

	date, report, path := lastEODReport(now)
	if date.Format("2006-01-02") != yesterday.Format("2006-01-02") || report != "# yesterday" || path != yesterdayPath {
		t.Fatalf("lastEODReport() = %v, %q, %q; want yesterday report", date, report, path)
	}
}
