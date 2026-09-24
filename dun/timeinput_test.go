package dun

import (
	"testing"
	"time"
)

func TestParseTimeInput(t *testing.T) {
	for _, tc := range []struct {
		name       string
		input      string
		wantHour   int
		wantMinute int
		wantOK     bool
	}{
		{name: "24 hour with leading zero", input: "06:30", wantHour: 6, wantMinute: 30, wantOK: true},
		{name: "24 hour without leading zero", input: "6:30", wantHour: 6, wantMinute: 30, wantOK: true},
		{name: "legacy seconds", input: "06:30:00", wantHour: 6, wantMinute: 30, wantOK: true},
		{name: "compact am", input: "6a", wantHour: 6, wantOK: true},
		{name: "full am", input: "6am", wantHour: 6, wantOK: true},
		{name: "compact pm with minutes", input: "6:30p", wantHour: 18, wantMinute: 30, wantOK: true},
		{name: "digit compact pm", input: "630p", wantHour: 18, wantMinute: 30, wantOK: true},
		{name: "digit compact full pm", input: "1230pm", wantHour: 12, wantMinute: 30, wantOK: true},
		{name: "spaced full pm", input: " 6:30 PM ", wantHour: 18, wantMinute: 30, wantOK: true},
		{name: "midnight", input: "midnight", wantOK: true},
		{name: "noon", input: "noon", wantHour: 12, wantOK: true},
		{name: "12am", input: "12am", wantOK: true},
		{name: "12pm", input: "12pm", wantHour: 12, wantOK: true},
		{name: "bare hour is ambiguous", input: "6", wantOK: false},
		{name: "digit only is ambiguous", input: "630", wantOK: false},
		{name: "invalid 24 hour", input: "24:00", wantOK: false},
		{name: "invalid 12 hour", input: "13pm", wantOK: false},
		{name: "invalid minute", input: "6:60", wantOK: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hour, minute, ok := parseTimeInput(tc.input)
			if hour != tc.wantHour || minute != tc.wantMinute || ok != tc.wantOK {
				t.Fatalf("parseTimeInput(%q) = (%d, %d, %v), want (%d, %d, %v)", tc.input, hour, minute, ok, tc.wantHour, tc.wantMinute, tc.wantOK)
			}
		})
	}
}

func TestWithinWorkHoursDefaultsBlankBounds(t *testing.T) {
	now := time.Date(2026, time.September, 23, 9, 0, 0, 0, time.Local)
	if !withinWorkHours(Config{}, now) {
		t.Fatal("withinWorkHours(Config{}, 09:00) = false, want true using default bounds")
	}
}

func TestCanonicalTimeInput(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "6a", want: "06:00"},
		{input: "6:30 pm", want: "18:30"},
		{input: "630p", want: "18:30"},
		{input: "noon", want: "12:00"},
		{input: "midnight", want: "00:00"},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := canonicalTimeInput(tc.input)
			if err != nil || got != tc.want {
				t.Fatalf("canonicalTimeInput(%q) = (%q, %v), want (%q, nil)", tc.input, got, err, tc.want)
			}
		})
	}
}

func TestWithinWorkHoursUsesTimeShortcuts(t *testing.T) {
	cfg := Config{DayStart: "6a", DayEnd: "6pm"}
	for _, tc := range []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "at start", now: time.Date(2026, time.September, 21, 6, 0, 0, 0, time.UTC), want: true},
		{name: "before start", now: time.Date(2026, time.September, 21, 5, 59, 0, 0, time.UTC), want: false},
		{name: "at end", now: time.Date(2026, time.September, 21, 18, 0, 0, 0, time.UTC), want: true},
		{name: "after end", now: time.Date(2026, time.September, 21, 18, 1, 0, 0, time.UTC), want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := withinWorkHours(cfg, tc.now); got != tc.want {
				t.Fatalf("withinWorkHours(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}
