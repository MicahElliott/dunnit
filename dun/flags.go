package dun

import (
	"regexp"
	"strings"
)

// ledgerFlag describes one fixed, non-customizable item flag. The code is
// stored in the text ledger; Icon and Label are used by Daybook's compact
// display and the Edit Entry toggles.
type ledgerFlag struct {
	Code  string
	Icon  string
	Label string
	Help  string
}

var ledgerFlags = []ledgerFlag{
	{"!!", "‼️", "Important", "Keep this item visible and protected from being forgotten."},
	{"??", "❓", "Needs clarification", "The task needs more thought or a clearer definition."},
	{"@@", "👥", "Follow up with someone", "Another person needs to be involved."},
	{"++", "↪️", "Follow-on work", "More work is expected after this item is resolved."},
}

// flagTokenPattern finds a flag at the start of text or after whitespace.
// The trailing boundary is checked in flagTokenMatches because Go's regexp
// package does not support look-ahead assertions. Keeping flags as standalone
// tokens means prose such as "Do the thing!!" remains ordinary text.
var flagTokenPattern = regexp.MustCompile(`(?:^|[ \t])(?:!!|\?\?|@@|\+\+)`)

type flagTokenMatch struct {
	start int
	end   int
	code  string
}

func flagTokenMatches(text string) []flagTokenMatch {
	var matches []flagTokenMatch
	for _, loc := range flagTokenPattern.FindAllStringIndex(text, -1) {
		start, end := loc[0], loc[1]
		if end < len(text) && text[end] != ' ' && text[end] != '\t' {
			continue
		}
		codeStart := start
		if text[codeStart] == ' ' || text[codeStart] == '\t' {
			codeStart++
		}
		matches = append(matches, flagTokenMatch{
			start: start,
			end:   end,
			code:  text[codeStart:end],
		})
	}
	return matches
}

// flagDefinitions returns the fixed flag registry in display and canonical
// storage order.
func flagDefinitions() []ledgerFlag { return ledgerFlags }

func flagDefinition(code string) (ledgerFlag, bool) {
	for _, flag := range ledgerFlags {
		if flag.Code == code {
			return flag, true
		}
	}
	return ledgerFlag{}, false
}

// extractFlags returns known standalone flags in the registry's stable order.
// Unknown punctuation is left as ordinary entry text.
func extractFlags(text string) []string {
	present := make(map[string]bool)
	for _, match := range flagTokenMatches(text) {
		present[match.code] = true
	}
	var flags []string
	for _, flag := range ledgerFlags {
		if present[flag.Code] {
			flags = append(flags, flag.Code)
		}
	}
	return flags
}

func hasFlag(text, code string) bool {
	for _, flag := range extractFlags(text) {
		if flag == code {
			return true
		}
	}
	return false
}

// stripFlags removes known flag tokens while preserving the surrounding
// prose as much as practical. It trims only the resulting outer whitespace.
func stripFlags(text string) string {
	matches := flagTokenMatches(text)
	for i := len(matches) - 1; i >= 0; i-- {
		match := matches[i]
		start, end := match.start, match.end
		if start == end {
			continue
		}
		if text[start] == ' ' || text[start] == '\t' {
			text = text[:start] + text[end:]
			continue
		}
		if end < len(text) && (text[end] == ' ' || text[end] == '\t') {
			end++
		}
		text = text[:start] + text[end:]
	}
	return strings.TrimSpace(text)
}

// setFlags replaces all known flags with the supplied set, placing them at
// the start of the entry text in stable order. Lifecycle text helpers use
// this canonical form so their leading verb remains easy to normalize after
// the flags are stripped.
func setFlags(text string, flags []string) string {
	text = stripFlags(text)
	if len(flags) == 0 {
		return text
	}
	var known []string
	for _, flag := range ledgerFlags {
		for _, want := range flags {
			if flag.Code == want {
				known = append(known, flag.Code)
				break
			}
		}
	}
	if len(known) == 0 {
		return text
	}
	return strings.Join(known, " ") + " " + text
}

func toggleFlag(text, code string, enabled bool) string {
	flags := extractFlags(text)
	filtered := make([]string, 0, len(flags)+1)
	for _, flag := range flags {
		if flag != code {
			filtered = append(filtered, flag)
		}
	}
	if enabled && !hasFlag(text, code) {
		filtered = append(filtered, code)
	}
	return setFlags(text, filtered)
}
