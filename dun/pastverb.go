package dun

import "strings"

// pastverb.go implements a small, dependency-free English past-tense
// converter for exactly one purpose: when a Plan-group item (TODO/
// IDEA/GOAL/etc, typically phrased as an imperative, e.g. "Fix the
// login bug") is converted to DONE (see recordConvertedDone,
// todos.go), the leading verb is flipped to past tense ("Fixed the
// login bug") so completed items read naturally rather than staying
// stuck in imperative mood forever.
//
// Deliberately NOT a general-purpose inflection library (no
// pluralization, no articles, no full irregular-verb coverage) --
// just enough common English verbs + regular-verb spelling rules to
// handle typical short TODO-style phrasing. Micah considered adding
// github.com/cv/go-inflect for this but it pulls in golang.org/x/text
// plus test-only transitive deps (golang.org/x/perf, testify) for a
// ~100-function library when only one function's worth of behavior
// is actually wanted here -- not worth the added module-graph/binary
// surface for this narrow use, so this file is a small from-scratch
// equivalent instead.

// irregularPastTense maps common irregular verbs (lowercase) to their
// simple past tense form. Deliberately narrow: the ~70 or so verbs
// most likely to lead a short TODO/GOAL/FIXME-style phrase (per
// Micah's own TODOs), not an exhaustive list of all English
// irregulars.
var irregularPastTense = map[string]string{
	"go": "went", "do": "did", "make": "made", "take": "took",
	"write": "wrote", "send": "sent", "build": "built", "buy": "bought",
	"bring": "brought", "think": "thought", "meet": "met", "run": "ran",
	"get": "got", "give": "gave", "find": "found", "tell": "told",
	"sell": "sold", "teach": "taught", "catch": "caught", "keep": "kept",
	"leave": "left", "feel": "felt", "hold": "held", "read": "read",
	"set": "set", "put": "put", "cut": "cut", "hit": "hit",
	"let": "let", "cost": "cost", "spend": "spent", "lend": "lent",
	"deal": "dealt", "mean": "meant", "draw": "drew", "drive": "drove",
	"eat": "ate", "fall": "fell", "fly": "flew", "forget": "forgot",
	"freeze": "froze", "grow": "grew", "hang": "hung", "hear": "heard",
	"hide": "hid", "know": "knew", "lead": "led", "lose": "lost",
	"pay": "paid", "ride": "rode", "ring": "rang", "rise": "rose",
	"see": "saw", "seek": "sought", "shake": "shook", "shrink": "shrank",
	"shut": "shut", "sing": "sang", "sink": "sank", "sit": "sat",
	"sleep": "slept", "speak": "spoke", "spring": "sprang", "stand": "stood",
	"steal": "stole", "stick": "stuck", "sting": "stung", "strike": "struck",
	"swear": "swore", "sweep": "swept", "swim": "swam", "swing": "swung",
	"throw": "threw", "understand": "understood", "wake": "woke",
	"wear": "wore", "win": "won", "wind": "wound", "wring": "wrung",
}

// isVowel reports whether r is an English vowel letter (lowercase
// check only -- callers always pass an already-lowercased rune).
func isVowel(r rune) bool {
	switch r {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	}
	return false
}

// countSyllables is a deliberately crude vowel-group counter (counts
// runs of consecutive vowels as one syllable each), good enough to
// distinguish "short, one-syllable" verbs like "stop"/"plan" (where
// the final consonant doubles: stopped/planned) from longer verbs
// like "listen"/"happen" (where it doesn't) -- not a linguistically
// accurate syllable counter, just enough for this narrow purpose.
func countSyllables(lower string) int {
	count := 0
	inVowelGroup := false
	for _, r := range lower {
		v := isVowel(r)
		if v && !inVowelGroup {
			count++
		}
		inVowelGroup = v
	}
	if count == 0 {
		count = 1 // e.g. "cry" has no vowel-letter in this simple scheme's sense... actually 'y' isn't counted as a vowel here; treat as 1 syllable minimum
	}
	return count
}

// shouldDoubleFinalConsonant reports whether verb's final consonant
// should double before adding "ed" (CVC pattern in a short,
// one-syllable word, e.g. "stop" -> "stopped", "plan" -> "planned"),
// per the standard English spelling rule. lower must already be
// lowercased.
func shouldDoubleFinalConsonant(lower string) bool {
	if len(lower) < 3 || countSyllables(lower) != 1 {
		return false
	}
	last := rune(lower[len(lower)-1])
	secondLast := rune(lower[len(lower)-2])
	thirdLast := rune(lower[len(lower)-3])
	if isVowel(last) || last == 'w' || last == 'x' || last == 'y' {
		return false
	}
	if !isVowel(secondLast) {
		return false
	}
	if isVowel(thirdLast) {
		return false // VVC (e.g. "eat"), don't double
	}
	return true
}

// matchCase re-applies the casing pattern of original onto replacement
// (both assumed ASCII-letters-only, as English verbs are): all-caps
// original -> all-caps replacement, capitalized-first-letter original
// -> capitalized replacement, otherwise (including all-lowercase)
// left as given (already-lowercase from the rule tables/functions
// that produce replacement).
func matchCase(original, replacement string) string {
	if original == strings.ToUpper(original) {
		return strings.ToUpper(replacement)
	}
	if len(original) > 0 && original[:1] == strings.ToUpper(original[:1]) {
		return strings.ToUpper(replacement[:1]) + replacement[1:]
	}
	return replacement
}

// PastTense returns the simple past tense of an English verb,
// preserving the original's casing pattern (all-caps, capitalized, or
// lowercase). Applies irregularPastTense first, then regular spelling
// rules (-e -> -ed, consonant+y -> -ied, CVC doubling, default -ed).
// Returns verb unchanged if it's empty or contains non-letter
// characters (so it's safe to call on arbitrary leading "words" that
// might actually be punctuation/numbers/tags).
func PastTense(verb string) string {
	if verb == "" {
		return verb
	}
	for _, r := range verb {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return verb
		}
	}
	lower := strings.ToLower(verb)

	if past, ok := irregularPastTense[lower]; ok {
		return matchCase(verb, past)
	}

	switch {
	case strings.HasSuffix(lower, "e"):
		return matchCase(verb, lower+"d")
	case strings.HasSuffix(lower, "y") && len(lower) > 1 && !isVowel(rune(lower[len(lower)-2])):
		return matchCase(verb, lower[:len(lower)-1]+"ied")
	case shouldDoubleFinalConsonant(lower):
		return matchCase(verb, lower+string(lower[len(lower)-1])+"ed")
	default:
		return matchCase(verb, lower+"ed")
	}
}

// PastTenseLeadingWord converts only the first whitespace-delimited
// word of text to past tense (via PastTense), leaving everything else
// -- including all remaining whitespace/punctuation/words -- exactly
// as given. No-ops (returns text unchanged) if text is empty or has
// no leading word. Used by recordConvertedDone (todos.go) to flip a
// Plan item's imperative-phrased leading verb ("Fix the login bug")
// to past tense ("Fixed the login bug") when it resolves to DONE.
func PastTenseLeadingWord(text string) string {
	idx := strings.IndexAny(text, " \t")
	if idx == -1 {
		return PastTense(text)
	}
	return PastTense(text[:idx]) + text[idx:]
}
