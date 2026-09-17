package dun

import (
	"path/filepath"
	"strconv"
	"time"
)

// weekMonthInfo computes the (year, month abbreviation, ISO week) for a
// given date, where the month is determined by that date's ISO week's
// *start* (Monday), not the date's own calendar month. This fixes the
// bug where weeks spanning month boundaries create two separate
// directories (e.g. 2026/w36-Aug and 2026/w36-Sep for the same week 36).
//
// Returns (year, moname, week) ready for use in path construction.
func weekMonthInfo(date time.Time) (int, string, int) {
	yr, wk := date.ISOWeek()
	// Compute the Monday of this ISO week. ISOWeek returns the year and
	// week number; to find Monday, we need to walk backward from the
	// given date to its Monday.
	// time.Weekday: Monday=1, ..., Sunday=0. If date is Monday, diff=0.
	weekday := date.Weekday()
	if weekday == time.Sunday {
		weekday = 7 // Sunday is 0; treat it as 7 for calculation
	}
	daysBack := int(weekday) - 1 // Monday is 1, so Mon needs 0 days back
	monday := date.AddDate(0, 0, -daysBack)

	// Now use the Monday's month for the directory name.
	moname := monday.Format("Jan")
	// But we need the year/week from the original date (which may differ
	// from Monday's if the date is in a different year's week -- rare but
	// possible at year boundaries). Actually, ISOWeek already handles that,
	// so stick with the computed yr, wk.
	return yr, moname, wk
}

// ledgerDirFor returns the new year/<month>/w<week>/ directory path for
// the given ISO year/week and month abbreviation. The month is now
// always the week's start (Monday), not the individual date's month.
func ledgerDirFor(yr, wk int, moname string) string {
	return filepath.Join(DunnitDir(), strconv.Itoa(yr), moname, "w"+strconv.Itoa(wk))
}

// ledgerPathFor returns the full ledger file path for the given date,
// computing the correct directory via weekMonthInfo and filename from
// the date itself.
func ledgerPathFor(date time.Time) (dir, path string) {
	yr, moname, wk := weekMonthInfo(date)
	dir = ledgerDirFor(yr, wk, moname)
	path = filepath.Join(dir, "ledger-"+date.Format("Mon-20060102")+".txt")
	return dir, path
}

// weeklyReportPath returns the save path for a weekly Review report for
// the week containing anchor, themed for theme. Now nested inside the
// week's directory: <year>/<month>/w<week>/review-week-<period>-<date>[-<theme>].md
func weeklyReportPath(anchor time.Time, theme string) string {
	yr, moname, wk := weekMonthInfo(anchor)
	dir := ledgerDirFor(yr, wk, moname)
	filename := reportFilename("review-week", reviewReportDateToken(periodWeek, anchor), theme, time.Now())
	return filepath.Join(dir, filename)
}

// weeklyReportPathForKind returns the save path for a non-Review report
// covering the week containing anchor, such as a Standup or Status report.
func weeklyReportPathForKind(kind string, anchor, generated time.Time) string {
	yr, moname, wk := weekMonthInfo(anchor)
	dir := ledgerDirFor(yr, wk, moname)
	filename := reportFilename(kind, "w"+strconv.Itoa(wk), "", generated)
	return filepath.Join(dir, filename)
}

// monthlyReportPath returns the save path for a monthly Review report for
// the month containing anchor, themed for theme. Moved to:
// <year>/<month>/review-month-<token>-<theme>.md
func monthlyReportPath(anchor time.Time, theme string) string {
	yr := anchor.Year()
	moname := anchor.Format("Jan")
	token := anchor.Format("200601") // YYYYMM format
	dir := filepath.Join(DunnitDir(), strconv.Itoa(yr), moname)

	filename := reportFilename("review-month", token, theme, time.Now())
	return filepath.Join(dir, filename)
}

// quarterlyReportPath returns the save path for a quarterly Review report.
// These stay flat under the year directory: <year>/review-quarter-<token>-<theme>.md
func quarterlyReportPath(anchor time.Time, theme string) string {
	yr := anchor.Year()
	q := quarterOf(anchor)
	token := strconv.Itoa(yr) + "Q" + strconv.Itoa(q)

	filename := reportFilename("review-quarter", token, theme, time.Now())
	return filepath.Join(DunnitDir(), strconv.Itoa(yr), filename)
}

// yearlyReportPath returns the save path for a yearly Review report.
// These stay flat under the year directory: <year>/review-year-<token>-<theme>.md
func yearlyReportPath(anchor time.Time, theme string) string {
	yr := anchor.Year()
	token := strconv.Itoa(yr)

	filename := reportFilename("review-year", token, theme, time.Now())
	return filepath.Join(DunnitDir(), strconv.Itoa(yr), filename)
}
