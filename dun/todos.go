package dun

import (
	"sort"
	"strings"
)

// OpenItem is a not-yet-resolved item pulled from today's ledger
// (FR-07, extended for the WAITING/QUESTION/FIXME/RISK follow-up):
// shown together under Daybook's "Upcoming" section and in SOD/SOM.
type OpenItem struct {
	Category string // one of openTrackedCategories
	Text     string
	// LineIndex is the 0-based index of this item's originating line
	// within readLedgerLines(), populated by getCategoryGroupItems and
	// (as of the Planned/Reflections Edit-button addition)
	// parseOpenItems/getOpenItems too -- lets replaceLedgerLineTextAt
	// target the exact line even if other lines share the same
	// category+text.
	LineIndex int
}

// openTrackedCategories are the categories tracked as "open items"
// needing eventual follow-up/resolution -- shown in Daybook's
// Upcoming section and SOD/SOM, and resolvable via the Done/Postpone
// actions. TODO/GOAL were the original FR-07 set; WAITING/QUESTION/
// FIXME/RISK share the same "logged now, tracked as open, resolved
// later" pattern (unlike day-to-day capture categories like DONE/
// TIL), so they're tracked the same way.
var openTrackedCategories = []string{"TODO", "GOAL", "WAITING", "QUESTION", "FIXME", "RISK"}

func isOpenTrackedCategory(cat string) bool {
	for _, c := range openTrackedCategories {
		if cat == c {
			return true
		}
	}
	return false
}

// resolvingCategories are the categories an open item can be
// "resolved" into from the Upcoming list, and what button triggers
// each: DONE via the "Done" button (actually completed), SOMEDAY via
// the "Postpone" button (deliberately deferred rather than pretending
// it's done -- see FR-07 follow-up), DISCARDED via the "Nah" button
// (deliberately dropped, not done and not deferred -- just no longer
// relevant). All three leave the original line untouched (append-only
// ledger design) and are recognized by parseOpenItems as removing the
// item from the open/Upcoming list. DISCARDED isn't a selectable
// Daybook category (see categories.go) -- it only ever gets written
// via recordDiscarded, never picked by hand.
var resolvingCategories = []string{"DONE", "SOMEDAY", "DISCARDED"}

// convertedSuffix marks a resolving line (DONE or SOMEDAY) as having
// been generated from an open item, so parseOpenItems can recognize
// it and exclude the original from the "open" list. Kept as an exact,
// greppable suffix rather than a separate marker file, to stay
// append-only/plain-text (per project's ledger design).
func convertedSuffix(category string) string {
	return " (via " + category + ")"
}

// parseLedgerLine splits a ledger line "[HH:MM:SS] CATEGORY text"
// into category and text. Returns ok=false if the line doesn't look
// like a well-formed ledger entry.
func parseLedgerLine(line string) (category, text string, ok bool) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// parseOpenItems scans ledger lines for open-tracked-category entries
// that have not yet been resolved (converted to DONE or postponed to
// SOMEDAY, via recordConvertedDone/recordPostponed), in first-seen
// order.
func parseOpenItems(lines []string) []OpenItem {
	var open []OpenItem
	resolved := make(map[string]bool) // "CATEGORY\x00text" -> true

	for i, line := range lines {
		cat, text, ok := parseLedgerLine(line)
		if !ok {
			continue
		}
		isResolving := false
		for _, rc := range resolvingCategories {
			if cat == rc {
				isResolving = true
				break
			}
		}
		if isResolving {
			for _, srcCat := range openTrackedCategories {
				if suffix := convertedSuffix(srcCat); strings.HasSuffix(text, suffix) {
					orig := strings.TrimSuffix(text, suffix)
					resolved[srcCat+"\x00"+orig] = true
				}
			}
			continue
		}
		if isOpenTrackedCategory(cat) {
			open = append(open, OpenItem{Category: cat, Text: text, LineIndex: i})
		}
	}

	var result []OpenItem
	for _, item := range open {
		if !resolved[item.Category+"\x00"+item.Text] {
			result = append(result, item)
		}
	}
	return result
}

// getOpenItems returns today's open (unresolved) tracked items.
func getOpenItems() []OpenItem {
	return parseOpenItems(readLedgerLines())
}

