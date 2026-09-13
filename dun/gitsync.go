package dun

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func runGitSyncAction(a fyne.App, action string) {
	go func() {
		if err := gitSync(action); err != nil {
			if trayWindow != nil {
				dialog.ShowError(err, trayWindow)
			}
			return
		}
		if trayWindow != nil {
			dialog.ShowInformation("Dunnit Sync", "Git "+action+" completed.", trayWindow)
		}
	}()
}

func gitSync(action string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not installed or not on PATH: %w", err)
	}
	args := []string{"-C", DunnitDir()}
	switch action {
	case "push":
		args = append(args, "add", "-A")
		if err := runGit(args...); err != nil {
			return err
		}
		if err := commitStagedSyncChanges(DunnitDir()); err != nil {
			return err
		}
		return runGit("-C", DunnitDir(), "push")
	case "pull":
		args = append(args, "add", "-A")
		if err := runGit(args...); err != nil {
			return err
		}
		if err := commitStagedSyncChanges(DunnitDir()); err != nil {
			return err
		}
		return runGit("-C", DunnitDir(), "pull", "--rebase")
	default:
		return fmt.Errorf("unknown git sync action %q", action)
	}
}

func commitStagedSyncChanges(dir string) error {
	diff, err := runGitOutput("-C", dir, "diff", "--cached", "--unified=0")
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return nil
	}
	return runGit("-C", dir, "commit", "-m", syncCommitMessage(diff))
}

func syncCommitMessage(stagedDiff string) string {
	oldest, newest, ok := stagedEntryTimeRange(stagedDiff)
	if !ok {
		return "Dunnit sync [no ledger entries]"
	}
	return fmt.Sprintf("Dunnit sync [%s - %s]", oldest.Format("2006-01-02 15:04:05"), newest.Format("2006-01-02 15:04:05"))
}

// stagedEntryTimeRange finds added ledger entries in the staged diff. This
// reports the entries actually included in the commit, while allowing an
// accompanying report or config file to be staged by the same git add -A.
func stagedEntryTimeRange(stagedDiff string) (oldest, newest time.Time, ok bool) {
	var date *time.Time
	for _, line := range strings.Split(stagedDiff, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			date = ledgerFileDate(strings.TrimPrefix(line, "+++ b/"))
			continue
		}
		if date == nil || !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		entry, parsed := parseLedgerEntry(strings.TrimPrefix(line, "+"), *date, "", 0)
		if !parsed || entry.Time.IsZero() {
			continue
		}
		if !ok || entry.Time.Before(oldest) {
			oldest = entry.Time
		}
		if !ok || entry.Time.After(newest) {
			newest = entry.Time
		}
		ok = true
	}
	return oldest, newest, ok
}

func runGit(args ...string) error {
	_, err := runGitOutput(args...)
	return err
}

func runGitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
