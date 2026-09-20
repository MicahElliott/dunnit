package dun

import (
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
			want: "Tue midnight",
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
