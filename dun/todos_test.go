package dun

import "testing"

func TestParseOpenItems(t *testing.T) {
	lines := []string{
		"[08:00:00] TODO write report",
		"[08:05:00] GOAL learn go",
		"[09:00:00] DONE write report (via TODO)",
		"[10:00:00] TODO another task",
		"[11:00:00] TODO stalled task",
		"[11:05:00] SOMEDAY stalled task (via TODO)",
	}
	open := parseOpenItems(lines)
	if len(open) != 2 {
		t.Fatalf("expected 2 open items, got %d: %+v", len(open), open)
	}
	if open[0].Category != "GOAL" || open[0].Text != "learn go" {
		t.Errorf("unexpected open[0]: %+v", open[0])
	}
	if open[1].Category != "TODO" || open[1].Text != "another task" {
		t.Errorf("unexpected open[1]: %+v", open[1])
	}
}

func TestParseOpenItems_InflectedDoneResolvesSource(t *testing.T) {
	lines := []string{
		"[08:00:00] TODO write report",
		"[08:05:00] DONE wrote report (via TODO)",
		"[09:00:00] TODO fix the login bug",
		"[09:05:00] DONE fixed the login bug (via TODO)",
	}

	open := parseOpenItems(lines)
	if len(open) != 0 {
		t.Fatalf("expected inflected DONE entries to resolve TODOs, got %+v", open)
	}
}
