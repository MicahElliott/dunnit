package dun

import (
	"sort"
	"strings"
	"time"
)

// OpenItem is a not-yet-resolved item pulled from today's ledger
// (FR-07, extended for the WAITING/QUESTION/FIXME/RISK follow-up):
// shown together under Daybook's "Upcoming" section and in SOD/SOM.
type OpenItem struct {
	Category string // one of openTrackedCategories
	Text     string
	Time     time.Time // timestamp parsed from the originating ledger line
	// LineIndex is the 0-based index of this item's originating line
	// within readLedgerLines(), populated by getCategoryGroupItems and
	// (as of the Planned/Reflections Edit-button addition)
	// parseOpenItems/getOpenItems too -- lets replaceLedgerLineTextAt
	// target the exact line even if other lines share the same
	// category+text.
	LineIndex int
	// Source is the ledger file containing LineIndex. It is populated for
	// historical SOD context rows; an empty value means today's ledger.
	Source string
}

// openTrackedCategories are the categories tracked as "open items"
// needing eventual follow-up/resolution -- shown in Daybook's
// Upcoming section and SOD/SOM, and resolvable via the Done/Postpone
// actions. TODO/DOING/GOAL were the original FR-07 set plus the active
// lifecycle state; WAITING/QUESTION/
// FIXME/RISK share the same "logged now, tracked as open, resolved
// later" pattern (unlike day-to-day capture categories like DONE/
// TIL), so they're tracked the same way.
var openTrackedCategories = []string{"DOING", "TODO", "GOAL", "WAITING", "QUESTION", "FIXME", "RISK"}

// legacyOngoingCategory is retained only for history-aware code and tests.
// It is intentionally absent from Categories and openTrackedCategories.
const legacyOngoingCategory = "ONGOING"

func isLifecycleCategory(cat string) bool {
	return cat == "TODO" || cat == "DOING"
}

func isLifecycleEndpoint(cat string) bool {
	return cat == "DONE" || cat == "HANDLED" || cat == "FAIL" || cat == "WASTED"
}

// openItemKey identifies the logical item represented by an open ledger
// entry. TODO and DOING intentionally share a key so state transitions and
// daily carry-forward copies collapse to one current item.
func openItemKey(category, text string) string {
	if isLifecycleCategory(category) {
		category = "TODO/DOING"
		text = normalizeLifecycleEntryText(text)
	}
	return category + "\x00" + stripCarryForwardSince(text)
}

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
// item from the open/Upcoming list. DISCARDED is a historical/dedicated-flow
// category (see categories.go) -- it only ever gets written via
// recordDiscarded, never picked by hand.
var resolvingCategories = []string{"DONE", "HANDLED", "FAIL", "WASTED", "SOMEDAY", "DISCARDED"}

// convertedSuffix marks a resolving line (DONE, HANDLED, or SOMEDAY) as having
// been generated from an open item, so parseOpenItems can recognize
// it and exclude the original from the "open" list. Kept as an exact,
// greppable suffix rather than a separate marker file, to stay
// append-only/plain-text (per project's ledger design).
func convertedSuffix(category string) string {
	return " (via " + category + ")"
}

// resolutionMatches reports whether resolvedText identifies sourceText.
// DONE entries may inflect the source item's leading verb for display
// (for example, "write report" becomes "wrote report"), while SOMEDAY
// and DISCARDED entries retain the original text.
func resolutionMatches(sourceText, resolvedText string) bool {
	sourceText = stripFlags(sourceText)
	resolvedText = stripFlags(resolvedText)
	return resolvedText == sourceText ||
		resolvedText == PastTenseLeadingWord(sourceText) ||
		resolvedText == inflectLifecycleText(sourceText, "DONE")
}

type resolvedOpenItem struct {
	category string
	text     string
}

