package dun

import "testing"

func TestSplitTrailingMeta(t *testing.T) {
	cases := map[string][2]string{
		"Fix the login bug":                    {"Fix the login bug", ""},
		"walk the dog @15m":                    {"walk the dog", " @15m"},
		"finish the report (since 2026-08-28)": {"finish the report", " (since 2026-08-28)"},
		"old todo \u26a0 4d":                   {"old todo", " \u26a0 4d"},
		"todo (since 2026-08-28) \u26a0 4d":    {"todo", " (since 2026-08-28) \u26a0 4d"},
	}
	for in, want := range cases {
		core, meta := splitTrailingMeta(in)
		if core != want[0] || meta != want[1] {
			t.Errorf("splitTrailingMeta(%q) = (%q, %q), want (%q, %q)", in, core, meta, want[0], want[1])
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
