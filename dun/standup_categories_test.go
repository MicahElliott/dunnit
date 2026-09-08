package dun

import "testing"

func TestStandupCategories_IncludesEndAndHiliteExcludingInternalMarkers(t *testing.T) {
	want := map[string]bool{
		// end (excluding ONGOING)
		"DONE": true, "FAIL": true, "WASTED": true,
		// hilite (excluding EODOnly SUMMARY/PRODUCTIVITY/MEETING_HOURS)
		"TIL": true, "KUDOS": true, "WIN": true, "PSA": true, "OVERCOMING": true,
		"INNOVATION": true, "LEADERSHIP": true,
		"IMPACT": true, "MILESTONE": true, "CAREER": true,
	}
	got := buildStandupCategories()
	if len(got) != len(want) {
		t.Errorf("buildStandupCategories() = %v (len %d), want %v (len %d)", got, len(got), want, len(want))
	}
	for cat := range want {
		if !got[cat] {
			t.Errorf("expected %q to be included in standupCategories", cat)
		}
	}
	for _, excluded := range []string{"ONGOING", "SUMMARY", "PRODUCTIVITY", "MEETING_HOURS", "TODO", "GOAL"} {
		if got[excluded] {
			t.Errorf("expected %q to be excluded from standupCategories", excluded)
		}
	}
}
