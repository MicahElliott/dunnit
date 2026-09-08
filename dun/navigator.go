package dun

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// navigatorDateRangeOptions are the fixed choices for Navigator's
// date-range dropdown -- "All time" (no bound) plus the current/last
// N of each periodConfigs unit, reusing periodOffsetAnchor/
// periodNominalRange (period.go) rather than inventing a separate
// range-addressing scheme. Deliberately a small fixed list (not a
// full date-picker) to match Navigator's other filters' low-effort,
// dropdown-driven style -- a free date-range picker can be added
// later if this proves too coarse in practice.
var navigatorDateRangeOptions = []string{
	"All time",
	"Today",
	"This Week", "Last Week",
	"This Month", "Last Month",
	"This Quarter", "Last Quarter",
	"This Year", "Last Year",
}

// navigatorDateRange resolves one of navigatorDateRangeOptions into a
// concrete [from, to] bound (zero values for "All time", meaning
// unbounded on that side -- matches LedgerQuery's own "zero means
// unconstrained" convention).
func navigatorDateRange(option string) (from, to time.Time) {
	now := time.Now()
	switch option {
	case "Today":
		return periodNominalRange(periodDay, now)
	case "This Week":
		return periodNominalRange(periodWeek, now)
	case "Last Week":
		return periodNominalRange(periodWeek, periodOffsetAnchor(periodWeek, now, -1))
	case "This Month":
		return periodNominalRange(periodMonth, now)
	case "Last Month":
		return periodNominalRange(periodMonth, periodOffsetAnchor(periodMonth, now, -1))
	case "This Quarter":
		return periodNominalRange(periodQuarter, now)
	case "Last Quarter":
		return periodNominalRange(periodQuarter, periodOffsetAnchor(periodQuarter, now, -1))
	case "This Year":
		return periodNominalRange(periodYear, now)
	case "Last Year":
		return periodNominalRange(periodYear, periodOffsetAnchor(periodYear, now, -1))
	default: // "All time" or unrecognized
		return time.Time{}, time.Time{}
	}
}

// parseNavigatorTagsInput splits a freeform, space/comma-separated
// tags field (same input convention as Settings' Report Exclude Tags
// field) into a clean []string of "#tag" tokens -- tolerant of the
// user typing with or without a leading "#", and of either comma or
// whitespace as separators.
func parseNavigatorTagsInput(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	var tags []string
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if !strings.HasPrefix(f, "#") {
			f = "#" + f
		}
		tags = append(tags, f)
	}
	return tags
}

