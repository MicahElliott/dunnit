package main

import (
	core "dun/dun"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIRecordsEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DUNNIT_DIR", dir)

	if got := runCLI([]string{"DONE", "Finished the CLI fix"}); got != 0 {
		t.Fatalf("runCLI returned %d, want 0", got)
	}

	var ledger string
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "ledger-") {
			ledger = path
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk ledger directory: %v", err)
	}
	if ledger == "" {
		t.Fatal("runCLI did not create a ledger file")
	}
	data, err := os.ReadFile(ledger)
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	if !strings.Contains(string(data), " DONE Finished the CLI fix\n") {
		t.Fatalf("ledger = %q, want DONE entry", data)
	}
}

func TestRunCLIReturnsNonZeroWhenLedgerWriteFails(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dunnit-root-file")
	if err := os.WriteFile(dir, []byte("not a directory"), 0644); err != nil {
		t.Fatalf("create blocking path: %v", err)
	}
	t.Setenv("DUNNIT_DIR", dir)

	if got := runCLI([]string{"DONE", "This cannot be recorded"}); got == 0 {
		t.Fatal("runCLI returned 0 after the ledger write failed")
	}
}

func TestRunCLITagDefinition(t *testing.T) {
	t.Setenv("DUNNIT_DIR", t.TempDir())
	if got := runCLI([]string{
		"tag", "#SCRUM-12345", "--title", "Email delivery", "--kind", "ticket",
		"--status", "active", "--alias", "old-ticket", "--exclude",
	}); got != 0 {
		t.Fatalf("runCLI tag returned %d, want 0", got)
	}
	definition, found, err := core.LoadTagDefinition("SCRUM-12345")
	if err != nil || !found {
		t.Fatalf("LoadTagDefinition = %#v, %v, found %v", definition, err, found)
	}
	if definition.Title != "Email delivery" || definition.Kind != "ticket" || definition.Status != "active" || len(definition.Aliases) != 1 || !definition.Exclude {
		t.Fatalf("tag definition = %#v, want CLI fields", definition)
	}
}
