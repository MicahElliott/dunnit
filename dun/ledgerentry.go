package dun

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// LedgerEntry is one parsed line from a ledger file -- the shared
// unit that search/tags/trend/navigator features should read from
// instead of each independently re-parsing raw ledger text. See
// docs/ledger-index-design.md for the fuller design discussion behind
// introducing this (2026-09-02 navigator groundwork).
type LedgerEntry struct {
	// Date is the day component only, taken from the ledger file's
	// name (ledgerFileDate) -- cheap and reliable, doesn't depend on
	// the line's own "[HH:MM:SS]" stamp parsing successfully.
	Date time.Time
	// Time is Date combined with the line's "[HH:MM:SS]" stamp, via
	// the same parsing logic as parseLedgerLineTime. Zero value
	// (time.Time{}) if the line's stamp didn't parse.
	Time time.Time
	// Category and Text are the line's category code and the
	// remaining free text (tags still embedded in Text, not
	// stripped out) -- same split as parseLedgerLine.
	Category string
	Text     string
	// Tags is Text's #tag tokens, pre-extracted via extractTags so
	// callers don't need to re-run the regex themselves.
	Tags []string
	// Mins is parsed from a trailing " @Nm" suffix in Text (see
	// ui.go's withMins), 0 if absent/invalid. Note this does NOT
	// strip the "@Nm" substring back out of Text -- Text stays the
	// full original string as written to the ledger.
	Mins int
	// Source is the ledger file path this entry came from, and Line
	// is its 0-based line number within that file -- both provided
	// for "jump to ledger"/context-display purposes (e.g. search
	// results, same role summarize.go's callers use filepath.Base for
	// today).
	Source string
	Line   int
}

// entryMinsPattern matches a trailing " @Nm" mins suffix (e.g.
// " @20m") at the end of a ledger line's text, mirroring the format
// ui.go's withMins appends.
var entryMinsPattern = regexp.MustCompile(`@(\d+)m$`)

// parseEntryMins returns the minutes value from a trailing " @Nm"
// suffix in text, or 0 if absent/invalid.
func parseEntryMins(text string) int {
	m := entryMinsPattern.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// parseLedgerEntry parses one raw ledger line into a LedgerEntry,
// given the day (from the source file's name) it belongs to and its
// source file path/line number. Returns ok=false if the line doesn't
// look like a well-formed ledger entry (same shape parseLedgerLine
// already checks for).
func parseLedgerEntry(line string, date time.Time, source string, lineNum int) (entry LedgerEntry, ok bool) {
	category, text, parsedOK := parseLedgerLine(line)
	if !parsedOK {
		return LedgerEntry{}, false
	}
	t, timeOK := parseLedgerLineTime(line, date)
	if !timeOK {
		t = time.Time{}
	}
	return LedgerEntry{
		Date:     date,
		Time:     t,
		Category: category,
		Text:     text,
		Tags:     extractTags(text),
		Mins:     parseEntryMins(text),
		Source:   source,
		Line:     lineNum,
	}, true
}
