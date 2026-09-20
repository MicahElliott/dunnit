package dun

import (
	"regexp"
	"strings"
	"time"
)

// carryForwardSincePrefix marks a copied-forward open item with its
// original log date, e.g. " s/2026-09-11". The full date keeps the stored
// marker unambiguous; the row renderer turns it into a compact age/date badge.
// See docs/todo-carryforward-design.md.
const carryForwardSincePrefix = " s/"

var carryForwardSincePattern = regexp.MustCompile(` s/(\d{4}-\d{2}-\d{2})`)
var legacyCarryForwardSincePattern = regexp.MustCompile(` \(since (\d{4}-\d{2}-\d{2})\)`)

func carryForwardSinceSuffix(date time.Time) string {
	return carryForwardSincePrefix + date.Format("2006-01-02")
}

// parseCarryForwardSince extracts the full date embedded by
// carryForwardSinceSuffix from text, if present.
func parseCarryForwardSince(text string) (since time.Time, ok bool) {
	for _, pattern := range []*regexp.Regexp{
		carryForwardSincePattern,
		legacyCarryForwardSincePattern,
	} {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			t, err := time.ParseInLocation("2006-01-02", match[1], time.Local)
			if err != nil || (!since.IsZero() && !t.Before(since)) {
				continue
			}
			since = t
		}
	}
	return since, !since.IsZero()
}

// stripCarryForwardSince removes current and legacy carry-forward suffixes
// from text. Repeating the removal also cleans up rows that accumulated
// more than one marker in older versions of the app.
func stripCarryForwardSince(text string) string {
	for {
		removed := false
		for _, pattern := range []*regexp.Regexp{
			carryForwardSincePattern,
			legacyCarryForwardSincePattern,
		} {
			matches := pattern.FindAllStringIndex(text, -1)
			if len(matches) > 0 {
				match := matches[len(matches)-1]
				if match[1] != len(text) {
					continue
				}
				text = strings.TrimRight(text[:match[0]], " \t")
				removed = true
				break
			}
		}
		if !removed {
			return text
		}
	}
}

// staleDateFor returns the date an open item's staleness/carry-
// forward "since" should be measured from: its own embedded
// carryForwardSinceSuffix if it already has one (i.e. it was already
// copied forward at least once before), otherwise entryDate (the
// date it was actually first logged).
func staleDateFor(text string, entryDate time.Time) time.Time {
	if since, ok := parseCarryForwardSince(text); ok {
		return since
	}
	return entryDate
}

// priorOpenItems scans every ledger entry dated strictly before
// today (via AllLedgerEntries, oldest-first, spanning every ledger
// file -- not just yesterday's) for openTrackedCategories entries
// that were never resolved (converted to DONE/SOMEDAY/DISCARDED at
// any later date, in any file) -- same resolution logic as
// todos.go's parseOpenItems, generalized to span history instead of
// a single day's lines. Each returned OpenItem's Text already has any
// prior carryForwardSinceSuffix stripped, paired with its original
// "since" date (staleDateFor) for the caller to re-annotate.
func priorOpenItems() (items []OpenItem, sinceDates []time.Time) {
	entries := AllLedgerEntries()
	today := time.Now()
	ty, tm, td := today.Date()

	type candidate struct {
		item  OpenItem
		since time.Time
	}
	active := make(map[string]candidate)
	var order []string

	for _, e := range entries {
		ey, em, ed := e.Date.Date()
		isToday := ey == ty && em == tm && ed == td
		isResolving := false
		for _, rc := range resolvingCategories {
			if e.Category == rc {
				isResolving = true
				break
			}
		}
		if isResolving {
			for _, srcCat := range openTrackedCategories {
				if suffix := convertedSuffix(srcCat); strings.HasSuffix(e.Text, suffix) {
					orig := strings.TrimSuffix(e.Text, suffix)
					for key, item := range active {
						if isLifecycleCategory(item.item.Category) && isLifecycleCategory(srcCat) {
							if resolutionMatches(
								stripCarryForwardSince(item.item.Text),
								stripCarryForwardSince(orig)) {
								delete(active, key)
							}
							continue
						}
						if item.item.Category == srcCat && resolutionMatches(
							stripCarryForwardSince(item.item.Text),
							stripCarryForwardSince(orig)) {
							delete(active, key)
						}
					}
				}
			}
			continue
		}
		if isToday {
			// Don't treat today's own entries as "prior" open
			// items -- carry-forward only ever looks at what was
			// left open as of *before* today.
			continue
		}
		if isOpenTrackedCategory(e.Category) {
			key := openItemKey(e.Category, e.Text)
			if _, exists := active[key]; !exists {
				order = append(order, key)
			}
			active[key] = candidate{
				item:  OpenItem{Category: e.Category, Text: stripCarryForwardSince(e.Text)},
				since: staleDateFor(e.Text, e.Date),
			}
		}
	}

	for _, key := range order {
		item, ok := active[key]
		if !ok {
			continue
		}
		items = append(items, item.item)
		sinceDates = append(sinceDates, item.since)
	}
	return items, sinceDates
}

