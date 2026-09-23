package dun

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// periodSummaryTitle is the canonical Markdown title for a generated
// period summary. It is kept separate from periodLabel because reports
// should say "Week Summary" and show the full calendar-week range even
// when the UI is configured to display a five-day work week.
func periodSummaryTitle(period summaryPeriod, anchor time.Time) string {
	switch period {
	case periodWeek:
		start, _ := periodNominalRange(periodWeek, anchor)
		_, week := start.ISOWeek()
		end := start.AddDate(0, 0, 6)
		return fmt.Sprintf("Week Summary (W%d — %s)", week,
			shortDateRange(start, end))
	case periodMonth:
		return "Month Summary (" + anchor.Format("Jan 2006") + ")"
	case periodQuarter:
		q := quarterOf(anchor)
		months := [4]string{"Jan–Mar", "Apr–Jun", "Jul–Sep", "Oct–Dec"}
		return fmt.Sprintf("Quarter Summary (Q%d %d — %s)", q,
			anchor.Year(), months[q-1])
	case periodYear:
		return fmt.Sprintf("Year Summary (%d)", anchor.Year())
	case periodDay:
		return "Day Summary (" + anchor.Format("Mon Jan 2, 2006") + ")"
	default:
		return string(period) + " Summary"
	}
}

func shortDateRange(start, end time.Time) string {
	if start.Month() == end.Month() {
		return start.Format("Jan 2") + "-" + end.Format("2")
	}
	return start.Format("Jan 2") + "-" + end.Format("Jan 2")
}

// currentPeriodRange returns the nominal period, clipped at now when the
// selected period is still in progress. The report title continues to use
// the complete nominal range so a partial report is still clearly tied to
// its calendar unit.
func currentPeriodRange(period summaryPeriod, anchor, now time.Time) (from, to time.Time) {
	from, to = periodNominalRange(period, anchor)
	if to.After(now) {
		to = now
	}
	return from, to
}

// summaryReportPath returns the save path for the older standalone Summary
// command, using the same period directories and date-token convention as
// themed Review reports while keeping Summary files distinguishable.
func summaryReportPath(period summaryPeriod, anchor time.Time) string {
	reviewPath := reviewReportPath(period, anchor, "")
	filename := reportFilename("summary-"+strings.ToLower(string(period)),
		reviewReportDateToken(period, anchor), "", time.Now())
	return filepath.Join(filepath.Dir(reviewPath), filename)
}

func periodSummaryPrompt(period summaryPeriod, title string) string {
	return fmt.Sprintf(
		"Create a detailed Markdown %s report titled %q. The first line must be exactly %q; do not add another title or an 'Impact report' heading. Use these sections: "+
			"## Summary, ## Accomplishments, ## Hilites and Callouts, ## Still To Do, ## Risks and Blockers (omit if there are none), ## Learnings, and ## Conclusion. "+
			"The Summary should assess sentiment, productivity, pace, meeting load, and overall progress when the input supports it. Preserve concrete details from the ledger, especially every useful hilite and pending item. The Conclusion must be one or two sentences of prose, not bullets, that state the overall result and the most useful next focus. Do not invent facts. Use a little more detail than a standup update, with concise bullets inside sections. "+
			reportUnitName(period), title, "# "+title) + reportMentionPromptGuidance()
}

func reportUnitName(period summaryPeriod) string {
	return strings.ToLower(string(period))
}

// periodReportInput adds explicit, structured signals to raw ledger input.
// Raw lines remain available for detail, while these sections keep the LLM
// from overlooking metrics, hilites, or unresolved work.
func periodReportInput(period summaryPeriod, anchor time.Time, ledgerText string) string {
	from, to := currentPeriodRange(period, anchor, time.Now())
	signals := periodReportSignals(from, to)
	if signals == "" {
		return ledgerText
	}
	return ledgerText + "\n\nStructured report context:\n\n" + signals
}

