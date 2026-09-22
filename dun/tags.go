package dun

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// tagPattern matches a "#tag" token: '#' followed by one or more
// word-ish characters (letters, digits, underscore, hyphen, colon --
// covers things like "#pts:3").
var tagPattern = regexp.MustCompile(`#[\w:-]+`)

// numericTagPattern matches a tag that's strictly digits after the
// "#" (e.g. "#12345") -- typically an accidental tag (a phone number,
// ticket number, or similar pasted-in digits rather than an
// intentional tag), so these are specifically filtered down to just
// the single most recent one in tag-listing UI (see
// filterNumericTags) rather than cluttering "Common tags:"/"Show all"
// with every numeric one ever used.
var numericTagPattern = regexp.MustCompile(`^#\d+$`)

func isNumericTag(tag string) bool {
	return numericTagPattern.MatchString(tag)
}

// extractTags returns all distinct #tag tokens found in text, in
// first-seen order.
func extractTags(text string) []string {
	seen := make(map[string]bool)
	var tags []string
	for _, m := range tagPattern.FindAllString(text, -1) {
		if !seen[m] {
			seen[m] = true
			tags = append(tags, m)
		}
	}
	return tags
}

// splitPrimaryTag returns the last tag in text and the display text with
// that one occurrence removed. The ledger keeps the original text; this
// helper only changes the Daybook presentation.
func splitPrimaryTag(text string) (tag, body string) {
	matches := tagPattern.FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return "", text
	}
	last := matches[len(matches)-1]
	return text[last[0]:last[1]], strings.TrimSpace(text[:last[0]] + text[last[1]:])
}

// tagCache holds a scanned + deduplicated list of every #tag seen
// across all ledger files under DunnitDir(), refreshed at most every
// tagCacheTTL rather than rescanning on every keystroke (FR-10).
type tagCache struct {
	mu        sync.Mutex
	tags      []string
	scannedAt time.Time
}

const tagCacheTTL = 5 * time.Minute

var globalTagCache tagCache

// KnownTags returns the cached list of all distinct tags seen across
// ledger history, rescanning if the cache is empty or stale.
func KnownTags() []string {
	globalTagCache.mu.Lock()
	defer globalTagCache.mu.Unlock()
	if globalTagCache.tags == nil || time.Since(globalTagCache.scannedAt) > tagCacheTTL {
		globalTagCache.tags = scanAllTags()
		globalTagCache.scannedAt = time.Now()
	}
	return globalTagCache.tags
}

// InvalidateTagCache forces the next KnownTags() call to rescan,
// useful right after recording a new entry that might contain a tag
// not seen before.
func InvalidateTagCache() {
	globalTagCache.mu.Lock()
	defer globalTagCache.mu.Unlock()
	globalTagCache.tags = nil
}

