package dun

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ledgerHasEntries reports whether the given date has at least one ledger
// entry that is not excluded by the user's report-exclude tags. Excluded-only
// activity should not advance the streak shown in Daybook.
func ledgerHasEntries(date time.Time) bool {
	for _, entry := range AllLedgerEntries() {
		if dateOnly(entry.Date).Equal(dateOnly(date)) && !isExcludedStreakEntry(entry) {
			return true
		}
	}
	return false
}

func ledgerHasAnyEntries(date time.Time) bool {
	for _, entry := range AllLedgerEntries() {
		if dateOnly(entry.Date).Equal(dateOnly(date)) {
			return true
		}
	}
	return false
}

func isExcludedStreakEntry(entry LedgerEntry) bool {
	return lineHasExcludedTag(entry.Text, LoadConfig().ReportExcludeTags)
}

func filterExcludedStreakEntries(entries []LedgerEntry) []LedgerEntry {
	filtered := make([]LedgerEntry, 0, len(entries))
	for _, entry := range entries {
		if !isExcludedStreakEntry(entry) {
			filtered = append(filtered, entry)
		}
	}
	return filtered
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
	// Start at today, walk backward one day at a time, skipping weekends.
	// An untouched current day is still in progress and may be skipped;
	// an excluded-only day was handled but does not qualify for the streak,
	// so it breaks the streak.
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
		if first && !ledgerHasAnyEntries(day) {
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

const (
	maxStreakCallouts     = 2
	doneDaysForCallout    = 3
	donePerDayForCallout  = 6 // "more than five"
	hilitesForCallout     = 5
	peopleForCallout      = 7
	ticketsForCallout     = 5
	projectTagsForCallout = 4
	tilDaysForCallout     = 3
	durationForCallout    = 6 * 60
	winDaysForCallout     = 3
	hiliteKindsForCallout = 3
	repeatedTagForCallout = 5
	continuityForCallout  = 3
)

// ticketTagPattern accepts plain numeric markers and common project-key
// markers such as #SCRUM-12345. Requiring a hyphen before the number avoids
// treating ordinary word tags such as #planning as tickets.
var ticketTagPattern = regexp.MustCompile(`(?i)^#(?:[a-z][a-z0-9]*-)?[0-9]+$`)

func isTicketTag(tag string) bool {
	return ticketTagPattern.MatchString(tag)
}

func dateOnly(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func isWorkday(t time.Time) bool {
	return t.Weekday() != time.Saturday && t.Weekday() != time.Sunday
}

func inDateWindow(date, from, through time.Time) bool {
	date, from, through = dateOnly(date), dateOnly(from), dateOnly(through)
	return !date.Before(from) && !date.After(through)
}

func recentWorkdayRange(now time.Time, count int) (from, through time.Time) {
	through = dateOnly(now)
	for !isWorkday(through) {
		through = through.AddDate(0, 0, -1)
	}
	from = through
	for remaining := 1; remaining < count; remaining++ {
		from = from.AddDate(0, 0, -1)
		for !isWorkday(from) {
			from = from.AddDate(0, 0, -1)
		}
	}
	return from, through
}

func entryTags(entry LedgerEntry) []string {
	if len(entry.Tags) > 0 {
		return entry.Tags
	}
	return extractTags(entry.Text)
}

func entryPeople(entry LedgerEntry) []string {
	if len(entry.People) > 0 {
		return entry.People
	}
	return extractPeople(entry.Text)
}

func isHiliteEntry(entry LedgerEntry) bool {
	for _, category := range Categories {
		if category.Code == entry.Category {
			return category.Group == "hilite" && !category.EODOnly
		}
	}
	return false
}

func entriesOnDate(entries []LedgerEntry, date time.Time) []LedgerEntry {
	date = dateOnly(date)
	var result []LedgerEntry
	for _, entry := range entries {
		if dateOnly(entry.Date).Equal(date) {
			result = append(result, entry)
		}
	}
	return result
}

func distinctPeople(entries []LedgerEntry, from, through time.Time) map[string]bool {
	people := map[string]bool{}
	for _, entry := range entries {
		if !inDateWindow(entry.Date, from, through) {
			continue
		}
		for _, person := range entryPeople(entry) {
			people[personKey(person)] = true
		}
	}
	return people
}

func distinctTags(entries []LedgerEntry, from, through time.Time, tickets bool) map[string]string {
	tags := map[string]string{}
	for _, entry := range entries {
		if !inDateWindow(entry.Date, from, through) {
			continue
		}
		for _, tag := range entryTags(entry) {
			if isTicketTag(tag) != tickets {
				continue
			}
			key := strings.ToLower(tag)
			if _, ok := tags[key]; !ok {
				tags[key] = tag
			}
		}
	}
	return tags
}

func mostUsedTag(entries []LedgerEntry, from, through time.Time, minimum int) (string, bool) {
	entries = deduplicateCarryForwardEntries(entries)
	counts := map[string]int{}
	labels := map[string]string{}
	for _, entry := range entries {
		if !inDateWindow(entry.Date, from, through) {
			continue
		}
		for _, tag := range entryTags(entry) {
			if isTicketTag(tag) {
				continue
			}
			key := strings.ToLower(tag)
			counts[key]++
			labels[key] = tag
		}
	}
	for key, count := range counts {
		if count >= minimum {
			return labels[key], true
		}
	}
	return "", false
}

func doneCountOnDate(entries []LedgerEntry, date time.Time) int {
	count := 0
	for _, entry := range entriesOnDate(entries, date) {
		if entry.Category == "DONE" {
			count++
		}
	}
	return count
}

func hasConsecutiveDoneDays(entries []LedgerEntry, through time.Time) bool {
	from, through := recentWorkdayRange(through, doneDaysForCallout)
	for day := from; !day.After(through); day = day.AddDate(0, 0, 1) {
		if isWorkday(day) && doneCountOnDate(entries, day) < donePerDayForCallout {
			return false
		}
	}
	return true
}

func hasConsecutiveTagDays(entries []LedgerEntry, from, through time.Time, count int) (string, bool) {
	entries = deduplicateCarryForwardEntries(entries)
	for day := from; !day.After(through); day = day.AddDate(0, 0, 1) {
		if !isWorkday(day) {
			continue
		}
		for _, entry := range entriesOnDate(entries, day) {
			for _, tag := range entryTags(entry) {
				if isTicketTag(tag) {
					continue
				}
				run := 1
				for next := day.AddDate(0, 0, 1); run < count; next = next.AddDate(0, 0, 1) {
					for !isWorkday(next) {
						next = next.AddDate(0, 0, 1)
					}
					if next.After(through) {
						break
					}
					found := false
					for _, nextEntry := range entriesOnDate(entries, next) {
						for _, nextTag := range entryTags(nextEntry) {
							if strings.EqualFold(nextTag, tag) {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					if !found {
						break
					}
					run++
				}
				if run >= count {
					return tag, true
				}
			}
		}
	}
	return "", false
}

// streakCalloutCandidates returns positive signals that are currently true.
// The rules intentionally measure different kinds of progress: consistency,
// throughput, breadth, learning, time, and sustained focus. The ordinary
// TODO -> DOING -> DONE path is not a signal because it is normal daily flow.
func streakCalloutCandidates(entries []LedgerEntry, now time.Time, loggingStreak int) []string {
	entries = deduplicateCarryForwardEntries(filterExcludedStreakEntries(entries))
	weekFrom, weekThrough := weekStart(now), dateOnly(now)
	recentFrom, recentThrough := recentWorkdayRange(now, 5)
	var callouts []string

	if loggingStreak > 0 {
		callouts = append(callouts, fmt.Sprintf("🔥 %d consecutive workdays logged", loggingStreak))
	}
	if hasConsecutiveDoneDays(entries, now) {
		callouts = append(callouts, "🏁 3 workdays in a row with 6+ DONEs")
	}

	hiliteCount, winDays, tilDays, duration := 0, map[string]bool{}, map[string]bool{}, 0
	hiliteKinds := map[string]bool{}
	for _, entry := range entries {
		if !inDateWindow(entry.Date, weekFrom, weekThrough) || !isWorkday(entry.Date) {
			continue
		}
		if isHiliteEntry(entry) {
			hiliteCount++
			hiliteKinds[entry.Category] = true
		}
		if entry.Category == "WIN" {
			winDays[dateOnly(entry.Date).Format("2006-01-02")] = true
		}
		if entry.Category == "TIL" {
			tilDays[dateOnly(entry.Date).Format("2006-01-02")] = true
		}
		duration += entry.Mins
	}
	if hiliteCount >= hilitesForCallout {
		callouts = append(callouts, "✨ 5 Hilites this week")
	}
	if len(winDays) >= winDaysForCallout {
		callouts = append(callouts, "🏆 WINs on 3 days this week")
	}
	if len(tilDays) >= tilDaysForCallout {
		callouts = append(callouts, "🌱 Learned something on 3 days this week")
	}
	if duration >= durationForCallout {
		callouts = append(callouts, "⏱️ Logged 6h of explicit time this week")
	}
	if len(hiliteKinds) >= hiliteKindsForCallout {
		callouts = append(callouts, "🌈 Used 3 different Hilite types this week")
	}

	if len(distinctPeople(entries, weekFrom, weekThrough)) >= peopleForCallout {
		callouts = append(callouts, "🤝 Worked with 7 people this week")
	}
	if len(distinctTags(entries, recentFrom, recentThrough, true)) >= ticketsForCallout {
		callouts = append(callouts, "🎫 Covered 5 tickets in the last 5 workdays")
	}
	if len(distinctTags(entries, weekFrom, weekThrough, false)) >= projectTagsForCallout {
		callouts = append(callouts, "🗂️ Touched 4 projects or topics this week")
	}
	if tag, ok := mostUsedTag(entries, weekFrom, weekThrough, repeatedTagForCallout); ok {
		callouts = append(callouts, fmt.Sprintf("🔁 Used %s in 5 entries this week", tag))
	}
	if tag, ok := hasConsecutiveTagDays(entries, weekFrom, weekThrough, continuityForCallout); ok {
		callouts = append(callouts, fmt.Sprintf("🎯 Kept %s moving for 3 workdays", tag))
	}
	return callouts
}

func selectStreakCallouts(candidates []string) []string {
	if len(candidates) <= maxStreakCallouts {
		return candidates
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	random.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	return candidates[:maxStreakCallouts]
}

func currentStreakCallouts() (all, selected []string) {
	all = streakCalloutCandidates(AllLedgerEntries(), time.Now(), CurrentStreak())
	selected = selectStreakCallouts(append([]string{}, all...))
	return all, selected
}

func showStreakAchievementsWindow(a fyne.App, callouts []string) {
	w := a.NewWindow("Dunnit: Achievements")
	list := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("%d streak indicators hit today:", len(callouts))),
	)
	for _, callout := range callouts {
		list.Add(widget.NewLabel("• " + callout))
	}
	w.SetContent(windowPad(container.NewVScroll(list)))
	w.Resize(fyne.NewSize(440, 420))
	w.Show()
}

// streakSummary returns a compact positive summary plus a button for the
// complete achievement list. It stays empty when nothing qualifies.
func streakSummary(a fyne.App) fyne.CanvasObject {
	all, selected := currentStreakCallouts()
	if len(all) == 0 {
		return widget.NewLabel("")
	}
	lead := fmt.Sprintf("🎉 Wow, you hit %d streak indicators today!", len(all))
	if len(selected) == 1 {
		lead += " One of those is:"
	} else {
		lead += " A couple of those are:"
	}
	return container.NewVBox(
		widget.NewLabel(lead+"\n"+strings.Join(selected, "\n")),
		widget.NewButton(fmt.Sprintf("See all %d achievements…", len(all)), func() {
			showStreakAchievementsWindow(a, all)
		}),
	)
}