// resolvedOpenItemMatches reports whether a later open entry is the same
// logical item as a recorded resolution. A carried-forward copy keeps the
// original text and its s/YYYY-MM-DD marker, so it can be ignored after the
// resolution without suppressing a newly entered, unmarked TODO.
func resolvedOpenItemMatches(category, text string, resolved resolvedOpenItem) bool {
	text = stripCarryForwardSince(text)
	resolved.text = stripCarryForwardSince(resolved.text)
	if isLifecycleCategory(category) && isLifecycleCategory(resolved.category) {
		return resolutionMatches(text, resolved.text)
	}
	return category == resolved.category && resolutionMatches(text, resolved.text)
}

func isResolvedCarryForward(category, text string, resolutions []resolvedOpenItem) bool {
	if !hasCarryForwardSince(text) {
		return false
	}
	for _, resolved := range resolutions {
		if resolvedOpenItemMatches(category, text, resolved) {
			return true
		}
	}
	return false
}

func hasCarryForwardSince(text string) bool {
	_, ok := parseCarryForwardSince(text)
	return ok
}

// parseLedgerLine splits a ledger line "[HH:MM] CATEGORY text" (or a
// legacy seconds-bearing line) into category and text.
// into category and text. Returns ok=false if the line doesn't look
// like a well-formed ledger entry.
func parseLedgerLine(line string) (category, text string, ok bool) {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return "", "", false
	}
	return parts[1], parts[2], true
}

// parseOpenItems scans ledger lines for open-tracked-category entries that
// have not yet been resolved. TODO and DOING entries with the same task text
// are one logical item, with the newest active state winning.
func parseOpenItems(lines []string) []OpenItem {
	active := make(map[string]OpenItem)
	var order []string
	ordered := make(map[string]bool)
	var resolutions []resolvedOpenItem
	today := time.Now()

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
					resolutions = append(resolutions, resolvedOpenItem{category: srcCat, text: orig})
					resolveOpenItems(active, srcCat, orig)
				}
			}
			continue
		}
		if isOpenTrackedCategory(cat) {
			if isResolvedCarryForward(cat, text, resolutions) {
				continue
			}
			key := openItemKey(cat, text)
			if !ordered[key] {
				order = append(order, key)
				ordered[key] = true
			}
			stamp, _ := parseLedgerLineTime(line, today)
			active[key] = OpenItem{Category: cat, Text: text, Time: stamp, LineIndex: i}
		}
	}

	result := make([]OpenItem, 0, len(active))
	for _, key := range order {
		if item, ok := active[key]; ok {
			result = append(result, item)
		}
	}
	return result
}

func resolveOpenItems(active map[string]OpenItem, sourceCategory, resolvedText string) {
	for key, item := range active {
		if isLifecycleCategory(item.Category) && isLifecycleCategory(sourceCategory) {
			if resolutionMatches(stripCarryForwardSince(item.Text), stripCarryForwardSince(resolvedText)) {
				delete(active, key)
			}
			continue
		}
		if item.Category == sourceCategory && resolutionMatches(
			stripCarryForwardSince(item.Text), stripCarryForwardSince(resolvedText)) {
			delete(active, key)
		}
	}
}

// getOpenItems returns today's open (unresolved) tracked items.
func getOpenItems() []OpenItem {
	return parseOpenItems(readLedgerLines())
}

// recordConvertedDone logs a DONE entry referencing an original open
// item's text, marking it as resolved (FR-07). The original line is
// left untouched (append-only ledger design). item.Text's leading
// word is normalized to the terminal past tense (inflectLifecycleText,
// pastverb.go) -- Plan items are typically phrased as imperatives
// ("Fix the login bug"), which reads oddly once marked DONE, so this
// converts it to "Fixed the login bug" instead. Purely cosmetic;
// harmless no-op if the leading word isn't a recognized/regular verb.
func recordConvertedDone(item OpenItem) {
	recordActivity(inflectLifecycleText(item.Text, "DONE")+convertedSuffix(item.Category), "DONE")
}