type openHistoryItem struct {
	item  OpenItem
	since time.Time
	date  time.Time
}

// stalePlanItem keeps the original date alongside the active item so SOD can
// explain why it needs review. OpenItem is embedded so existing item actions
// can continue to use the same value.
type stalePlanItem struct {
	OpenItem
	Since time.Time
}

// dailyCarryCategories are the items that belong in today's plan. Other open
// categories remain useful Start of Day context, but copying them into
// Daybook makes the plan noisy and makes RISK look like an action.
var dailyCarryCategories = map[string]bool{
	"TODO":  true,
	"DOING": true,
}

const dailyCarryLookbackDays = 7

// openItemsThrough returns the open-item state at the end of through. It
// applies the same append-only resolution rules as parseOpenItems, while
// retaining each item's original date for stale-age display.
func openItemsThrough(entries []LedgerEntry, through time.Time) (map[string]openHistoryItem, []string) {
	active := make(map[string]openHistoryItem)
	var order []string
	ordered := make(map[string]bool)
	var resolutions []resolvedOpenItem

	for _, e := range entries {
		if e.Date.After(through) {
			break
		}
		isResolving := false
		for _, rc := range resolvingCategories {
			if e.Category == rc {
				isResolving = true
				break
			}
		}
		if isResolving {
			for _, srcCat := range openTrackedCategories {
				if suffix := convertedSuffix(srcCat); strings.HasSuffix(e.Text, suffix) {
					orig := strings.TrimSuffix(e.Text, suffix)
					resolutions = append(resolutions, resolvedOpenItem{category: srcCat, text: orig})
					for key, state := range active {
						if isLifecycleCategory(state.item.Category) && isLifecycleCategory(srcCat) {
							if resolutionMatches(stripCarryForwardSince(state.item.Text), stripCarryForwardSince(orig)) {
								delete(active, key)
							}
							continue
						}
						if state.item.Category == srcCat && resolutionMatches(
							stripCarryForwardSince(state.item.Text), stripCarryForwardSince(orig)) {
							delete(active, key)
						}
					}
				}
			}
			continue
		}
		if !isOpenTrackedCategory(e.Category) {
			continue
		}
		if isResolvedCarryForward(e.Category, e.Text, resolutions) {
			continue
		}
		key := openItemKey(e.Category, e.Text)
		if !ordered[key] {
			order = append(order, key)
			ordered[key] = true
		}
		active[key] = openHistoryItem{
			item: OpenItem{
				Category:  e.Category,
				Text:      stripCarryForwardSince(e.Text),
				Time:      e.Time,
				LineIndex: e.Line,
				Source:    e.Source,
			},
			since: staleDateFor(e.Text, e.Date),
			date:  e.Date,
		}
	}
	return active, order
}

// lastActiveLedgerDate returns the newest prior date with at least one
// ledger entry. It is deliberately independent of the carry-forward source:
// a quiet day may still be the most recent day whose risks or reflections
// should be shown as Start of Day context.
func lastActiveLedgerDate(entries []LedgerEntry, now time.Time) (time.Time, bool) {
	var latest time.Time
	for _, e := range entries {
		if e.Date.Format("2006-01-02") == now.Format("2006-01-02") {
			continue
		}
		if e.Date.Before(now) && e.Date.After(latest) {
			latest = e.Date
		}
	}
	return latest, !latest.IsZero()
}

// openItemsAtDate returns items that were open at the end of date and remain
// unresolved now. This keeps SOD reminders tied to the last active ledger
// without resurfacing something already completed today.
func openItemsAtDate(entries []LedgerEntry, date, now time.Time) []OpenItem {
	atDate, order := openItemsThrough(entries, date)
	current, _ := openItemsThrough(entries, now)
	var items []OpenItem
	for _, key := range order {
		state, ok := atDate[key]
		if !ok {
			continue
		}
		if _, stillOpen := current[key]; stillOpen {
			items = append(items, state.item)
		}
	}
	return items
}

