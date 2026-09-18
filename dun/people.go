package dun

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// personPattern matches a person marker such as @Brandon or @Surbhi.
// Names must start with a letter and may contain letters, digits,
// underscores, and hyphens. A boundary check in extractPeople keeps
// ordinary text such as AT&T from turning the trailing T into a person.
var personPattern = regexp.MustCompile(`@[\p{L}][\p{L}\p{N}_-]*`)

func isPersonMatchBoundary(text string, start int) bool {
	if start == 0 {
		return true
	}
	previous, _ := utf8.DecodeLastRuneInString(text[:start])
	return !unicode.IsLetter(previous) && !unicode.IsDigit(previous) && previous != '_'
}

// extractPeople returns distinct @person tokens in first-seen order.
// The original spelling is preserved so the ledger remains exactly what
// the user typed; callers that compare people should use personKey.
func extractPeople(text string) []string {
	seen := make(map[string]bool)
	var people []string
	for _, match := range personPattern.FindAllStringIndex(text, -1) {
		if !isPersonMatchBoundary(text, match[0]) {
			continue
		}
		person := text[match[0]:match[1]]
		key := personKey(person)
		if !seen[key] {
			seen[key] = true
			people = append(people, person)
		}
	}
	return people
}

func personKey(person string) string {
	return strings.ToLower(person)
}

func containsPersonFold(people []string, want string) bool {
	want = personKey(want)
	for _, person := range people {
		if personKey(person) == want {
			return true
		}
	}
	return false
}

// personStat is the person equivalent of tagStat. The label preserves the
// most recently seen spelling for compact UI insertion.
type personStat struct {
	label       string
	count       int
	recentCount int
	score       float64
	lastSeen    time.Time
}

// gatherPersonStats scans the shared ledger index once and computes a
// frequency+recency score for each person marker.
func gatherPersonStats() map[string]*personStat {
	stats := map[string]*personStat{}
	now := time.Now()
	for _, entry := range AllLedgerEntries() {
		for _, person := range entry.People {
			key := personKey(person)
			stat := stats[key]
			if stat == nil {
				stat = &personStat{}
				stats[key] = stat
			}
			stat.count++
			daysSince := now.Sub(entry.Date).Hours() / 24
			if daysSince < 0 {
				daysSince = 0
			}
			stat.score += math.Pow(0.5, daysSince/tagRecencyHalfLife)
			if daysSince < tagRecentWindowDays {
				stat.recentCount++
			}
			if stat.label == "" || entry.Date.After(stat.lastSeen) {
				stat.label = person
				stat.lastSeen = entry.Date
			}
		}
	}
	finalizePersonStats(stats, now)
	return stats
}

func finalizePersonStats(stats map[string]*personStat, now time.Time) {
	for _, stat := range stats {
		daysSinceLast := now.Sub(stat.lastSeen).Hours() / 24
		if daysSinceLast < 0 {
			daysSinceLast = 0
		}
		recency := math.Pow(0.5, daysSinceLast/tagRecencyHalfLife)
		frequency := math.Log1p(math.Min(stat.score, tagFrequencyCap))
		stat.score = tagRecencyWeight*recency + tagFrequencyWeight*frequency
	}
}

func rankPeopleByScore(stats map[string]*personStat) []string {
	people := make([]string, 0, len(stats))
	for person := range stats {
		people = append(people, person)
	}
	sort.Slice(people, func(i, j int) bool {
		left, right := stats[people[i]], stats[people[j]]
		if left.score != right.score {
			return left.score > right.score
		}
		return left.label < right.label
	})
	return people
}

func commonPeopleWithStats(limit int) (people []string, stats map[string]*personStat) {
	stats = gatherPersonStats()
	ranked := rankPeopleByScore(stats)
	if limit >= 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	return ranked, stats
}

func formatPersonWithCount(person string, stat *personStat) string {
	if stat == nil {
		return person
	}
	return fmt.Sprintf("%s(%d)", stat.label, stat.count)
}

func personUsageTooltip(stat *personStat) string {
	if stat == nil {
		return ""
	}
	return fmt.Sprintf("Mentioned %d times in the last %d days", stat.recentCount,
		tagRecentWindowDays)
}

func hasPeople(text string) bool {
	return len(extractPeople(text)) > 0
}