// inflectLifecycleText normalizes a lifecycle item's leading verb for its
// target category: TODO is base/future-facing, DOING is present participle,
// and terminal categories use simple past. Resolution metadata is removed
// before inflection and is added by the transition that needs it.
func inflectLifecycleText(text, category string) string {
	flags := extractFlags(text)
	if !isLifecycleCategory(category) && !isLifecycleEndpoint(category) {
		return text
	}
	base := BaseTenseLeadingWord(stripResolutionSuffix(stripFlags(text)))
	var result string
	switch {
	case category == "DOING":
		result = PresentParticipleLeadingWord(base)
	case isLifecycleEndpoint(category):
		result = PastTenseLeadingWord(base)
	default:
		result = base
	}
	return setFlags(result, flags)
}

func transitionLifecycleText(text, fromCategory, toCategory string) string {
	// BaseTenseLeadingWord is intentionally applied regardless of the source
	// category, making this safe for text already normalized for the selector
	// or for older ledger rows that still contain their original imperative.
	return inflectLifecycleText(text, toCategory)
}

// completePlannedItem changes a TODO/DOING row to DONE in place, preserving
// its timestamp, task text, and cumulative minutes. The marker lets
// historical carry-forward copies recognize the lifecycle completion.
func completePlannedItem(item OpenItem) error {
	return completePlannedEndpoint(item, "DONE", item.Text)
}

// completePlannedEndpoint changes a lifecycle row to a selected terminal
// endpoint in place. The source marker keeps the logical open item resolved
// without changing the row's timestamp or accumulated duration.
func completePlannedEndpoint(item OpenItem, endpoint, text string) error {
	if !isLifecycleCategory(item.Category) || item.LineIndex < 0 {
		return nil
	}
	if !isLifecycleEndpoint(endpoint) {
		return nil
	}
	return replaceLedgerItemAt(item, endpoint,
		strings.TrimSpace(inflectLifecycleText(text, endpoint))+convertedSuffix(item.Category))
}

// startPlannedItem changes a TODO row to DOING in place. Repeated attempts
// against an already active item are harmless.
func startPlannedItem(item OpenItem) error {
	if item.Category != "TODO" || item.LineIndex < 0 {
		return nil
	}
	return replaceLedgerItemAt(item, "DOING", transitionLifecycleText(item.Text, item.Category, "DOING"))
}

// dittoLifecycleItem turns the latest DONE lifecycle row back into DOING or
// increments an existing DOING row. It never appends a second lifecycle row.
func dittoLifecycleItem(item OpenItem, delta int) error {
	if item.LineIndex < 0 {
		return nil
	}
	if item.Category == "DONE" {
		return replaceLedgerItemAt(item, "DOING", transitionLifecycleText(item.Text, item.Category, "DOING"))
	}
	if item.Category == "DOING" && delta > 0 {
		return replaceLedgerItemTextAt(item, incrementEntryMins(item.Text, delta))
	}
	return nil
}

// recordPostponed logs a SOMEDAY entry referencing an original open
// item's text, marking it as resolved without pretending it was
// completed -- for deliberately deferring an item so the Upcoming
// list doesn't grow unbounded. The original line is left untouched.
func recordPostponed(item OpenItem) {
	recordActivity(inflectLifecycleText(item.Text, "TODO")+convertedSuffix(item.Category), "SOMEDAY")
}

// recordDiscarded logs a DISCARDED entry referencing an original open
// item's text, marking it as resolved via outright dismissal (the
// "Nah" button) -- distinct from Postpone (SOMEDAY, meant to revisit
// later) since a discarded item isn't expected to come back. The
// original line is left untouched.
func recordDiscarded(item OpenItem) {
	recordActivity(inflectLifecycleText(item.Text, "TODO")+convertedSuffix(item.Category), "DISCARDED")
}