// scanAllTags collects every distinct #tag across all ledger entries
// (via AllLedgerEntries(), the shared index -- see ledgerindex.go),
// sorted alphabetically.
func scanAllTags() []string {
	seen := make(map[string]bool)
	for _, e := range AllLedgerEntries() {
		for _, tag := range e.Tags {
			seen[tag] = true
		}
	}
	tags := make([]string, 0, len(seen))
	for t := range seen {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

// tagRecencyHalfLife controls how fast a tag's contribution to a
// frecency score decays with age. Each occurrence's weight halves
// every tagRecencyHalfLife days. The short half-life, together with
// the last-use component in finalizeTagStats, keeps a heavily-used
// historical tag from crowding out a recently-used tag.
const tagRecencyHalfLife = 7.0

const (
	tagRecentWindowDays = 30
	tagRecencyWeight    = 0.70
	tagFrequencyWeight  = 0.30
	tagFrequencyCap     = 10.0
)

// tagStat holds a tag's usage count and frecency score (see
// gatherTagStats), for display ("(count)") and ranking.
type tagStat struct {
	count       int
	recentCount int
	score       float64
	lastSeen    time.Time
}

// gatherTagStats scans all ledger history once (via
// AllLedgerEntries(), the shared index) and returns, per tag, its
// total occurrence count and a frecency score -- each occurrence's
// weight exponentially decays with the age of that occurrence
// (half-life tagRecencyHalfLife days), so a tag's score is dominated
// by recent usage rather than lifetime total (see git log 2026-08-31
// fix for why: an earlier version let heavy historical-but-stale
// usage permanently outrank genuinely recent tags).
func gatherTagStats() map[string]*tagStat {
	return gatherTagStatsFromEntries(deduplicateCarryForwardEntries(AllLedgerEntries()), time.Now())
}

func gatherTagStatsFromEntries(entries []LedgerEntry, now time.Time) map[string]*tagStat {
	stats := map[string]*tagStat{}
	for _, e := range entries {
		if len(e.Tags) == 0 {
			continue
		}
		daysSince := now.Sub(e.Date).Hours() / 24
		if daysSince < 0 {
			daysSince = 0
		}
		weight := math.Pow(0.5, daysSince/tagRecencyHalfLife)
		for _, tag := range e.Tags {
			st := stats[tag]
			if st == nil {
				st = &tagStat{}
				stats[tag] = st
			}
			st.count++
			st.score += weight
			if daysSince < tagRecentWindowDays {
				st.recentCount++
			}
			if e.Date.After(st.lastSeen) {
				st.lastSeen = e.Date
			}
		}
	}
	finalizeTagStats(stats, now)
	return stats
}

// deduplicateCarryForwardEntries collapses daily copies of one TODO/DOING
// lineage into its newest ledger entry. Carry-forward rows retain the
// original date in an s/YYYY-MM-DD marker, which lets tag and people counts
// treat a task carried across several days as one logical use while still
// keeping its latest copy for recency scoring. Unmarked entries remain
// independent unless a matching carried lineage exists.
func deduplicateCarryForwardEntries(entries []LedgerEntry) []LedgerEntry {
	carriedKeys := make(map[string]bool)
	for _, entry := range entries {
		if _, ok := parseCarryForwardSince(entry.Text); !ok {
			continue
		}
		if key, ok := carryForwardEntryKey(entry); ok {
			carriedKeys[key] = true
		}
	}

	latest := make(map[string]int)
	for i, entry := range entries {
		key, ok := carryForwardEntryKey(entry)
		if !ok || !carriedKeys[key] {
			continue
		}
		latest[key] = i
	}

	out := make([]LedgerEntry, 0, len(entries))
	for i, entry := range entries {
		key, ok := carryForwardEntryKey(entry)
		if ok && carriedKeys[key] && latest[key] != i {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func carryForwardEntryKey(entry LedgerEntry) (string, bool) {
	if !isLifecycleCategory(entry.Category) && !isLifecycleEndpoint(entry.Category) {
		return "", false
	}
	date, carried := parseCarryForwardSince(entry.Text)
	if !carried {
		date = entry.Date
	}
	text := stripResolutionSuffix(stripCarryForwardSince(entry.Text))
	category := entry.Category
	if isLifecycleCategory(category) || isLifecycleEndpoint(category) {
		category = "TODO/DOING"
		text = BaseTenseLeadingWord(text)
	}
	return date.Format("2006-01-02") + "\x00" + category + "\x00" + text, true
}

// finalizeTagStats combines a log-scaled frequency signal with the
// recency of the tag's latest use. Log scaling prevents a large pile of
// old occurrences from overpowering a recent tag, while the recent
// window count remains available for hover text.
func finalizeTagStats(stats map[string]*tagStat, now time.Time) {
	for _, st := range stats {
		daysSinceLast := now.Sub(st.lastSeen).Hours() / 24
		if daysSinceLast < 0 {
			daysSinceLast = 0
		}
		recency := math.Pow(0.5, daysSinceLast/tagRecencyHalfLife)
		frequency := math.Log1p(math.Min(st.score, tagFrequencyCap))
		st.score = tagRecencyWeight*recency + tagFrequencyWeight*frequency
	}
}

// filterNumericTags drops all-but-the-most-recently-used numeric tag
// (e.g. "#12345") from a tag-stats map -- these are typically
// accidental (pasted ticket/phone numbers etc, not intentional tags),
// so only the single most recent one is worth surfacing in tag-
// listing UI; older ones would just be clutter. Non-numeric tags are
// untouched.
func filterNumericTags(stats map[string]*tagStat) map[string]*tagStat {
	var mostRecentNumeric string
	var mostRecentSeen time.Time
	for tag, st := range stats {
		if !isNumericTag(tag) {
			continue
		}
		if mostRecentNumeric == "" || st.lastSeen.After(mostRecentSeen) {
			mostRecentNumeric = tag
			mostRecentSeen = st.lastSeen
		}
	}
	out := make(map[string]*tagStat, len(stats))
	for tag, st := range stats {
		if isNumericTag(tag) && tag != mostRecentNumeric {
			continue
		}
		out[tag] = st
	}
	return out
}

// rankTagsByScore returns tags from stats sorted by descending
// frecency score (ties broken alphabetically for stability).
func rankTagsByScore(stats map[string]*tagStat) []string {
	tags := make([]string, 0, len(stats))
	for tag := range stats {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool {
		si, sj := stats[tags[i]].score, stats[tags[j]].score
		if si != sj {
			return si > sj
		}
		return tags[i] < tags[j]
	})
	return tags
}

// formatTagWithCount renders a tag with its usage count in
// parentheses, e.g. "#boss(12)".
func formatTagWithCount(tag string, st *tagStat) string {
	return fmt.Sprintf("%s(%d)", tag, st.count)
}

// commonAndRecentTags returns up to limit tags from ledger history,
// each formatted with its usage count (e.g. "#boss(12)"), ranked by
// frecency (blended frequency + recency, see gatherTagStats) with
// all-but-the-most-recent numeric tag (e.g. "#12345") filtered out
// per Micah's preference -- those are typically accidental/pasted
// digits, not intentional tags, and clutter this short list.
func commonAndRecentTags(limit int) []string {
	stats := filterNumericTags(gatherTagStats())
	ranked := rankTagsByScore(stats)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]string, len(ranked))
	for i, tag := range ranked {
		out[i] = formatTagWithCount(tag, stats[tag])
	}
	return out
}

// commonAndRecentTagsWithCounts is the same ranking/filtering as
// commonAndRecentTags, but returns the raw tag and its count
// separately (rather than a single pre-formatted string) -- used by
// ui.go's clickable "Frecent tags:" row, which needs the bare tag
// (e.g. "#boss") to insert into the entry box on click, plus the
// count to still display alongside it.
func commonAndRecentTagsWithCounts(limit int) (tags []string, counts []int) {
	tags, stats := commonAndRecentTagsWithStats(limit)
	counts = make([]int, len(tags))
	for i, tag := range tags {
		counts[i] = stats[tag].count
	}
	return tags, counts
}

// commonAndRecentTagsWithStats returns the ranked tags and their stats from
// one history scan, so callers can render both counts and hover details
// without rescanning the ledger for each tag.
func commonAndRecentTagsWithStats(limit int) (tags []string, stats map[string]*tagStat) {
	stats = filterNumericTags(gatherTagStats())
	ranked := rankTagsByScore(stats)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked, stats
}

func tagUsageTooltip(tag string, stat *tagStat) string {
	if stat == nil {
		return ""
	}
	word := "times"
	if stat.recentCount == 1 {
		word = "time"
	}
	return fmt.Sprintf("Used %d %s in the last %d days", stat.recentCount, word,
		tagRecentWindowDays)
}

// matchingTags returns tags from candidates that contain fragment as
// a case-insensitive substring, with prefix matches (fragment matches
// right after the tag's "#") sorted first, then other substring
// matches -- each group alphabetical. fragment should already have
// its leading "#" stripped. Returns nil if fragment is empty (no
// suggestions until the user has typed something after "#").
func matchingTags(candidates []string, fragment string) []string {
	if fragment == "" {
		return nil
	}
	fragment = strings.ToLower(fragment)
	var prefixMatches, otherMatches []string
	for _, tag := range candidates {
		lower := strings.ToLower(tag)
		body := strings.TrimPrefix(lower, "#")
		switch {
		case strings.HasPrefix(body, fragment):
			prefixMatches = append(prefixMatches, tag)
		case strings.Contains(lower, fragment):
			otherMatches = append(otherMatches, tag)
		}
	}
	if len(prefixMatches) == 0 && len(otherMatches) == 0 {
		return nil
	}
	return append(prefixMatches, otherMatches...)
}

// currentTagFragment inspects text up to cursor (a rune index) and,
// if the cursor is positioned within or immediately after an
// in-progress "#tag" token (i.e. the nearest "#" before the cursor has
// no whitespace between it and the cursor), returns that token's start
// offset and the fragment typed so far (including the "#"). ok is
// false if the cursor isn't in a tag-typing position.
func currentTagFragment(text string, cursor int) (start int, fragment string, ok bool) {
	runes := []rune(text)
	if cursor < 0 || cursor > len(runes) {
		return 0, "", false
	}
	i := cursor - 1
	for i >= 0 && runes[i] != '#' && !isTagBreak(runes[i]) {
		i--
	}
	if i < 0 || runes[i] != '#' {
		return 0, "", false
	}
	return i, string(runes[i:cursor]), true
}

func isTagBreak(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

// showAllTagsWindow shows every distinct tag ever seen across ledger
// history in a standalone scrollable window -- the escape hatch from
// Daybook's "Common tags:" line, which only shows a handful. Same
// frecency ranking, usage count, and numeric-tag filtering as
// commonAndRecentTags, just unlimited. Editing/deleting a tag across
// all its historical occurrences is a possible future extension (see
// tags.go's package doc) -- not implemented here, this is read-only.
func showAllTagsWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: All Tags")

	stats := filterNumericTags(gatherTagStats())
	ranked := rankTagsByScore(stats)
	list := container.NewVBox()
	if len(ranked) == 0 {
		list.Add(widget.NewLabel("No tags found in ledger history yet."))
	}
	for _, tag := range ranked {
		list.Add(widget.NewLabel(formatTagWithCount(tag, stats[tag])))
	}

	w.SetContent(windowPad(container.NewVScroll(list)))
	w.Resize(fyne.NewSize(300, 500))
	w.Show()
}
