package dun

import (
	"testing"
)

func TestSplitTrailingMeta(t *testing.T) {
	cases := map[string][2]string{
		"Fix the login bug":                {"Fix the login bug", ""},
		"walk the dog @15m":                {"walk the dog", " @15m"},
		"long task @2h":                    {"long task", " @2h"},
		"multi-day task @4d":               {"multi-day task", " @4d"},
		"finish the report s/2026-08-28":   {"finish the report", " s/2026-08-28"},
		"old todo \u26a0\ufe0f4d":          {"old todo", " \u26a0\ufe0f4d"},
		"todo s/2026-08-28 \u26a0\ufe0f4d": {"todo", " s/2026-08-28 \u26a0\ufe0f4d"},
		"todo @10m (via DOING)":            {"todo", " @10m (via DOING)"},
	}
	for in, want := range cases {
		core, meta := splitTrailingMeta(in)
		if core != want[0] || meta != want[1] {
			t.Errorf("splitTrailingMeta(%q) = (%q, %q), want (%q, %q)", in, core, meta, want[0], want[1])
		}
	}
}

func TestDisplayMetadataToken(t *testing.T) {
	tests := []struct {
		token, wantLabel, wantTooltip string
	}{
		{" @30m", " ⏱30m", "Spent 30 mins"},
		{" s/2026-09-11", " 🌱09/11", "Created on 2026-09-11"},
		{" ⚠️5d", " ⚠️5d", "Open for 5 days"},
	}
	for _, tt := range tests {
		label, tooltip := displayMetadataToken(tt.token)
		if label != tt.wantLabel || tooltip != tt.wantTooltip {
			t.Errorf("displayMetadataToken(%q) = (%q, %q), want (%q, %q)",
				tt.token, label, tooltip, tt.wantLabel, tt.wantTooltip)
		}
	}
}

func TestStripDisplayMetadata(t *testing.T) {
	cases := map[string]string{
		"finish the report (via DOING)":      "finish the report",
		"finish the report @20m (via DOING)": "finish the report",
		"carry the task s/2026-09-01":        "carry the task",
		"old task ⚠️4d":                      "old task",
		"ordinary text with (parentheses)":   "ordinary text with (parentheses)",
	}
	for input, want := range cases {
		if got := stripDisplayMetadata(input); got != want {
			t.Errorf("stripDisplayMetadata(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSortItemsByLeadingTagUntaggedLast(t *testing.T) {
	items := []OpenItem{
		{Text: "no tag here"},
		{Text: "#beta second"},
		{Text: "#alpha first"},
		{Text: "another untagged"},
	}
	sortItemsByLeadingTag(items)
	want := []string{"#alpha first", "#beta second", "no tag here", "another untagged"}
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