// groupOpenItemsByCategory buckets items by category, preserving
// openTrackedCategories order, and skips empty buckets. Shared by
// Daybook's Upcoming section, SOD, and SOM so all three list open
// items (TODO/DOING/GOAL/WAITING/QUESTION/FIXME/RISK) the same way, rather
// than each hardcoding its own TODO-vs-GOAL binary split. Within each
// category bucket, items are ordered by primary tag (see
// primaryTagSortKey) rather than left in ledger/first-seen order --
// grouping same-tagged items together makes a category's items
// easier to scan when several distinct projects/tags are mixed
// together within it. Within each tag group, rows remain chronological.
func groupOpenItemsByCategory(items []OpenItem) (categories []string, grouped map[string][]OpenItem) {
	grouped = make(map[string][]OpenItem)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	for _, cat := range openTrackedCategories {
		if len(grouped[cat]) > 0 {
			categories = append(categories, cat)
			sortItemsByPrimaryTag(grouped[cat])
		}
	}
	return categories, grouped
}

// primaryTagSortKey returns the last #tag found in text, or "" if text
// has no tag. The last tag is the primary tag because users can place
// tags anywhere in a sentence while still adopting a natural
// verb-first, tags-at-the-end entry style.
func primaryTagSortKey(text string) string {
	tags := extractTags(text)
	if len(tags) == 0 {
		return ""
	}
	return tags[len(tags)-1]
}

// sortItemsByPrimaryTag sorts items in place by primaryTagSortKey,
// tagged items first (alphabetically by tag), untagged items
// (empty key) last -- the opposite of plain string comparison's
// default "" first ordering, since untagged items are meant to read
// as lower-priority/less-organized than anything with a tag. Items
// sharing a tag are then ordered by their ledger timestamp.
func sortItemsByPrimaryTag(items []OpenItem) {
	sort.SliceStable(items, func(i, j int) bool {
		aImportant := hasFlag(items[i].Text, "!!")
		bImportant := hasFlag(items[j].Text, "!!")
		if aImportant != bImportant {
			return aImportant
		}
		a, b := primaryTagSortKey(items[i].Text), primaryTagSortKey(items[j].Text)
		if a == "" && b == "" {
			return items[i].Time.Before(items[j].Time)
		}
		if a == "" {
			return false // untagged never sorts before a tagged item
		}
		if b == "" {
			return true // tagged always sorts before an untagged item
		}
		if a != b {
			return a < b
		}
		if items[i].Time.IsZero() || items[j].Time.IsZero() {
			return false
		}
		return items[i].Time.Before(items[j].Time)
	})
}