func periodReportSignals(from, to time.Time) string {
	entries := FilterLedgerEntries(LedgerQuery{From: from, To: to})
	cfg := LoadConfig()
	var included []LedgerEntry
	for _, entry := range entries {
		if eodEntryIncluded(entry, cfg) {
			included = append(included, entry)
		}
	}
	included = deduplicateCarryForwardEntries(included)
	if len(included) == 0 {
		return ""
	}

	completed := 0
	totalMinutes := 0
	productivityTotal := 0
	productivityCount := 0
	sentiments := make(map[string]int)
	var hilites []string
	for _, entry := range included {
		switch entry.Category {
		case "DONE":
			completed++
		case "PRODUCTIVITY":
			if value, err := strconv.Atoi(strings.TrimSpace(entry.Text)); err == nil {
				productivityTotal += value
				productivityCount++
			}
		case "SENTIMENT":
			value := strings.ToLower(strings.TrimSpace(entry.Text))
			if value != "" {
				sentiments[value]++
			}
		}
		totalMinutes += entry.Mins
		if categoryGroup(entry.Category) == "hilite" && entry.Text != "" {
			hilites = append(hilites, entry.Category+": "+entry.Text)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "- Entries: %d\n- Completed: %d\n", len(included), completed)
	if totalMinutes > 0 {
		fmt.Fprintf(&b, "- Tracked time: %s\n", formatReportDuration(totalMinutes))
	}
	if productivityCount > 0 {
		fmt.Fprintf(&b, "- Average productivity: %.1f/5\n",
			float64(productivityTotal)/float64(productivityCount))
	}
	if len(sentiments) > 0 {
		b.WriteString("- Sentiment entries: ")
		writeCounts(&b, sentiments)
		b.WriteByte('\n')
	}
	tags, people := reportMentionMaps(included)
	fmt.Fprintf(&b, "- People mentioned: %d\n- Topics mentioned: %d\n",
		len(people), len(tags))
	if mentions := formatReportMentionSections(tags, people, "Talking points from the ledger"); mentions != "" {
		b.WriteString("\n")
		b.WriteString(mentions)
		b.WriteByte('\n')
	}

	if len(hilites) > 0 {
		b.WriteString("\nHilites and callouts from the period:\n")
		for _, hilite := range hilites {
			b.WriteString("- ")
			b.WriteString(hilite)
			b.WriteByte('\n')
		}
	}

	openItems := reportOpenItemsThrough(to, cfg)
	if len(openItems) > 0 {
		b.WriteString("\nPending plan items as of the end of the period:\n")
		for _, item := range openItems {
			fmt.Fprintf(&b, "- %s: %s\n", item.Category, item.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

func reportEntriesInRange(from, to time.Time, categories map[string]bool) []LedgerEntry {
	all := FilterLedgerEntries(LedgerQuery{From: from, To: to})
	cfg := LoadConfig()
	var included []LedgerEntry
	for _, entry := range all {
		if len(categories) > 0 && !categories[entry.Category] {
			continue
		}
		if eodEntryIncluded(entry, cfg) {
			included = append(included, entry)
		}
	}
	sort.SliceStable(included, func(i, j int) bool {
		return included[i].Date.Before(included[j].Date)
	})
	return included
}

func reportMentionContextForRange(from, to time.Time, categories map[string]bool) string {
	tags, people := reportMentionMaps(reportEntriesInRange(from, to, categories))
	return formatReportMentionSections(tags, people, "Talking points from the ledger")
}

func categoryGroup(code string) string {
	for _, category := range Categories {
		if category.Code == code {
			return category.Group
		}
	}
	return ""
}

func formatReportDuration(minutes int) string {
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh %dm", minutes/60, minutes%60)
}

func writeCounts(b *strings.Builder, counts map[string]int) {
	first := true
	for _, label := range []string{"positive", "neutral", "negative"} {
		if count := counts[label]; count > 0 {
			if !first {
				b.WriteString(", ")
			}
			fmt.Fprintf(b, "%s %d", label, count)
			first = false
		}
	}
	for label, count := range counts {
		if label == "positive" || label == "neutral" || label == "negative" {
			continue
		}
		if !first {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%s %d", label, count)
		first = false
	}
}

func reportOpenItemsThrough(to time.Time, cfg Config) []OpenItem {
	var entries []LedgerEntry
	for _, entry := range AllLedgerEntries() {
		if entry.Date.After(to) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		entries = append(entries, entry)
	}
	active, order := openItemsThrough(entries, to)
	items := make([]OpenItem, 0, len(order))
	for _, key := range order {
		if state, ok := active[key]; ok {
			items = append(items, state.item)
		}
	}
	return items
}

// normalizePeriodReport keeps the period-report call sites descriptive while
// sharing the canonical title behavior with the other generated reports.
func normalizePeriodReport(text, title string) string {
	return normalizeReport(text, title)
}
