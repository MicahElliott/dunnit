package dun

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// dailySummaryPath returns the path for date's markdown summary doc
// (FR-18), living alongside that date's ledger file in the same
// year/month/week directory (same naming scheme as getLedger, just
// "summary-" instead of "ledger-" and ".md" instead of ".txt").
func dailySummaryPath(date time.Time) (dir, path string) {
	dir, _ = ledgerPathFor(date)
	path = filepath.Join(dir, "summary-"+date.Format("20060102")+".md")
	return dir, path
}

// draftDailySummary generates initial markdown content for date's
// summary doc via the existing gh copilot pipeline (summarize.go),
// scoped to just that single day's ledger. Returns "" (with the
// error) if there's nothing to summarize or the copilot call fails.
//
// hasRealLedgerContent (not a bare "" check) guards the copilot call:
// gatherLedgerTextForDate/concatLedgerFiles always emit a
// "# ledger-....txt" header line for any file that exists, even one
// with zero actual entries in it -- a bare emptiness check on that
// result is therefore always false (non-empty) even when there's
// nothing real to summarize, which previously let a near-empty ledger
// through to gh copilot and got back a confused response describing
// the missing content instead of a real summary (real bug, hit via
// both auto-draft-at-EOD and the manual "Daily Summary Doc..." tray
// item).
func draftDailySummary(date time.Time) (string, error) {
	ledgerText := gatherLedgerTextForDate(date)
	if !hasRealLedgerContent(ledgerText) {
		return "", nil
	}
	return summarizeWithCopilot(ledgerText)
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

// ensureDailySummaryDoc creates today's (or the given date's) summary
// doc with LLM-drafted content, only if it doesn't already exist --
// never overwrites existing hand-edited content (FR-18's core
// guarantee). Returns the file path and whether it was newly created.
func ensureDailySummaryDoc(date time.Time) (path string, created bool, err error) {
	dir, path := dailySummaryPath(date)
	if _, statErr := os.Stat(path); statErr == nil {
		return path, false, nil // already exists, leave it alone
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return path, false, err
	}
	content, err := draftDailySummary(date)
	if err != nil {
		return path, false, err
	}
	if content == "" {
		content = "# " + date.Format("2006-01-02") + "\n\n(no ledger entries to summarize yet)\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return path, false, err
	}
	return path, true, nil
}