// isExcludedTagItem reports whether item's primary tag
// (primaryTagSortKey) matches one of excludeTags (case-insensitive,
// same normalization as summarize.go's lineHasExcludedTag) -- used to
// push items tagged with a user-configured "noise" tag
// (Config.ReportExcludeTags) below even the untagged items in
// Planned/Endings/Hilites, hidden by default behind a "Show all"
// toggle (mirroring Planned's existing non-TODO-category toggle).
// Untagged items are never excluded regardless of excludeTags.
func isExcludedTagItem(text string, excludeTags []string) bool {
	tag := primaryTagSortKey(text)
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
// sortItemsByPrimaryTag) into visible (shown by default) and excluded
// (primary tag matches excludeTags, hidden behind a "Show all"
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

// categoryIconPrefix returns the category's icon plus a separating space
// for categorized item lists. Unknown categories retain the old bullet so
// callers remain readable if historical data contains an old code.
func categoryIconPrefix(cat string) string {
	if emoji := EmojiForCode(cat); emoji != "" {
		return emoji + " "
	}
	return "\u2022 "
}

// lastLifecycleItem returns the most recently logged DONE or DOING entry,
// with lifecycle metadata stripped and its current line index. It remains
// useful to history-oriented callers; the live Ditto control uses the
// narrower lastDoingItem lookup below.
func lastLifecycleItem() (item OpenItem, ok bool) {
	lines := readLedgerLines()
	for i := len(lines) - 1; i >= 0; i-- {
		cat, text, lineOk := parseLedgerLine(lines[i])
		if !lineOk || (cat != "DONE" && cat != "DOING") {
			continue
		}
		return OpenItem{Category: cat, Text: stripResolutionSuffix(text), LineIndex: i}, true
	}
	return OpenItem{}, false
}

// lastDoingItem returns the most recently logged active DOING entry,
// with lifecycle metadata stripped and its current line index. Ditto
// uses this narrower lookup so it extends work that is still active
// instead of resurrecting an already completed DONE item.
func lastDoingItem() (item OpenItem, ok bool) {
	lines := readLedgerLines()
	for i := len(lines) - 1; i >= 0; i-- {
		cat, text, lineOk := parseLedgerLine(lines[i])
		if !lineOk || cat != "DOING" {
			continue
		}
		return OpenItem{Category: cat, Text: stripResolutionSuffix(text), LineIndex: i}, true
	}
	return OpenItem{}, false
}

// lastDoneItem returns the most recently logged DONE entry (text with
// any "(via CATEGORY)" suffix stripped, plus its LineIndex into
// readLedgerLines() for in-place rewriting), or ok=false if there is
// no DONE entry logged yet today. This is retained for callers that need
// the latest terminal item without considering active DOING work.
func lastDoneItem() (item OpenItem, ok bool) {
	lines := readLedgerLines()
	for i := len(lines) - 1; i >= 0; i-- {
		cat, text, lineOk := parseLedgerLine(lines[i])
		if !lineOk || cat != "DONE" {
			continue
		}
		text = stripResolutionSuffix(text)
		return OpenItem{Category: cat, Text: text, LineIndex: i}, true
	}
	return OpenItem{}, false
}

func stripResolutionSuffix(text string) string {
	for _, srcCat := range append(append([]string{}, openTrackedCategories...), somedayCategory) {
		text = strings.TrimSuffix(text, convertedSuffix(srcCat))
	}
	return text
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
		text = stripResolutionSuffix(text)
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
// openTrackedCategories ("plan"). EODOnly metadata such as PRODUCTIVITY
// and MEETING_HOURS is deliberately omitted from Hilites. Strips the
// "(via CATEGORY)" suffix
// (see convertedSuffix) from DONE entries converted from an open
// item, same as getCompletedItems did -- harmless no-op for any other
// category, which never carries that suffix.
func getCategoryGroupItems(group string) []OpenItem {
	codes := make(map[string]bool)
	for _, c := range Categories {
		if c.Group != group || c.EODOnly {
			continue
		}
		codes[c.Code] = true
	}
	var out []OpenItem
	today := time.Now()
	for i, line := range readLedgerLines() {
		cat, text, ok := parseLedgerLine(line)
		if !ok || !codes[cat] {
			continue
		}
		text = stripResolutionSuffix(text)
		stamp, _ := parseLedgerLineTime(line, today)
		out = append(out, OpenItem{Category: cat, Text: text, Time: stamp, LineIndex: i})
	}
	return out
}

// groupCategoryItemsByGroup buckets items (as returned by
// getCategoryGroupItems) by category, preserving group's
// categoryGroupOrder and skipping empty buckets -- the general-
// purpose sibling of groupOpenItemsByCategory, letting Endings and
// Hilites show per-category sub-headings the same way Planned
// already does. Also sorts each bucket by primary tag, same as
// groupOpenItemsByCategory (see sortItemsByPrimaryTag).
func groupCategoryItemsByGroup(group string, items []OpenItem) (categories []string, grouped map[string][]OpenItem) {
	grouped = make(map[string][]OpenItem)
	for _, item := range items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	for _, cat := range categoryGroupOrder(group) {
		if len(grouped[cat]) > 0 {
			categories = append(categories, cat)
			sortItemsByPrimaryTag(grouped[cat])
		}
	}
	return categories, grouped
}
