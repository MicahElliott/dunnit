package dun

import "time"

// isUSFederalHoliday reports whether date is one of the 11 US federal
// holidays (5 U.S.C. 6103). Computed algorithmically rather than a
// per-year hardcoded date table, since most of these are "Nth weekday
// of month" rules (not fixed calendar dates) and this way correctly
// covers any year without needing yearly maintenance. Time-of-day is
// ignored (compares only year/month/day).
//
// Deliberately does NOT observe the "if it falls on a weekend, shift
// to the nearest weekday" rule some federal-employee calendars use --
// gating Dunnit's nudges off already treats every Saturday/Sunday as
// off regardless (see withinWorkHours), so an in-lieu-of weekday
// shift would be redundant here; if a holiday lands on a weekend, that
// day was already going to be off.
func isUSFederalHoliday(date time.Time) bool {	y := date.Year()
	m := date.Month()
	d := date.Day()

	switch {
	case m == time.January && d == 1:
		return true // New Year's Day
	case m == time.January && d == nthWeekdayOfMonth(y, m, time.Monday, 3):
		return true // Birthday of Martin Luther King, Jr. (3rd Monday)
	case m == time.February && d == nthWeekdayOfMonth(y, m, time.Monday, 3):
		return true // Washington's Birthday / Presidents' Day (3rd Monday)
	case m == time.May && d == lastWeekdayOfMonth(y, m, time.Monday):
		return true // Memorial Day (last Monday)
	case m == time.June && d == 19:
		return true // Juneteenth National Independence Day
	case m == time.July && d == 4:
		return true // Independence Day
	case m == time.September && d == nthWeekdayOfMonth(y, m, time.Monday, 1):
		return true // Labor Day (1st Monday)
	case m == time.October && d == nthWeekdayOfMonth(y, m, time.Monday, 2):
		return true // Columbus Day (2nd Monday)
	case m == time.November && d == 11:
		return true // Veterans Day
	case m == time.November && d == nthWeekdayOfMonth(y, m, time.Thursday, 4):
		return true // Thanksgiving Day (4th Thursday)
	case m == time.December && d == 25:
		return true // Christmas Day
	}
	return false
}

// nthWeekdayOfMonth returns the day-of-month of the nth occurrence
// (1-based) of weekday in the given year/month.
func nthWeekdayOfMonth(year int, month time.Month, weekday time.Weekday, n int) int {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	daysUntil := (int(weekday) - int(first.Weekday()) + 7) % 7
	return 1 + daysUntil + (n-1)*7
}

// lastWeekdayOfMonth returns the day-of-month of the last occurrence
// of weekday in the given year/month.
func lastWeekdayOfMonth(year int, month time.Month, weekday time.Weekday) int {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	last := time.Date(year, month, lastDay, 0, 0, 0, 0, time.UTC)
	daysBack := (int(last.Weekday()) - int(weekday) + 7) % 7
	return lastDay - daysBack
}
