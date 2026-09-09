package dun

import (
	"fmt"
	"os/exec"
	"strings"

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
		if err := runGit("-C", DunnitDir(), "commit", "-m", "Dunnit sync"); err != nil && !strings.Contains(err.Error(), "nothing to commit") {
			return err
		}
		return runGit("-C", DunnitDir(), "push")
	case "pull":
		args = append(args, "add", "-A")
		if err := runGit(args...); err != nil {
			return err
		}
		if err := runGit("-C", DunnitDir(), "commit", "-m", "Dunnit sync"); err != nil && !strings.Contains(err.Error(), "nothing to commit") {
			return err
		}
		return runGit("-C", DunnitDir(), "pull", "--rebase")
	default:
		return fmt.Errorf("unknown git sync action %q", action)
	}
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
