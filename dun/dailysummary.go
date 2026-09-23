package dun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// eodReportPath returns the path for date's markdown EOD report
// (FR-18), living alongside that date's ledger file in the same
// year/month/week directory (same naming scheme as getLedger, just
// "eod-" instead of "ledger-" and ".md" instead of ".txt").
func eodReportPath(date time.Time) (dir, path string) {
	dir, _ = ledgerPathFor(date)
	path = filepath.Join(dir, "eod-"+date.Format("Mon-20060102")+".md")
	return dir, path
}

// draftDailySummary generates initial markdown content for date's
// EOD report via the existing configured LLM CLI pipeline (summarize.go),
// scoped to just that single day's ledger. Returns "" (with the
// error) if there's nothing to summarize or the LLM CLI call fails.
//
// hasRealLedgerContent (not a bare "" check) guards the LLM CLI call:
// gatherLedgerTextForDate/concatLedgerFiles always emit a
// "# ledger-....txt" header line for any file that exists, even one
// with zero actual entries in it -- a bare emptiness check on that
// result is therefore always false (non-empty) even when there's
// nothing real to summarize, which previously let a near-empty ledger
// through to configured LLM CLI and got back a confused response describing
// the missing content instead of a real summary (real bug, hit via
// both auto-draft-at-EOD and the manual "EOD Report..." tray
// item).
func draftDailySummary(date time.Time) (string, error) {
	return draftDailySummaryContext(context.Background(), date)
}

func draftDailySummaryContext(ctx context.Context, date time.Time) (string, error) {
	ledgerText := gatherLedgerTextForDate(date)
	if !hasRealLedgerContent(ledgerText) {
		return "", nil
	}
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	to := from.AddDate(0, 0, 1).Add(-time.Nanosecond)
	if mentions := reportMentionContextForRange(from, to, nil); mentions != "" {
		ledgerText += "\n\n" + mentions
	}
	return summarizeWithLLMCLIPromptContext(ctx, eodSummaryPrompt(), ledgerText)
}

func eodSummaryPrompt() string {
	return "Create a detailed but compact Markdown end-of-day recap from this ledger. " +
		"Treat the ledger as the source of truth and account for every meaningful entry. " +
		"Make the first section a grouped ledger summary with separate headings for " +
		"categories such as DONE, DOING, TODO, and GOAL; do not scatter entries from " +
		"different categories together. " +
		"Do not omit repeated DONE entries: preserve each distinct completed outcome, " +
		"combining entries only when they clearly describe the same work. " +
		"Use this vocabulary: DONE means completed work; TODO means an open task; " +
		"DOING means work in progress; WAITING means blocked on another person or " +
		"thing; RISK means a tracked concern; TIL means something learned; GOAL " +
		"means a larger objective; MEETING means agenda or discussion notes. " +
		"Include several concrete bullets and retain useful people, topics, learnings, " +
		"and follow-up details. Be informative rather than ultra-concise, do not " +
		"invent facts, and do not include a title or a separate statistics section." +
		reportMentionPromptGuidance()
}

// hasRealLedgerContent reports whether ledgerText (as produced by
// concatLedgerFiles/gatherLedgerTextForDate) contains at least one
// real entry line, as opposed to just "# filename" header line(s)
// with nothing beneath them.
func hasRealLedgerContent(ledgerText string) bool {
	for _, line := range strings.Split(ledgerText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "# ") {
			continue
		}
		return true
	}
	return false
}

// ensureEODReport creates today's (or the given date's) EOD report
// with LLM-drafted content, only if it doesn't already exist --
// never overwrites existing hand-edited content (FR-18's core
// guarantee). Returns the file path and whether it was newly created.
func ensureEODReport(date time.Time) (path string, created bool, err error) {
	return ensureEODReportContext(context.Background(), date)
}

func ensureEODReportContext(ctx context.Context, date time.Time) (path string, created bool, err error) {
	dir, path := eodReportPath(date)
	if _, statErr := os.Stat(path); statErr == nil {
		return path, false, nil // already exists, leave it alone
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return path, false, err
	}
	content, err := draftDailySummaryContext(ctx, date)
	if err != nil {
		return path, false, err
	}
	if content == "" {
		content = "# " + date.Format("2006-01-02") + "\n\n(no ledger entries to summarize yet)\n"
	}
	content = augmentEODReport(content, date)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return path, false, err
	}
	return path, true, nil
}

// writeReportFileIfAbsent writes a newly generated EOD report without
// replacing an existing file. EOD checks this before opening, and O_EXCL
// keeps that no-overwrite guarantee if another process creates the report
// while the form is open.
func writeReportFileIfAbsent(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return err
}
