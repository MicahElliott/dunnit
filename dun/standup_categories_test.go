package dun

import (
	"strings"
	"testing"
	"time"
)

func TestStandupWindowStartLabel(t *testing.T) {
	location := time.FixedZone("test", -7*60*60)
	cases := []struct {
		name string
		when time.Time
		want string
	}{
		{
			name: "midnight is named explicitly",
			when: time.Date(2026, time.September, 15, 0, 0, 0, 0, location),
			want: "start of Tue",
		},
		{
			name: "timed boundary keeps clock format",
			when: time.Date(2026, time.September, 15, 9, 30, 0, 0, location),
			want: "Tue 09:30",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := standupWindowStartLabel(tc.when); got != tc.want {
				t.Fatalf("standupWindowStartLabel() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStandupWindowStartFallbackIsStartOfSourceDay(t *testing.T) {
	location := time.FixedZone("test", -7*60*60)
	now := time.Date(2026, time.September, 16, 9, 0, 0, 0, location)
	want := time.Date(2026, time.September, 15, 0, 0, 0, 0, location)
	if got := standupWindowStart(Config{}, now); !got.Equal(want) {
		t.Fatalf("standupWindowStart() = %v, want %v", got, want)
	}
}

func TestStandupPromptInputIncludesOpenPlanItems(t *testing.T) {
	now := time.Date(2026, time.September, 15, 9, 0, 0, 0, time.UTC)
	input := standupPromptInput(
		[]string{"finished the release"},
		[]OpenItem{
			{Category: "TODO", Text: "write the follow-up"},
			{Category: "DOING", Text: "review the rollout"},
			{Category: "GOAL", Text: "improve reliability"},
		},
		now,
	)
	for _, want := range []string{
		"Completed/notable items from yesterday:",
		"finished the release",
		"Currently open TODOs/DOING/GOALs (candidates for \"today\"):",
		"[TODO] write the follow-up",
		"[DOING] review the rollout",
		"[GOAL] improve reliability",
		"write the follow-up",
		"review the rollout",
		"improve reliability",
	} {
		if !strings.Contains(input, want) {
			t.Errorf("standup prompt input does not contain %q: %q", want, input)
		}
	}
}

func TestStandupEditableInputRoundTripsAllReportInputs(t *testing.T) {
	lines := []string{"finished the release", "learned something useful"}
	openItems := []OpenItem{
		{Category: "TODO", Text: "write the follow-up"},
		{Category: "DOING", Text: "review the rollout s/2026-09-14"},
	}
	text := formatStandupEditableInput(lines, openItems)
	gotLines, gotOpenItems := parseStandupEditableInput(text)
	if strings.Join(gotLines, "\x00") != strings.Join(lines, "\x00") {
		t.Fatalf("editable completed items = %v, want %v", gotLines, lines)
	}
	if len(gotOpenItems) != len(openItems) {
		t.Fatalf("editable open items = %+v, want %+v", gotOpenItems, openItems)
	}
	wantOpenText := []string{"write the follow-up", "review the rollout"}
	for i, want := range openItems {
		if gotOpenItems[i].Category != want.Category || gotOpenItems[i].Text != wantOpenText[i] {
			t.Fatalf("editable open item %d = %+v, want category %q and text %q", i, gotOpenItems[i], want.Category, wantOpenText[i])
		}
	}
}

func TestStandupActivityLabelNamesFridayAfterWeekend(t *testing.T) {
	location := time.FixedZone("test", -7*60*60)
	monday := time.Date(2026, time.September, 21, 9, 0, 0, 0, location)
	tuesday := monday.AddDate(0, 0, 1)
	if got := standupActivityLabel(monday); got != "Friday" {
		t.Fatalf("standupActivityLabel(Monday) = %q, want Friday", got)
	}
	if got := standupActivityLabel(tuesday); got != "yesterday" {
		t.Fatalf("standupActivityLabel(Tuesday) = %q, want yesterday", got)
	}
}

func TestStandupOpenItemsForReportFiltersExcludedTags(t *testing.T) {
	withTempDunnitDir(t)
	cfg := LoadConfig()
	cfg.ReportExcludeTags = []string{"#home"}
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	recordActivity("private errand #home", "TODO")
	recordActivity("ship work #work", "DOING")

	items := standupOpenItemsForReport()
	if len(items) != 1 || items[0].Text != "ship work #work" {
		t.Fatalf("standup open items = %+v, want only the non-excluded item", items)
	}
}

func TestParseLedgerLineTimeAcceptsMinuteAndLegacySecondStamps(t *testing.T) {
	date := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.Local)
	tests := []struct {
		line    string
		wantSec int
	}{
		{"[09:15] DONE minute stamp", 0},
		{"[09:15:42] DONE legacy stamp", 42},
	}
	for _, tt := range tests {
		got, ok := parseLedgerLineTime(tt.line, date)
		if !ok || got.Hour() != 9 || got.Minute() != 15 || got.Second() != tt.wantSec {
			t.Errorf("parseLedgerLineTime(%q) = %v, %v; want 09:15:%02d",
				tt.line, got, ok, tt.wantSec)
		}
	}
}

func TestStandupCategories_IncludesEndAndHiliteExcludingInternalMarkers(t *testing.T) {
	want := map[string]bool{
		// end (excluding ONGOING)
		"DONE": true, "HANDLED": true, "FAIL": true, "WASTED": true,
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