// recordConvertedDone logs a DONE entry referencing an original open
// item's text, marking it as resolved (FR-07). The original line is
// left untouched (append-only ledger design). item.Text's leading
// word is flipped to past tense first (PastTenseLeadingWord,
// pastverb.go) -- Plan items are typically phrased as imperatives
// ("Fix the login bug"), which reads oddly once marked DONE, so this
// converts it to "Fixed the login bug" instead. Purely cosmetic;
// harmless no-op if the leading word isn't a recognized/regular verb
// (returned unchanged by PastTense in that case).
func recordConvertedDone(item OpenItem) {
	recordActivity(PastTenseLeadingWord(item.Text)+convertedSuffix(item.Category), "DONE")
}

// recordPostponed logs a SOMEDAY entry referencing an original open
// item's text, marking it as resolved without pretending it was
// completed -- for deliberately deferring an item so the Upcoming
// list doesn't grow unbounded. The original line is left untouched.
func recordPostponed(item OpenItem) {
	recordActivity(item.Text+convertedSuffix(item.Category), "SOMEDAY")
}

// recordDiscarded logs a DISCARDED entry referencing an original open
// item's text, marking it as resolved via outright dismissal (the
// "Nah" button) -- distinct from Postpone (SOMEDAY, meant to revisit
// later) since a discarded item isn't expected to come back. The
// original line is left untouched.
func recordDiscarded(item OpenItem) {
	recordActivity(item.Text+convertedSuffix(item.Category), "DISCARDED")
}

// groupOpenItemsByCategory buckets items by category, preserving
// openTrackedCategories order, and skips empty buckets. Shared by
// Daybook's Upcoming section, SOD, and SOM so all three list open
// items (TODO/GOAL/WAITING/QUESTION/FIXME/RISK) the same way, rather
// than each hardcoding its own TODO-vs-GOAL binary split. Within each
// category bucket, items are ordered by leading tag (see
// leadingTagSortKey) rather than left in ledger/first-seen order --
// grouping same-tagged items together makes a category's items
// easier to scan when several distinct projects/tags are mixed
// together within it.
func groupOpenItemsByCategory(items []OpenItem) (categories []string, grouped map[string][]OpenItem) {
	grouped = make(map[string][]OpenItem)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	for _, cat := range openTrackedCategories {
		if len(grouped[cat]) > 0 {
			categories = append(categories, cat)
			sortItemsByLeadingTag(grouped[cat])
		}
	}
	return categories, grouped
}

// leadingTagSortKey returns the first #tag found in text (per
// extractTags), or "" if text has no tag -- used to sort items within
// a category bucket so same-tagged items cluster together.
func leadingTagSortKey(text string) string {
	tags := extractTags(text)
	if len(tags) == 0 {
		return ""
	}
	return tags[0]
}

// sortItemsByLeadingTag sorts items in place by leadingTagSortKey,
// tagged items first (alphabetically by tag), untagged items
// (empty key) last -- the opposite of plain string comparison's
// default "" first ordering, since untagged items are meant to read
// as lower-priority/less-organized than anything with a tag. Stable,
// so items sharing a tag (or both untagged) keep their original
// relative (ledger/first-seen) order.
func sortItemsByLeadingTag(items []OpenItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := leadingTagSortKey(items[i].Text), leadingTagSortKey(items[j].Text)
		if a == "" && b == "" {
			return false
		}
		if a == "" {
			return false // untagged never sorts before a tagged item
		}
		if b == "" {
			return true // tagged always sorts before an untagged item
		}
		return a < b
	})
}

// isExcludedTagItem reports whether item's leading tag
// (leadingTagSortKey) matches one of excludeTags (case-insensitive,
// same normalization as summarize.go's lineHasExcludedTag) -- used to
// push items tagged with a user-configured "noise" tag
// (Config.ReportExcludeTags) below even the untagged items in
// Planned/Endings/Hilites, hidden by default behind a "Show all"
// toggle (mirroring Planned's existing non-TODO-category toggle).
// Untagged items are never excluded regardless of excludeTags.
func isExcludedTagItem(text string, excludeTags []string) bool {
	tag := leadingTagSortKey(text)
	if tag == "" || len(excludeTags) == 0 {
		return false
	}
	for _, ex := range excludeTags {
		if strings.EqualFold(tag, ex) {
			return true
		}
	}
	return false
}

// splitExcludedTagItems partitions items (already sorted via
// sortItemsByLeadingTag) into visible (shown by default) and excluded
// (leading tag matches excludeTags, hidden behind a "Show all"
// toggle), preserving each side's relative order.
func splitExcludedTagItems(items []OpenItem, excludeTags []string) (visible, excluded []OpenItem) {
	for _, item := range items {
		if isExcludedTagItem(item.Text, excludeTags) {
			excluded = append(excluded, item)
		} else {
			visible = append(visible, item)
		}
	}
	return visible, excluded
}

