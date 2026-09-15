package dun

import (
	"reflect"
	"testing"
	"time"
)

func TestSortRecurringItems(t *testing.T) {
	items := []RecurringItem{
		{Cadence: "monthly", Text: "month"},
		{Cadence: "daily", Time: "10:00", Text: "later"},
		{Cadence: "weekly", Text: "week"},
		{Cadence: "daily", Time: "09:00", Text: "earlier"},
	}

	sortRecurringItems(items)
	var got []string
	for _, item := range items {
		got = append(got, item.Text)
	}
	want := []string{"earlier", "later", "week", "month"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted item order = %v, want %v", got, want)
	}
}

func TestRecurringItemOccurrence(t *testing.T) {
	location := time.UTC
	cases := []struct {
		name string
		item RecurringItem
		now  time.Time
		want time.Time
		ok   bool
	}{
		{
			name: "daily timed item",
			item: RecurringItem{Cadence: "daily", Time: "09:30"},
			now:  time.Date(2026, time.September, 15, 8, 0, 0, 0, location),
			want: time.Date(2026, time.September, 15, 9, 30, 0, 0, location),
			ok:   true,
		},
		{
			name: "monthly day 31 clamps",
			item: RecurringItem{Cadence: "monthly", DayOfMonth: 31, Time: "09:30"},
			now:  time.Date(2026, time.September, 30, 8, 0, 0, 0, location),
			want: time.Date(2026, time.September, 30, 9, 30, 0, 0, location),
			ok:   true,
		},
		{
			name: "untimed item remains suggestion",
			item: RecurringItem{Cadence: "daily"},
			now:  time.Date(2026, time.September, 15, 8, 0, 0, 0, location),
			ok:   false,
		},
		{
			name: "invalid time ignored",
			item: RecurringItem{Cadence: "daily", Time: "25:00"},
			now:  time.Date(2026, time.September, 15, 8, 0, 0, 0, location),
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := recurringItemOccurrence(tc.item, tc.now)
			if ok != tc.ok || (ok && !got.Equal(tc.want)) {
				t.Fatalf("recurringItemOccurrence() = (%v, %v), want (%v, %v)", got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestDueForRecurringItemReminder(t *testing.T) {
	item := RecurringItem{Cadence: "daily", Time: "09:00"}
	location := time.UTC
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"at scheduled time", time.Date(2026, time.September, 15, 9, 0, 0, 0, location), true},
		{"within tolerance", time.Date(2026, time.September, 15, 9, 1, 30, 0, location), true},
		{"outside tolerance", time.Date(2026, time.September, 15, 9, 3, 0, 0, location), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dueForRecurringItemReminder(item, tc.now, 2*time.Minute); got != tc.want {
				t.Fatalf("dueForRecurringItemReminder() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDueForPostMeetingNudge(t *testing.T) {
	location := time.UTC
	meeting := RecurringMeeting{Cadence: "daily", Time: "10:00"}
	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"too soon", time.Date(2026, time.September, 15, 10, 15, 0, 0, location), false},
		{"in summary window", time.Date(2026, time.September, 15, 10, 30, 0, 0, location), true},
		{"window expired", time.Date(2026, time.September, 15, 10, 46, 0, 0, location), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dueForPostMeetingNudge(meeting, tc.now); got != tc.want {
				t.Fatalf("dueForPostMeetingNudge() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLastOccurrenceSupportsBiweeklyParity(t *testing.T) {
	now := time.Date(2026, time.September, 15, 10, 30, 0, 0, time.UTC)
	meeting := RecurringMeeting{
		Cadence: "biweekly-even",
		DOW:     int(time.Tuesday),
		Time:    "10:00",
	}
	want := time.Date(2026, time.September, 15, 10, 0, 0, 0, time.UTC)
	if got := lastOccurrence(meeting, now); !got.Equal(want) {
		t.Fatalf("lastOccurrence() = %v, want %v", got, want)
	}
}

func TestNextOccurrenceKeepsQuarterlyAnchor(t *testing.T) {
	meeting := RecurringMeeting{
		Cadence:    "quarterly",
		DayOfMonth: 15,
		Time:       "10:00",
		AnchorDate: "2026-09-15",
	}
	now := time.Date(2026, time.October, 1, 8, 0, 0, 0, time.UTC)
	want := time.Date(2026, time.December, 15, 10, 0, 0, 0, time.UTC)
	if got := nextOccurrence(meeting, now); !got.Equal(want) {
		t.Fatalf("nextOccurrence() = %v, want %v", got, want)
	}
}

func TestSortRecurringMeetings(t *testing.T) {
	meetings := []RecurringMeeting{
		{Cadence: "quarterly", Tag: "#q"},
		{Cadence: "weekly", Tag: "#w"},
		{Cadence: "monthly", Tag: "#m"},
		{Cadence: "daily", Tag: "#d"},
		{Cadence: "biweekly-odd", Tag: "#o"},
	}
	sortRecurringMeetings(meetings)
	var got []string
	for _, meeting := range meetings {
		got = append(got, meeting.Tag)
	}
	want := []string{"#d", "#w", "#o", "#m", "#q"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted meeting order = %v, want %v", got, want)
	}
}

func TestMeetingCadenceOptionsIncludeExplicitBiweeklyAndQuarterlyChoices(t *testing.T) {
	want := []string{"biweekly-odd", "biweekly-even", "quarterly"}
	for _, choice := range want {
		found := false
		for _, option := range meetingCadenceOptions {
			if option == choice {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("meetingCadenceOptions is missing %q", choice)
		}
	}
}