// showNavigatorWindow opens the "Navigator" -- a browse/search window
// over all ledger history. Combines three composable filters
// (category, tags, date range) built on the shared LedgerQuery/
// FilterLedgerEntries layer (ledgerquery.go), plus "Ask AI about
// these" (feeds the currently-filtered entries into
// summarizeWithCopilotPrompt with a free-form question) and
// "Histogram..." (a plain-text per-category bar chart of the current
// filtered set, formatCategoryHistogram) -- see docs/navigator-design.md
// for the fuller design discussion and remaining open items
// (report-corpus search now lives in reportslibrary.go).
//
// EODOnly categories are included in the category dropdown (unlike
// showHelp's picker-oriented legend) -- Navigator is about browsing
// what's actually in the ledger, and EODOnly categories (SUMMARY/
// PRODUCTIVITY/MEETING_HOURS) do appear there even though they're
// never hand-picked.
func showNavigatorWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: Navigator")

	catOptions := make([]string, 0, len(Categories)+1)
	catOptions = append(catOptions, "All")
	for _, c := range Categories {
		catOptions = append(catOptions, c.Code)
	}
	catSelect := widget.NewSelect(catOptions, nil)
	catSelect.SetSelected("All")

	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("#tag1, #tag2 (blank = any)")

	rangeSelect := widget.NewSelect(navigatorDateRangeOptions, nil)
	rangeSelect.SetSelected("All time")

	results := widget.NewMultiLineEntry()
	results.Wrapping = fyne.TextWrapOff
	results.SetMinRowsVisible(18)

	countLabel := widget.NewLabel("")

	// currentEntries holds the last-computed filtered set, so "Ask AI
	// about these" can reuse it without re-filtering.
	var currentEntries []LedgerEntry

	buildQuery := func() LedgerQuery {
		q := LedgerQuery{}
		if catSelect.Selected != "" && catSelect.Selected != "All" {
			q.Categories = []string{catSelect.Selected}
		}
		q.Tags = parseNavigatorTagsInput(tagsEntry.Text)
		q.From, q.To = navigatorDateRange(rangeSelect.Selected)
		return q
	}

	refresh := func() {
		currentEntries = FilterLedgerEntries(buildQuery())
		if len(currentEntries) == 0 {
			countLabel.SetText("No entries found.")
			results.SetText("")
			return
		}
		countLabel.SetText(pluralCount(len(currentEntries), "entry", "entries"))
		var sb strings.Builder
		for _, e := range currentEntries {
			sb.WriteString(e.Date.Format("2006-01-02") + " " +
				e.Time.Format("15:04:05") + "  " +
				e.Category + "  " + e.Text + "\n")
		}
		results.SetText(sb.String())
	}
	catSelect.OnChanged = func(string) { refresh() }
	rangeSelect.OnChanged = func(string) { refresh() }
	tagsEntry.OnChanged = func(string) { refresh() }
	refresh()

	askAIBtn := widget.NewButton("Ask AI about these...", func() {
		showNavigatorAskAIDialog(a, w, currentEntries)
	})
	histogramBtn := widget.NewButton("Histogram...", func() {
		showNavigatorHistogramWindow(a, currentEntries)
	})

	filterRow := container.NewBorder(nil, nil,
		widget.NewLabel("Category:"), nil,
		container.NewHBox(catSelect,
			widget.NewLabel("Tags:"), tagsEntry,
			widget.NewLabel("Range:"), rangeSelect,
		))

	content := container.NewVBox(
		filterRow,
		container.NewBorder(nil, nil, nil, container.NewHBox(histogramBtn, askAIBtn), countLabel),
		results,
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(720, 560))
	w.Show()
}

// showNavigatorAskAIDialog prompts for a free-form question, then
// feeds it plus entries' text (via ledgerEntriesToText) into
// summarizeWithCopilotPrompt as a one-shot Q&A -- reusing the same
// gh-copilot integration point every other AI-report feature
// (Standup/Status Report/Annual Review/Kickoff-Review) already funnels
// through, just with a free-form question instead of a fixed
// instruction template. Runs synchronously in a goroutine with a
// simple "working..." placeholder, matching this codebase's existing
// pattern for long-running copilot calls (see summarize.go/
// dailysummary.go).
func showNavigatorAskAIDialog(a fyne.App, parent fyne.Window, entries []LedgerEntry) {
	if len(entries) == 0 {
		dialog.ShowInformation("Nothing to Ask About",
			"No entries match the current filters.", parent)
		return
	}

	question := widget.NewEntry()
	question.SetPlaceHolder("e.g. What did I accomplish on this topic?")

	dialog.ShowCustomConfirm("Ask AI", "Ask", "Cancel", question, func(ok bool) {
		if !ok || strings.TrimSpace(question.Text) == "" {
			return
		}
		q := question.Text
		go func() {
			instructions := fmt.Sprintf(
				"Answer the following question using only the ledger "+
					"entries provided below as source material -- be concise, "+
					"and if the entries don't contain enough information to "+
					"answer, say so rather than guessing. Question: %q", q)
			answer, err := summarizeWithCopilotPrompt(instructions, ledgerEntriesToText(entries))
			if err != nil {
				dialog.ShowError(err, parent)
				return
			}
			showNavigatorAIAnswerWindow(a, q, answer)
		}()
	}, parent)
}

// ledgerEntriesToText renders entries back into ledger-line-shaped
// text ("[HH:MM:SS] CATEGORY text", one per line, prefixed with the
// entry's date since entries here may span many days unlike a single
// day's raw ledger file) for feeding to summarizeWithCopilotPrompt.
func ledgerEntriesToText(entries []LedgerEntry) string {
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(e.Date.Format("2006-01-02") + " [" + e.Time.Format("15:04:05") + "] " +
			e.Category + " " + e.Text + "\n")
	}
	return sb.String()
}

