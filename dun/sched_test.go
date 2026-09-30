package dun

import (
	"testing"
	"time"
)

func TestParseWeekdayAcceptsShortAndLegacyNames(t *testing.T) {
	cases := map[string]time.Weekday{
		"Sun":       time.Sunday,
		"Mon":       time.Monday,
		"Tue":       time.Tuesday,
		"Wed":       time.Wednesday,
		"Thu":       time.Thursday,
		"Fri":       time.Friday,
		"Sat":       time.Saturday,
		"Wednesday": time.Wednesday,
	}
	for input, want := range cases {
		got, ok := parseWeekday(input)
		if !ok || got != want {
			t.Errorf("parseWeekday(%q) = (%v, %v), want (%v, true)", input, got, ok, want)
		}
	}
}
