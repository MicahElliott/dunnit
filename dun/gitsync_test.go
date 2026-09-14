package dun

import (
	"strings"
	"testing"
	"time"
)

func TestSyncCommitMessageUsesAddedLedgerRange(t *testing.T) {
	diff := strings.Join([]string{
		"diff --git a/2026/Sep/ledger-Sat-20260912.txt b/2026/Sep/ledger-Sat-20260912.txt",
		"+++ b/2026/Sep/ledger-Sat-20260912.txt",
		"@@",
		"+[08:15:00] TODO start the report",
		"+[16:45:00] DONE finish the report",
		"diff --git a/status-w37-20260914.md b/status-w37-20260914.md",
		"+++ b/status-w37-20260914.md",
		"@@",
		"+# summary",
	}, "\n")

	want := "Dunnit sync [2026-09-12 08:15:00 - 2026-09-12 16:45:00]"
	if got := syncCommitMessage(diff); got != want {
		t.Fatalf("syncCommitMessage() = %q, want %q", got, want)
	}
}

func TestStagedEntryTimeRangeIgnoresContextAndReports(t *testing.T) {
	diff := strings.Join([]string{
		"+++ b/2026/Sep/ledger-Sat-20260912.txt",
		"@@",
		" [07:00:00] DONE old context",
		"+[09:00:00] DOING active work",
		"+++ b/2026/Sep/report.md",
		"@@",
		"+[23:59:59] not a ledger entry",
	}, "\n")

	oldest, newest, ok := stagedEntryTimeRange(diff)
	if !ok {
		t.Fatal("stagedEntryTimeRange() reported no ledger entries")
	}
	want := time.Date(2026, time.September, 12, 9, 0, 0, 0, time.Local)
	if !oldest.Equal(want) || !newest.Equal(want) {
		t.Fatalf("range = %v..%v, want %v..%v", oldest, newest, want, want)
	}
}

func TestSyncCommitMessageWithoutLedgerEntries(t *testing.T) {
	if got := syncCommitMessage("+++ b/summary.md\n+report text\n"); got != "Dunnit sync [no ledger entries]" {
		t.Fatalf("syncCommitMessage() = %q, want no-ledger fallback", got)
	}
}
