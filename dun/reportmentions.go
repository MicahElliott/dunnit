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
	for _, entry := range entries {
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
// readable talking-point material. Keeping one tag/person per bullet gives an
// LLM a structure it can group without turning the input into a tag cloud.
func formatReportMentionSections(tags, people map[string]reportMention, heading string) string {
	if len(tags) == 0 && len(people) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(heading)
	b.WriteString(":\n")
	if len(tags) > 0 {
		b.WriteString("### Tags\n")
		for _, mention := range rankedReportMentions(tags) {
			fmt.Fprintf(&b, "- **%s** — %d %s\n", mention.label, mention.count, pluralizeCount(mention.count, "mention", "mentions"))
		}
	}
	if len(people) > 0 {
		b.WriteString("### People\n")
		for _, mention := range rankedReportMentions(people) {
			fmt.Fprintf(&b, "- **%s** — %d %s\n", mention.label, mention.count, pluralizeCount(mention.count, "mention", "mentions"))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

const reportMentionPromptGuidanceText = " Use the supplied talking-point lists as source material: group related entries under a small number of concrete bullets, and preserve useful tags and people as bold Markdown (for example **#project** and **@person**) when they clarify the point. Do not produce a tag cloud, and do not mention filtering or internal report setup."

func reportMentionPromptGuidance() string { return reportMentionPromptGuidanceText }
