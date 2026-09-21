package dun

import "testing"

func TestAllCategoriesHaveHelpText(t *testing.T) {
	for _, category := range Categories {
		if category.Help == "" {
			t.Errorf("%s has no help text", category.Code)
		}
	}
}

func TestEmojiForCode(t *testing.T) {
	if got := EmojiForCode("TODO"); got != "📌" {
		t.Errorf("EmojiForCode(TODO) = %q, want pushpin", got)
	}
	if got := EmojiForCode("missing"); got != "" {
		t.Errorf("EmojiForCode(missing) = %q, want empty", got)
	}
}

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
	if len(labels) < 2 || labels[0] != doing.Label() || labels[1] != EmojiForCode("TODO")+" TODO" {
		t.Errorf("Plan picker labels = %v, want DOING before TODO", labels)
	}
}
