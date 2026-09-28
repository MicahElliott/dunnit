package dun

import (
	"testing"
)

func recordSomedayTestActivity(t *testing.T, text, category string) {
	t.Helper()
	if err := recordActivity(text, category); err != nil {
		t.Fatal(err)
	}
}

func TestGatherSomedayItems_ListsUnhandled(t *testing.T) {
	withTempDunnitDir(t)

	recordSomedayTestActivity(t, "clean the garage", "SOMEDAY")
	recordSomedayTestActivity(t, "read that book", "SOMEDAY")

	items := gatherSomedayItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 SOMEDAY items, got %d: %v", len(items), items)
	}
}

func TestGatherSomedayItems_DeduplicatesLogicalItems(t *testing.T) {
	withTempDunnitDir(t)

	recordSomedayTestActivity(t, "plan the launch #work", "SOMEDAY")
	recordSomedayTestActivity(t, "plan the launch #work", "SOMEDAY")
	recordSomedayTestActivity(t, "read the proposal #work", "SOMEDAY")

	items := gatherSomedayItems()
	if len(items) != 2 {
		t.Fatalf("expected duplicate SOMEDAY rows to collapse, got %d: %v", len(items), items)
	}
}

func TestGatherSomedayItems_ExcludesPromoted(t *testing.T) {
	withTempDunnitDir(t)

	recordSomedayTestActivity(t, "clean the garage", "SOMEDAY")
	item := OpenItem{Category: "SOMEDAY", Text: "clean the garage"}
	if err := promoteSomedayItem(item, "TODO"); err != nil {
		t.Fatal(err)
	}

	items := gatherSomedayItems()
	if len(items) != 0 {
		t.Fatalf("expected promoted SOMEDAY item to no longer appear, got %v", items)
	}

	// Promoting should have logged a fresh TODO.
	todos := getOpenItems()
	found := false
	for _, o := range todos {
		if o.Category == "TODO" && o.Text == "clean the garage" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected promoted item to appear as an open TODO, got %v", todos)
	}
}

func TestGatherSomedayItems_ExcludesDiscarded(t *testing.T) {
	withTempDunnitDir(t)

	recordSomedayTestActivity(t, "clean the garage", "SOMEDAY")
	if err := discardSomedayItem(OpenItem{Category: "SOMEDAY", Text: "clean the garage"}); err != nil {
		t.Fatal(err)
	}

	items := gatherSomedayItems()
	if len(items) != 0 {
		t.Fatalf("expected discarded SOMEDAY item to no longer appear, got %v", items)
	}
}