// categoryPlural returns a simple plural label for a category code,
// used as a sub-heading (e.g. "TODOs", "GOALs", "WAITINGs" -- good
// enough for these short all-caps codes, no need for real pluralization
// rules).
func categoryPlural(cat string) string {
	return cat + "s"
}

// lastDoneItem returns the most recently logged DONE entry (text with
// any "(via CATEGORY)" suffix stripped, plus its LineIndex into
// readLedgerLines() for in-place rewriting), or ok=false if there is
// no DONE entry logged yet today. Used by Ditto (ui.go) so it only
// ever repeats/extends the last *DONE* item, not whatever category
// happens to be most recent overall (e.g. a RISK logged afterward).
func lastDoneItem() (item OpenItem, ok bool) {
	lines := readLedgerLines()
	for i := len(lines) - 1; i >= 0; i-- {
		cat, text, lineOk := parseLedgerLine(lines[i])
		if !lineOk || cat != "DONE" {
			continue
		}
		for _, srcCat := range openTrackedCategories {
			text = strings.TrimSuffix(text, convertedSuffix(srcCat))
		}
		return OpenItem{Category: cat, Text: text, LineIndex: i}, true
	}
	return OpenItem{}, false
}

// getCompletedItems returns today's DONE entries, in the order they
// were logged, for Daybook's collapsible "Completed" section. The
// "(via CATEGORY)" suffix (added when an open item is converted via
// the Upcoming list's Done button, see convertedSuffix) is stripped
// for a cleaner display -- the ledger itself keeps the full text as
// written, this only affects what's shown here.
func getCompletedItems() []string {
	var out []string
	for _, line := range readLedgerLines() {
		cat, text, ok := parseLedgerLine(line)
		if !ok || cat != "DONE" {
			continue
		}
		for _, srcCat := range openTrackedCategories {
			text = strings.TrimSuffix(text, convertedSuffix(srcCat))
		}
		out = append(out, text)
	}
	return out
}

// categoryGroupOrder returns the category codes belonging to group
// ("end"/"plan"/"hilite"), in Categories' declared order -- the
// canonical per-group ordering used to keep Daybook's Completed/
// Planned/Reflections sub-headings consistent with categories.go
// rather than each section re-deriving its own order.
func categoryGroupOrder(group string) []string {
	var codes []string
	for _, c := range Categories {
		if c.Group == group {
			codes = append(codes, c.Code)
		}
	}
	return codes
}

// getCategoryGroupItems returns today's entries whose category
// belongs to group ("end"/"hilite"), in first-seen order. Used by
// Daybook's Completed ("end") and Reflections ("hilite") sections --
// the general-purpose sibling of getOpenItems, which is specific to
// openTrackedCategories ("plan"). Strips the "(via CATEGORY)" suffix
// (see convertedSuffix) from DONE entries converted from an open
// item, same as getCompletedItems did -- harmless no-op for any other
// category, which never carries that suffix.
func getCategoryGroupItems(group string) []OpenItem {
	codes := make(map[string]bool)
	for _, c := range categoryGroupOrder(group) {
		codes[c] = true
	}
	var out []OpenItem
	for i, line := range readLedgerLines() {
		cat, text, ok := parseLedgerLine(line)
		if !ok || !codes[cat] {
			continue
		}
		for _, srcCat := range openTrackedCategories {
			text = strings.TrimSuffix(text, convertedSuffix(srcCat))
		}
		out = append(out, OpenItem{Category: cat, Text: text, LineIndex: i})
	}
	return out
}

// groupCategoryItemsByGroup buckets items (as returned by
// getCategoryGroupItems) by category, preserving group's
// categoryGroupOrder and skipping empty buckets -- the general-
// purpose sibling of groupOpenItemsByCategory, letting Endings and
// Hilites show per-category sub-headings the same way Planned
// already does. Also sorts each bucket by leading tag, same as
// groupOpenItemsByCategory (see sortItemsByLeadingTag).
func groupCategoryItemsByGroup(group string, items []OpenItem) (categories []string, grouped map[string][]OpenItem) {
	grouped = make(map[string][]OpenItem)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	for _, cat := range categoryGroupOrder(group) {
		if len(grouped[cat]) > 0 {
			categories = append(categories, cat)
			sortItemsByLeadingTag(grouped[cat])
		}
	}
	return categories, grouped
}