// showNavigatorAIAnswerWindow displays an Ask-AI answer in its own
// standalone window -- separate from Navigator's own window so the
// filtered browse view stays open/usable while reading the answer,
// and so multiple questions can be asked without each answer
// replacing the last.
func showNavigatorAIAnswerWindow(a fyne.App, question, answer string) {
	w := a.NewWindow("Dunnit: Navigator — Ask AI")
	body := widget.NewMultiLineEntry()
	body.Wrapping = fyne.TextWrapWord
	body.SetText(answer)
	content := container.NewBorder(
		widget.NewLabel("Q: "+question), nil, nil, nil,
		body,
	)
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(560, 420))
	w.Show()
}

// pluralCount renders n with the singular or plural form of noun,
// e.g. pluralCount(1, "entry", "entries") -> "1 entry",
// pluralCount(5, "entry", "entries") -> "5 entries".
func pluralCount(n int, singular, plural string) string {
	word := plural
	if n == 1 {
		word = singular
	}
	return strconv.Itoa(n) + " " + word
}

// categoryCounts returns, for the given entries, a map of Category ->
// count -- a small building block for a possible future "category
// histogram" navigator mode (not yet surfaced in the UI).
func categoryCounts(entries []LedgerEntry) map[string]int {
	counts := make(map[string]int)
	for _, e := range entries {
		counts[e.Category]++
	}
	return counts
}

// sortedCategoryCounts returns categoryCounts' keys sorted by
// descending count (ties broken alphabetically), for stable display
// order in any future histogram/summary view.
func sortedCategoryCounts(counts map[string]int) []string {
	cats := make([]string, 0, len(counts))
	for c := range counts {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool {
		if counts[cats[i]] != counts[cats[j]] {
			return counts[cats[i]] > counts[cats[j]]
		}
		return cats[i] < cats[j]
	})
	return cats
}

// histogramBarWidth is the max number of "*" characters used to
// render the largest bar in formatCategoryHistogram -- every other
// bar is scaled proportionally, so the chart fits a fixed-width
// window regardless of how large the top count is.
const histogramBarWidth = 40

// formatCategoryHistogram renders entries' per-category counts
// (categoryCounts/sortedCategoryCounts) as a plain-text ASCII bar
// chart, one line per category, widest bar first -- matching
// trend.go's existing house style (no charting library, plain text)
// rather than introducing a new dependency for this.
func formatCategoryHistogram(entries []LedgerEntry) string {
	if len(entries) == 0 {
		return "(no entries to chart)"
	}
	counts := categoryCounts(entries)
	cats := sortedCategoryCounts(counts)

	maxCatLen := 0
	maxCount := 0
	for _, c := range cats {
		if len(c) > maxCatLen {
			maxCatLen = len(c)
		}
		if counts[c] > maxCount {
			maxCount = counts[c]
		}
	}

	var sb strings.Builder
	for _, c := range cats {
		n := counts[c]
		barLen := histogramBarWidth
		if maxCount > 0 {
			barLen = n * histogramBarWidth / maxCount
		}
		if barLen == 0 && n > 0 {
			barLen = 1 // always show at least one char for a nonzero count
		}
		label := c + strings.Repeat(" ", maxCatLen-len(c))
		sb.WriteString(label + "  " + strings.Repeat("*", barLen) + " " + strconv.Itoa(n) + "\n")
	}
	return sb.String()
}

// showNavigatorHistogramWindow displays entries' per-category
// breakdown as a plain-text ASCII bar chart in its own window --
// separate from Navigator's own window so it can stay open
// side-by-side with the filtered browse view, same rationale as
// showNavigatorAIAnswerWindow.
func showNavigatorHistogramWindow(a fyne.App, entries []LedgerEntry) {
	w := a.NewWindow("Dunnit: Navigator — Histogram")
	body := widget.NewMultiLineEntry()
	body.Wrapping = fyne.TextWrapOff
	body.SetText(formatCategoryHistogram(entries))
	w.SetContent(windowPad(container.NewBorder(
		widget.NewLabel(pluralCount(len(entries), "entry", "entries")+" by category:"),
		nil, nil, nil,
		body,
	)))
	w.Resize(fyne.NewSize(480, 420))
	w.Show()
}