// dailyCarryForwardItems finds the newest prior calendar day in the seven-day
// lookback whose still-open plan contains TODO/DOING items. Items resolved by
// today are excluded, so a completed task cannot be resurrected by kickoff.
func dailyCarryForwardItems(now time.Time) (sourceDate time.Time, items []OpenItem, sinceDates []time.Time) {
	entries := AllLedgerEntries()
	current, _ := openItemsThrough(entries, now)
	for offset := 1; offset <= dailyCarryLookbackDays; offset++ {
		date := now.AddDate(0, 0, -offset)
		atDate, order := openItemsThrough(entries, date)
		var dayItems []OpenItem
		var daySince []time.Time
		seen := make(map[string]bool)
		for _, key := range order {
			state, ok := atDate[key]
			if !ok || !dailyCarryCategories[state.item.Category] || !sameCalendarDate(state.date, date) || seen[key] {
				continue
			}
			if _, stillOpen := current[key]; !stillOpen {
				continue
			}
			seen[key] = true
			dayItems = append(dayItems, state.item)
			daySince = append(daySince, state.since)
		}
		if len(dayItems) > 0 {
			return date, dayItems, daySince
		}
	}
	return time.Time{}, nil, nil
}

// carryForwardDailyPlan copies the selected source day's TODO/DOING items
// into today's ledger. The ledger itself is the idempotency guard: this can
// safely be called again after a window is reopened or another machine's
// already-synced copies are present.
func carryForwardDailyPlan(now time.Time) (sourceDate time.Time, items []OpenItem) {
	sourceDate, candidates, sinceDates := dailyCarryForwardItems(now)
	if sourceDate.IsZero() {
		return time.Time{}, nil
	}

	todayKeys := make(map[string]bool)
	for _, line := range readLedgerLines() {
		cat, text, ok := parseLedgerLine(line)
		if ok && dailyCarryCategories[cat] {
			todayKeys[openItemKey(cat, text)] = true
		}
	}
	for i, item := range candidates {
		key := openItemKey(item.Category, item.Text)
		if todayKeys[key] {
			continue
		}
		recordActivity(item.Text+carryForwardSinceSuffix(sinceDates[i]), item.Category)
		todayKeys[key] = true
	}
	return sourceDate, candidates
}

// staleReviewLookbackDays bounds SOD's daily-purpose review. This gives a
// missed kickoff enough recovery room without turning the daily surface into
// an archive of every unresolved item ever logged.
const staleReviewLookbackDays = 30

// staleDailyPlanItems returns unresolved TODO/DOING items old enough to need
// an explicit SOMEDAY decision, limited to the recent daily-planning horizon.
func staleDailyPlanItems(now time.Time) []stalePlanItem {
	entries := AllLedgerEntries()
	active, order := openItemsThrough(entries, now)
	var stale []stalePlanItem
	seen := make(map[string]bool)
	for _, key := range order {
		state, ok := active[key]
		age := daysSinceDate(state.since, now)
		if !ok || !dailyCarryCategories[state.item.Category] || age < staleReviewDays || age > staleReviewLookbackDays {
			continue
		}
		logicalKey := openItemKey(state.item.Category, state.item.Text)
		if seen[logicalKey] {
			continue
		}
		seen[logicalKey] = true
		stale = append(stale, stalePlanItem{OpenItem: state.item, Since: state.since})
	}
	return stale
}

func daysSinceDate(since, now time.Time) int {
	y1, m1, d1 := since.Date()
	y2, m2, d2 := now.Date()
	t1 := time.Date(y1, m1, d1, 0, 0, 0, 0, now.Location())
	t2 := time.Date(y2, m2, d2, 0, 0, 0, 0, now.Location())
	return int(t2.Sub(t1).Hours() / 24)
}

func sameCalendarDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

// daysSince returns the whole number of calendar days between since
// and now (0 if since is today).
func daysSince(since time.Time) int {
	y1, m1, d1 := since.Date()
	y2, m2, d2 := time.Now().Date()
	t1 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.Local)
	t2 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.Local)
	return int(t2.Sub(t1).Hours() / 24)
}

// staleReviewDays is the age at which Start of Day asks whether an open
// TODO/DOING item should move to SOMEDAY. The move remains explicit; age
// alone never changes the ledger.
const staleReviewDays = 7

// openItemDisplayText keeps the stored creation marker available to
// itemTextLabel, which renders it as a compact age/date badge. All open-item
// views use this helper so TODO, DOING, and the other tracked categories
// present the same lifecycle cue.
func openItemDisplayText(text string) string {
	// Normalize legacy or repeated markers to one current-format marker so
	// every open-item view presents the same creation and age metadata.
	since, _ := parseCarryForwardSince(text)
	text = stripCarryForwardSince(text)
	if !since.IsZero() {
		text += carryForwardSinceSuffix(since)
	}
	return text
}
