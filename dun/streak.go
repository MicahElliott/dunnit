package dun

import (
	"fmt"
	"os"
	"time"

	"fyne.io/fyne/v2/widget"
)

// ledgerHasEntries reports whether the ledger file for the given date
// exists and has at least one non-empty line (an empty file can be
// left behind by some code paths, e.g. directory creation without an
// actual entry -- treat that as "nothing logged").
func ledgerHasEntries(date time.Time) bool {
	_, fname := ledgerPathFor(date)
	info, err := os.Stat(fname)
	if err != nil {
		return false
	}
	return info.Size() > 0
}

// CurrentStreak (FR-28) computes the number of consecutive workdays
// (Mon-Fri, weekends skipped rather than breaking the streak) ending
// with the most recently completed workday that has at least one
// ledger entry, counting backward from today. Kept deliberately
// simple per Micah's "keep it simple for now" -- no persisted streak
// state, no punitive framing, just a read of existing ledger files
// each time it's asked for.
//
// If today already has an entry, today counts too. If today has no
// entry yet, today doesn't break the streak (it's still in progress
// -- the streak is based on the most recent day that's actually
// over/checkable), so we start counting from the most recent workday
// with an entry.
func CurrentStreak() int {
	streak := 0
	day := time.Now()
	// Skip forward-in-time-sense: start at today, walk backward one
	// day at a time, skipping weekends, until we hit a workday with no
	// entry.
	first := true
	for {
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			day = day.AddDate(0, 0, -1)
			continue
		}
		if ledgerHasEntries(day) {
			streak++
			day = day.AddDate(0, 0, -1)
			first = false
			continue
		}
		if first {
			// Today (or the most recent workday checked) has no
			// entry yet -- don't count it, but don't treat it as a
			// broken streak either; just move on to check the prior
			// workday.
			day = day.AddDate(0, 0, -1)
			first = false
			continue
		}
		break
	}
	return streak
}

// streakLabel returns a low-friction, positive-only widget showing
// the current logging streak (FR-28) -- blank/no widget content if
// the streak is 0, since there's no punitive framing for a broken or
// nonexistent streak, only encouragement when there's something to
// show.
func streakLabel() *widget.Label {
	n := CurrentStreak()
	if n <= 0 {
		return widget.NewLabel("")
	}
	plural := ""
	if n != 1 {
		plural = "s"
	}
	return widget.NewLabel(fmt.Sprintf("\U0001F525 %d consecutive workday%s logged \u2014 keep it up!", n, plural))
}
