package dun

import "testing"

func TestDoingCategoryIsVisiblePlannedAndTimeTrackable(t *testing.T) {
	var doing Category
	for _, category := range Categories {
		if category.Code == "DOING" {
			doing = category
			break
		}
	}
	if doing.Code == "" {
		t.Fatal("DOING is missing from the category registry")
	}
	if doing.Group != "plan" || doing.EODOnly {
		t.Fatalf("DOING category = %+v, want visible plan category", doing)
	}
	if !CategoryExists("DOING") || CategoryExists(legacyOngoingCategory) {
		t.Fatal("expected DOING to be current and ONGOING to remain legacy-only")
	}
	if !IsTimeTrackable("DOING") {
		t.Error("DOING should support cumulative minutes")
	}

	labels := CategoryLabelsForGroup(Config{}, "plan")
	found := false
	for _, label := range labels {
		if label == doing.Label() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Plan picker labels %v do not include %q", labels, doing.Label())
	}
}
