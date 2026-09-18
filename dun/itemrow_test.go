package dun

import (
	"testing"
	"time"
)

func TestSplitTrailingMeta(t *testing.T) {
	cases := map[string][2]string{
		"Fix the login bug":                    {"Fix the login bug", ""},
		"walk the dog ~15m":                    {"walk the dog", " ~15m"},
		"long task ~2h":                        {"long task", " ~2h"},
		"multi-day task ~4d":                   {"multi-day task", " ~4d"},
		"finish the report s/2026-08-28":       {"finish the report", " s/2026-08-28"},
		"finish the report (since 2026-08-28)": {"finish the report", " (since 2026-08-28)"},
		"todo s/2026-08-28":                    {"todo", " s/2026-08-28"},
		"todo ~10m (via DOING)":                {"todo", " ~10m (via DOING)"},
	}
	for in, want := range cases {
		core, meta := splitTrailingMeta(in)
		if core != want[0] || meta != want[1] {
			t.Errorf("splitTrailingMeta(%q) = (%q, %q), want (%q, %q)", in, core, meta, want[0], want[1])
		}
	}
}

func TestOpenItemDisplayTextCollapsesSinceMarkers(t *testing.T) {
	since := time.Now().Format("2006-01-02")
	input := "finish the report (since " + since + ") (since " + since + ")"
	want := "finish the report s/" + since
	if got := openItemDisplayText(input); got != want {
		t.Fatalf("openItemDisplayText(%q) = %q, want %q", input, got, want)
	}
}

func TestDisplayMetadataToken(t *testing.T) {
	tests := []struct {
		token, wantLabel, wantTooltip string
	}{
		{" ~30m", " ⏱︎30m", "Spent 30 mins"},
	}
	for _, tt := range tests {
		label, tooltip := displayMetadataToken(tt.token)
		if label != tt.wantLabel || tooltip != tt.wantTooltip {
			t.Errorf("displayMetadataToken(%q) = (%q, %q), want (%q, %q)",
				tt.token, label, tooltip, tt.wantLabel, tt.wantTooltip)
		}
	}
}

func TestDisplayMetadataTokenUsesAgeIndicator(t *testing.T) {
	since := time.Now().AddDate(0, 0, -5)
	token := " s/" + since.Format("2006-01-02")
	label, tooltip := displayMetadataToken(token)
	wantLabel := " 🟠5d"
	wantTooltip := "Open for 5 days"
	if label != wantLabel || tooltip != wantTooltip {
		t.Errorf("displayMetadataToken(%q) = (%q, %q), want (%q, %q)",
			token, label, tooltip, wantLabel, wantTooltip)
	}
}

func TestAgeIndicator(t *testing.T) {
	tests := []struct {
		days int
		want string
	}{
		{0, "🟡"},
		{1, "🟡"},
		{3, "🟡"},
		{4, "🟠"},
		{7, "🟠"},
		{8, "🔴"},
		{30, "🔴"},
	}
	for _, tt := range tests {
		if got := ageIndicator(tt.days); got != tt.want {
			t.Errorf("ageIndicator(%d) = %q, want %q", tt.days, got, tt.want)
		}
	}
}

func TestStripDisplayMetadata(t *testing.T) {
	cases := map[string]string{
		"finish the report (via DOING)":      "finish the report",
		"finish the report ~20m (via DOING)": "finish the report",
		"carry the task s/2026-09-01":        "carry the task",
		"ordinary text with (parentheses)":   "ordinary text with (parentheses)",
	}
	for input, want := range cases {
		if got := stripDisplayMetadata(input); got != want {
			t.Errorf("stripDisplayMetadata(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSplitPrimaryTagUsesLastTag(t *testing.T) {
	tag, body := splitPrimaryTag("Fumbled through #snap tab to work #12345")
	if tag != "#12345" || body != "Fumbled through #snap tab to work" {
		t.Fatalf("splitPrimaryTag = (%q, %q), want (#12345, body without primary)", tag, body)
	}
}

func TestSortItemsByPrimaryTagThenTime(t *testing.T) {
	items := []OpenItem{
		{Text: "no tag here", Time: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)},
		{Text: "worked #snap later", Time: time.Date(2026, 1, 1, 12, 2, 0, 0, time.UTC)},
		{Text: "worked #snap earlier", Time: time.Date(2026, 1, 1, 12, 1, 0, 0, time.UTC)},
		{Text: "#alpha first", Time: time.Date(2026, 1, 1, 12, 3, 0, 0, time.UTC)},
	}
	sortItemsByPrimaryTag(items)
	want := []string{"#alpha first", "worked #snap earlier", "worked #snap later", "no tag here"}
	for i, w := range want {
		if items[i].Text != w {
			t.Errorf("position %d: got %q, want %q", i, items[i].Text, w)
		}
	}
}

func TestSplitExcludedTagItems(t *testing.T) {
	items := []OpenItem{
		{Text: "#work do the thing"},
		{Text: "#home buy milk"},
		{Text: "no tag"},
	}
	visible, excluded := splitExcludedTagItems(items, []string{"#home"})
	if len(visible) != 2 || len(excluded) != 1 {
		t.Fatalf("got visible=%d excluded=%d, want 2/1", len(visible), len(excluded))
	}
	if excluded[0].Text != "#home buy milk" {
		t.Errorf("excluded[0] = %q, want #home item", excluded[0].Text)
	}
}
