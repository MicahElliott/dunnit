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
	// Mins is parsed from a " @Nm" suffix in Text (see ui.go's withMins),
	// including when lifecycle metadata follows it. 0 if absent/invalid.
	// Note this does NOT
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

// entryMinsPattern matches a minutes token (e.g. "@20m"). Boundary
// validation happens in entryMinsMatch because Go's regexp package
// deliberately does not support look-around assertions.
var entryMinsPattern = regexp.MustCompile(`@(\d+)m`)

// parseEntryMins returns the minutes value from a valid " @Nm" token
// in text, or 0 if absent/invalid. Lifecycle metadata such as
// " (via DOING)" may follow the token.
func parseEntryMins(text string) int {
	_, _, n, ok := entryMinsMatch(text)
	if !ok {
		return 0
	}
	return n
}

// entryMinsMatch returns the last valid minutes token and its byte span.
// A token is valid only when separated from surrounding text by whitespace
// or a string boundary, so prose such as "@20minutes" is left untouched.
func entryMinsMatch(text string) (start, end, mins int, ok bool) {
	matches := entryMinsPattern.FindAllStringSubmatchIndex(text, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match[0], match[1]
		if start > 0 && text[start-1] != ' ' && text[start-1] != '\t' {
			continue
		}
		if end < len(text) && text[end] != ' ' && text[end] != '\t' {
			continue
		}
		n, err := strconv.Atoi(text[match[2]:match[3]])
		if err != nil {
			continue
		}
		return start, end, n, true
	}
	return 0, 0, 0, false
}

// replaceEntryMins replaces the existing valid minutes token or appends one
// before known trailing lifecycle/carry-forward metadata. Existing malformed
// duration text is preserved and treated as absent.
func replaceEntryMins(text string, mins int) string {
	if mins < 0 {
		return text
	}
	if start, end, _, ok := entryMinsMatch(text); ok {
		return text[:start] + "@" + strconv.Itoa(mins) + "m" + text[end:]
	}
	leading := len(text) - len(strings.TrimLeft(text, " \t"))
	trailing := len(text) - len(strings.TrimRight(text, " \t"))
	coreEnd := len(text) - trailing
	core := text[leading:coreEnd]
	insertAt := len(core)
	for _, suffix := range []string{" (via TODO)", " (via DOING)"} {
		if idx := strings.LastIndex(core, suffix); idx >= 0 && idx < insertAt && idx+len(suffix) == len(core) {
			insertAt = idx
		}
	}
	if since := strings.LastIndex(core, " (since "); since >= 0 && strings.HasSuffix(core, ")") && since < insertAt {
		insertAt = since
	}
	return text[:leading] + core[:insertAt] + " @" + strconv.Itoa(mins) + "m" + core[insertAt:] + text[coreEnd:]
}

// incrementEntryMins adds delta to a valid existing minutes value. If the
// prior value is absent or malformed, it is treated as zero while all other
// text remains unchanged.
func incrementEntryMins(text string, delta int) string {
	if delta <= 0 {
		return text
	}
	_, _, current, ok := entryMinsMatch(text)
	if !ok {
		current = 0
	}
	return replaceEntryMins(text, current+delta)
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
