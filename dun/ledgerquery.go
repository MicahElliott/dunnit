package dun

import (
	"strings"
	"time"
)

// LedgerQuery composes filters across LedgerEntry -- the single
// query shape every search/tag/trend/navigator feature should build
// and pass to FilterLedgerEntries, rather than each hand-rolling its
// own scan+match logic. Zero value matches everything.
type LedgerQuery struct {
	// Categories restricts matches to entries whose Category is one
	// of these codes (case-sensitive, matching ledger convention).
	// Empty means "any category".
	Categories []string
	// Tags restricts matches to entries containing ALL of these tags
	// (case-insensitive, "#" prefix required, matching extractTags'
	// format). Empty means "any tags (or none)".
	Tags []string
	// From/To bound Date, inclusive on both ends. Zero value for
	// either means unbounded in that direction.
	From, To time.Time
	// Text, if non-empty, must appear as a case-insensitive substring
	// of the entry's Text.
	Text string
}

// Matches reports whether e satisfies every filter set on q.
func (q LedgerQuery) Matches(e LedgerEntry) bool {
	if len(q.Categories) > 0 && !containsString(q.Categories, e.Category) {
		return false
	}
	if !q.From.IsZero() && e.Date.Before(q.From) {
		return false
	}
	if !q.To.IsZero() && e.Date.After(q.To) {
		return false
	}
	if q.Text != "" && !strings.Contains(strings.ToLower(e.Text), strings.ToLower(q.Text)) {
		return false
	}
	for _, want := range q.Tags {
		if !containsTagFold(e.Tags, want) {
			return false
		}
	}
	return true
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func containsTagFold(tags []string, want string) bool {
	for _, t := range tags {
		if strings.EqualFold(t, want) {
			return true
		}
	}
	return false
}

// FilterLedgerEntries returns entries from AllLedgerEntries()
// matching q, in the same (chronological, oldest-first) order.
func FilterLedgerEntries(q LedgerQuery) []LedgerEntry {
	var out []LedgerEntry
	for _, e := range AllLedgerEntries() {
		if q.Matches(e) {
			out = append(out, e)
		}
	}
	return out
}
