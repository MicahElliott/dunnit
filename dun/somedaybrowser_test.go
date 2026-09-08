package dun

import (
	"testing"
)

func TestGatherSomedayItems_ListsUnhandled(t *testing.T) {
	withTempDunnitDir(t)

	recordActivity("clean the garage", "SOMEDAY")
	recordActivity("read that book", "SOMEDAY")

	items := gatherSomedayItems()
	if len(items) != 2 {
		t.Fatalf("expected 2 SOMEDAY items, got %d: %v", len(items), items)
	}
}

func TestGatherSomedayItems_ExcludesPromoted(t *testing.T) {
	withTempDunnitDir(t)

	recordActivity("clean the garage", "SOMEDAY")
	item := OpenItem{Category: "SOMEDAY", Text: "clean the garage"}
	promoteSomedayItem(item, "TODO")

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

	recordActivity("clean the garage", "SOMEDAY")
	discardSomedayItem(OpenItem{Category: "SOMEDAY", Text: "clean the garage"})

	items := gatherSomedayItems()
	if len(items) != 0 {
		t.Fatalf("expected discarded SOMEDAY item to no longer appear, got %v", items)
	}
}
