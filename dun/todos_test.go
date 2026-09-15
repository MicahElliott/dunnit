package dun

import (
	"testing"
	"time"
)

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

func TestGetCategoryGroupItemsExcludesEODOnlyFromHilites(t *testing.T) {
	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, time.Now(), []string{
		"[17:00:00] PRODUCTIVITY 4",
		"[17:01:00] MEETING_HOURS 2.5",
		"[17:02:00] WIN shipped the fix",
	})

	items := getCategoryGroupItems("hilite")
	if len(items) != 1 {
		t.Fatalf("got %d Hilites, want 1: %+v", len(items), items)
	}
	if items[0].Category != "WIN" {
		t.Fatalf("Hilite = %+v, want WIN only", items[0])
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

func TestParseOpenItems_CollapsesTodoAndDoing(t *testing.T) {
	lines := []string{
		"[08:00:00] TODO ship the fix",
		"[09:00:00] DOING ship the fix",
	}

	open := parseOpenItems(lines)
	if len(open) != 1 {
		t.Fatalf("expected one logical item, got %+v", open)
	}
	if open[0].Category != "DOING" || open[0].LineIndex != 1 {
		t.Fatalf("expected newest DOING state, got %+v", open[0])
	}
}

func TestParseOpenItems_IndependentDoneDoesNotResolveOpenItem(t *testing.T) {
	open := parseOpenItems([]string{
		"[08:00:00] TODO ship the fix",
		"[09:00:00] DONE ship the fix",
	})
	if len(open) != 1 || open[0].Category != "TODO" {
		t.Fatalf("independent DONE changed open lifecycle: %+v", open)
	}
}

func TestPlannedLifecycleTransitionsPreserveRowAndResolve(t *testing.T) {
	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, time.Now(), []string{
		"[08:00:00] TODO ship the fix @12m",
	})
	InvalidateLedgerCaches()

	item := getOpenItems()[0]
	if err := startPlannedItem(item); err != nil {
		t.Fatalf("startPlannedItem: %v", err)
	}
	if err := startPlannedItem(item); err != nil {
		t.Fatalf("repeated startPlannedItem: %v", err)
	}
	lines := readLedgerLines()
	wantDoing := "[08:00:00] DOING ship the fix @12m"
	if len(lines) != 1 || lines[0] != wantDoing {
		t.Fatalf("start changed row to %q, want %q", lines, wantDoing)
	}

	item = getOpenItems()[0]
	if err := completePlannedItem(item); err != nil {
		t.Fatalf("completePlannedItem: %v", err)
	}
	wantDone := "[08:00:00] DONE ship the fix @12m (via DOING)"
	lines = readLedgerLines()
	if len(lines) != 1 || lines[0] != wantDone {
		t.Fatalf("complete changed row to %q, want %q", lines, wantDone)
	}
	if got := parseEntryMins(lines[0]); got != 12 {
		t.Errorf("completed duration = %d, want 12", got)
	}
	if open := getOpenItems(); len(open) != 0 {
		t.Errorf("completed lifecycle still open: %+v", open)
	}
}

func TestDittoLifecycleItemKeepsOneRowAndAccumulatesMinutes(t *testing.T) {
	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, time.Now(), []string{
		"[08:00:00] DONE ship the fix @10m",
	})
	InvalidateLedgerCaches()

	item, ok := lastLifecycleItem()
	if !ok {
		t.Fatal("expected a latest lifecycle item")
	}
	if err := dittoLifecycleItem(item, 5); err != nil {
		t.Fatalf("DONE Ditto: %v", err)
	}
	lines := readLedgerLines()
	if len(lines) != 1 || lines[0] != "[08:00:00] DOING ship the fix @10m" {
		t.Fatalf("DONE Ditto produced %q", lines)
	}

	item, ok = lastLifecycleItem()
	if !ok {
		t.Fatal("expected the DOING item after Ditto")
	}
	if err := dittoLifecycleItem(item, 7); err != nil {
		t.Fatalf("DOING Ditto: %v", err)
	}
	lines = readLedgerLines()
	if len(lines) != 1 || lines[0] != "[08:00:00] DOING ship the fix @17m" {
		t.Fatalf("repeated Ditto produced %q", lines)
	}

	if err := completePlannedItem(OpenItem{Category: "DOING", Text: "ship the fix @17m", LineIndex: 0}); err != nil {
		t.Fatalf("complete Ditto item: %v", err)
	}
	if got := parseEntryMins(readLedgerLines()[0]); got != 17 {
		t.Errorf("aggregated duration = %d, want 17", got)
	}
}

func TestLastDoingItemDoesNotResurrectDone(t *testing.T) {
	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, time.Now(), []string{
		"[08:00:00] DOING active work @10m",
		"[09:00:00] DONE completed work @20m",
	})
	InvalidateLedgerCaches()

	item, ok := lastDoingItem()
	if !ok || item.Category != "DOING" || item.Text != "active work @10m" {
		t.Fatalf("lastDoingItem() = %+v, %v; want active DOING", item, ok)
	}
}

func TestCompletePlannedEndpointResolvesLifecycle(t *testing.T) {
	withTempDunnitDir(t)
	writeLedgerLinesForDate(t, time.Now(), []string{
		"[08:00:00] DOING abandon the experiment @7m",
	})
	InvalidateLedgerCaches()

	if err := completePlannedEndpoint(OpenItem{
		Category:  "DOING",
		Text:      "abandon the experiment @7m",
		LineIndex: 0,
	}, "WASTED", "abandon the experiment @7m"); err != nil {
		t.Fatalf("completePlannedEndpoint: %v", err)
	}
	if got := readLedgerLines(); len(got) != 1 || got[0] != "[08:00:00] WASTED abandon the experiment @7m (via DOING)" {
		t.Fatalf("endpoint transition = %v", got)
	}
	if open := getOpenItems(); len(open) != 0 {
		t.Fatalf("endpoint transition left item open: %+v", open)
	}
}

func TestLifecycleDurationMetadata(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		wantMins    int
		wantReplace string
	}{
		{"plain", "task @10m", 10, "task @15m"},
		{"hours", "task @2h", 120, "task @125m"},
		{"days", "task @4d", 4 * 24 * 60, "task @5765m"},
		{"completion marker", "task @10m (via DOING)", 10, "task @15m (via DOING)"},
		{"carry and marker", "task @10m (since 2026-09-01) (via DOING)", 10, "task @15m (since 2026-09-01) (via DOING)"},
		{"malformed", "task @xm (via DOING)", 0, "task @xm @5m (via DOING)"},
		{"missing", "task (since 2026-09-01)", 0, "task @5m (since 2026-09-01)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseEntryMins(tt.text); got != tt.wantMins {
				t.Errorf("parseEntryMins(%q) = %d, want %d", tt.text, got, tt.wantMins)
			}
			if got := incrementEntryMins(tt.text, 5); got != tt.wantReplace {
				t.Errorf("incrementEntryMins(%q) = %q, want %q", tt.text, got, tt.wantReplace)
			}
		})
	}
}
