package dun

import (
	"fmt"
	"sort"
	"strings"
)

type reportMention struct {
	label string
	count int
}

func addReportMention(mentions map[string]reportMention, label string) {
	key := strings.ToLower(label)
	mention := mentions[key]
	if mention.label == "" {
		mention.label = label
	}
	mention.count++
	mentions[key] = mention
}

func rankedReportMentions(mentions map[string]reportMention) []reportMention {
	result := make([]reportMention, 0, len(mentions))
	for _, mention := range mentions {
		result = append(result, mention)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].count != result[j].count {
			return result[i].count > result[j].count
		}
		return strings.ToLower(result[i].label) < strings.ToLower(result[j].label)
	})
	return result
}

func reportMentionMaps(entries []LedgerEntry) (tags, people map[string]reportMention) {
	tags = map[string]reportMention{}
	people = map[string]reportMention{}
	for _, entry := range deduplicateCarryForwardEntries(entries) {
		for _, tag := range entry.Tags {
			addReportMention(tags, tag)
		}
		for _, person := range entry.People {
			addReportMention(people, person)
		}
	}
	return tags, people
}

func reportMentionMapsFromText(lines []string) (tags, people map[string]reportMention) {
	tags = map[string]reportMention{}
	people = map[string]reportMention{}
	for _, line := range lines {
		for _, tag := range extractTags(line) {
			addReportMention(tags, tag)
		}
		for _, person := range extractPeople(line) {
			addReportMention(people, person)
		}
	}
	return tags, people
}

// formatReportMentionSections gives every report generator the same compact,
// readable talking-point material. Tags and people each fit on one line so a
// report can retain the useful counts without growing several screenfuls.
func formatReportMentionSections(tags, people map[string]reportMention, heading string) string {
	if len(tags) == 0 && len(people) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(heading)
	b.WriteString(":\n")
	if len(tags) > 0 {
		b.WriteString("Tags: ")
		b.WriteString(formatCompactReportMentions(tags))
		b.WriteByte('\n')
	}
	if len(people) > 0 {
		b.WriteString("People: ")
		b.WriteString(formatCompactReportMentions(people))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func formatCompactReportMentions(mentions map[string]reportMention) string {
	ranked := rankedReportMentions(mentions)
	values := make([]string, len(ranked))
	for i, mention := range ranked {
		values[i] = fmt.Sprintf("%s(%d)", mention.label, mention.count)
	}
	return strings.Join(values, " ")
}

const reportMentionPromptGuidanceText = " Use the supplied talking-point lists as source material: group related entries under a small number of concrete bullets, and preserve useful tags and people as bold Markdown (for example **#project** and **@person**) when they clarify the point. Do not produce a tag cloud, and do not mention filtering or internal report setup."

func reportMentionPromptGuidance() string { return reportMentionPromptGuidanceText }
