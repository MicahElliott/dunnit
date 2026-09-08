package dun

import (
	"strconv"
	"strings"
	"time"
)

// carryForwardSincePrefix marks a copied-forward open item with its
// original log date, e.g. " (since 2026-08-28)" -- parallel to
// todos.go's convertedSuffix convention, not a new tag/marker syntax.
// See docs/todo-carryforward-design.md.
const carryForwardSincePrefix = " (since "

func carryForwardSinceSuffix(date time.Time) string {
	return carryForwardSincePrefix + date.Format("2006-01-02") + ")"
}

// parseCarryForwardSince extracts the "since" date embedded by
// carryForwardSinceSuffix from text, if present. ok is false if text
// has no such suffix.
func parseCarryForwardSince(text string) (since time.Time, ok bool) {
	idx := strings.LastIndex(text, carryForwardSincePrefix)
	if idx == -1 || !strings.HasSuffix(text, ")") {
		return time.Time{}, false
	}
	datePart := text[idx+len(carryForwardSincePrefix) : len(text)-1]
	t, err := time.ParseInLocation("2006-01-02", datePart, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// stripCarryForwardSince removes a carryForwardSinceSuffix from text
// if present, otherwise returns text unchanged -- used so
// re-copying an already-once-carried-forward item doesn't stack a
// second "(since ...)" suffix on top of the first.
func stripCarryForwardSince(text string) string {
	idx := strings.LastIndex(text, carryForwardSincePrefix)
	if idx == -1 || !strings.HasSuffix(text, ")") {
		return text
	}
	return text[:idx]
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

	type key struct {
		cat, text string
	}
	resolved := make(map[key]bool)
	var openEntries []LedgerEntry

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
					orig = stripCarryForwardSince(orig)
					resolved[key{srcCat, orig}] = true
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
			openEntries = append(openEntries, e)
		}
	}

	for _, e := range openEntries {
		strippedText := stripCarryForwardSince(e.Text)
		if resolved[key{e.Category, strippedText}] {
			continue
		}
		items = append(items, OpenItem{Category: e.Category, Text: strippedText})
		sinceDates = append(sinceDates, staleDateFor(e.Text, e.Date))
	}
	return items, sinceDates
}

// runCarryForwardIfNeeded copies still-open items (openTrackedCategories)
// left unresolved from any prior ledger day into today's ledger,
// annotated with their original log date via carryForwardSinceSuffix
// -- see docs/todo-carryforward-design.md for the full design
// rationale.
//
// Idempotent per calendar day, guarded by Config.LastCarryForwardDate
// (persisted, not inferred from ledger content) -- safe to call from
// multiple touchpoints (Daybook open, SOD open, first recordActivity)
// without worrying about ordering or double-running: whichever fires
// first does the work and updates the marker, the rest become no-ops.
// This is deliberately NOT gated behind any wizard/dialog completing,
// so a user who starts logging entries before ever opening Kickoff
// still gets carry-forward.
func runCarryForwardIfNeeded() {
	cfg := LoadConfig()
	today := time.Now().Format("2006-01-02")
	if cfg.LastCarryForwardDate == today {
		return
	}

	// Mark done *before* copying items -- recordActivity below is the
	// same call path this function itself may be hooked into (see
	// call sites), so updating the marker first prevents the copy
	// loop's own recordActivity calls from re-triggering another
	// (redundant, infinite-recursion-shaped) carry-forward pass.
	cfg.LastCarryForwardDate = today
	if err := writeConfig(cfg); err != nil {
		// Non-fatal: worst case, carry-forward re-runs (harmlessly
		// duplicating already-copied lines) next touchpoint if the
		// marker failed to persist. Not worth surfacing to the user.
		_ = err
	}

	items, sinceDates := priorOpenItems()
	for i, item := range items {
		recordActivity(item.Text+carryForwardSinceSuffix(sinceDates[i]), item.Category)
	}
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

// staleDaysThreshold is how many days old a carried-forward item's
// "since" date must be before Planned flags it as stale (purely
// display -- see docs/todo-carryforward-design.md).
const staleDaysThreshold = 4

// staleBadge returns a short display suffix (e.g. " \u26a0 4d") for
// an item whose carryForwardSinceSuffix-embedded date is more than
// staleDaysThreshold days old, or "" if item.Text has no such suffix
// or isn't old enough yet to flag. Purely a display computation --
// nothing is written back to the ledger.
func staleBadge(text string) string {
	since, ok := parseCarryForwardSince(text)
	if !ok {
		return ""
	}
	days := daysSince(since)
	if days < staleDaysThreshold {
		return ""
	}
	return " \u26a0 " + strconv.Itoa(days) + "d"
}
