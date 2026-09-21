package dun

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// tomorrowLedgerPath returns the ledger directory and filename for
// tomorrow's date (same year/week/month path scheme as getLedger, but
// for time.Now().AddDate(0, 0, 1) instead of today).
func tomorrowLedgerPath() (string, string) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return ledgerPathFor(tomorrow)
}

// endOfDayAlreadyRun reports whether today's EOD was finalized or a report
// was created by another EOD entry point. The marker also covers an explicit
// Skip, which intentionally leaves no report file behind.
func endOfDayAlreadyRun(date time.Time) bool {
	cfg, err := loadConfig()
	if err == nil && cfg.LastEndOfDayDate == date.Format("2006-01-02") {
		return true
	}
	_, reportPath := eodReportPath(date)
	_, statErr := os.Stat(reportPath)
	return statErr == nil
}

// markEndOfDayRun records that the EOD form was completed. As with the SOD
// marker, a malformed config is left untouched rather than replaced with
// defaults just to save this UI state.
func markEndOfDayRun(date time.Time) {
	cfg, err := loadConfig()
	if err != nil {
		log.Println("Skipping End of Day marker:", err)
		return
	}
	dateText := date.Format("2006-01-02")
	if cfg.LastEndOfDayDate == dateText {
		return
	}
	cfg.LastEndOfDayDate = dateText
	if err := writeConfig(cfg); err != nil {
		log.Println("Error saving End of Day marker:", err)
	}
}

// appendTomorrowLine appends a single pre-formatted ledger line (sans
// trailing newline) to tomorrow's ledger file, creating the directory
// and file as needed.
func appendTomorrowLine(line string) error {
	line = normalizeLedgerText(line)
	fpath, fname := tomorrowLedgerPath()
	if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
		return err
	}
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

// recordTomorrowGoals appends each non-blank line as a GOAL entry to
// tomorrow's ledger file (so it's ready to go first thing).
func recordTomorrowGoals(lines []string) {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		appendTomorrowLine("[05:00] GOAL " + line)
	}
}

// eodOpenItemsSection builds one Postpone-opt-out checkbox section
// (used for TODO/DOING and QUESTION) of showEODWindow: a checkbox per
// still-open item of the given category, UNCHECKED by default.
//
// Naming/semantics note (2026-09-02, see
// docs/todo-carryforward-design.md): this section is an explicit
// opt-out from future daily planning. Checking a box Postpones that
// item (recordPostponed, into SOMEDAY); leaving it unchecked leaves
// it available for the next Start of Day carry-forward.
func eodOpenItemsSection(category string) (box *fyne.Container, items []OpenItem, checks []*widget.Check) {
	for _, item := range getOpenItems() {
		if item.Category == category {
			items = append(items, item)
		}
	}
	box = container.NewVBox()
	checks = make([]*widget.Check, len(items))
	for i, item := range items {
		c := widget.NewCheck("", nil)
		c.SetChecked(false)
		checks[i] = c
		box.Add(container.NewHBox(c,
			itemTextLabel(categoryIconPrefix(item.Category)+openItemDisplayText(item.Text))))
	}
	return box, items, checks
}

// eodLedgerLineLabel gives EOD's Today’s Items the same category, tag, link,
// duration, and carry-forward rendering used by Daybook's item rows.
func eodLedgerLineLabel(line string) fyne.CanvasObject {
	category, text, ok := parseLedgerLine(line)
	if !ok {
		return itemTextLabel(line)
	}
	return itemTextLabel(categoryIconPrefix(category) +
		openItemDisplayText(stripResolutionSuffix(text)))
}

func eodEntryIncluded(entry LedgerEntry, cfg Config) bool {
	for _, tag := range entry.Tags {
		for _, excluded := range cfg.ReportExcludeTags {
			if strings.EqualFold(tag, excluded) {
				return false
			}
		}
	}
	return true
}

func eodReportFacts(date time.Time) string {
	cfg := LoadConfig()
	people := make(map[string]bool)
	topics := make(map[string]bool)
	entries := 0
	for _, entry := range AllLedgerEntries() {
		if !sameCalendarDate(entry.Date, date) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		entries++
		for _, person := range entry.People {
			people[strings.ToLower(person)] = true
		}
		for _, tag := range entry.Tags {
			topics[strings.ToLower(tag)] = true
		}
	}
	if entries == 0 {
		return ""
	}
	return fmt.Sprintf("Worked with %d %s across %d %s.", len(people),
		pluralizeCount(len(people), "person", "people"), len(topics),
		pluralizeCount(len(topics), "topic", "topics"))
}

func pluralizeCount(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func appendEODReportFacts(report string, date time.Time) string {
	facts := eodReportFacts(date)
	if facts == "" {
		return report
	}
	trimmed := strings.TrimSpace(report)
	if strings.Contains("\n"+trimmed+"\n", "\n"+facts+"\n") {
		return trimmed + "\n"
	}
	if trimmed == "" {
		return facts + "\n"
	}
	return trimmed + "\n\n" + facts + "\n"
}

func appendEODLedgerDetails(report string, date time.Time) string {
	cfg := LoadConfig()
	var completed, learned []string
	for _, entry := range AllLedgerEntries() {
		if !sameCalendarDate(entry.Date, date) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		text := strings.TrimSpace(stripResolutionSuffix(entry.Text))
		if text == "" {
			continue
		}
		switch entry.Category {
		case "DONE":
			completed = append(completed, text)
		case "TIL":
			learned = append(learned, text)
		}
	}
	if len(completed) == 0 && len(learned) == 0 {
		return report
	}
	trimmed := strings.TrimSpace(report)
	var sections []string
	if len(completed) > 0 && !strings.Contains(trimmed, "## Completed items (from ledger)") {
		var b strings.Builder
		b.WriteString("## Completed items (from ledger)\n")
		for _, text := range completed {
			b.WriteString("- ")
			b.WriteString(text)
			b.WriteByte('\n')
		}
		sections = append(sections, strings.TrimSpace(b.String()))
	}
	if len(learned) > 0 && !strings.Contains(trimmed, "## Learnings (from ledger)") {
		var b strings.Builder
		b.WriteString("## Learnings (from ledger)\n")
		for _, text := range learned {
			b.WriteString("- ")
			b.WriteString(text)
			b.WriteByte('\n')
		}
		sections = append(sections, strings.TrimSpace(b.String()))
	}
	if len(sections) == 0 {
		return trimmed + "\n"
	}
	if trimmed == "" {
		return strings.Join(sections, "\n\n") + "\n"
	}
	return trimmed + "\n\n" + strings.Join(sections, "\n\n") + "\n"
}

func augmentEODReport(report string, date time.Time) string {
	return appendEODReportFacts(appendEODLedgerDetails(report, date), date)
}

// eodReportStats summarizes the ledger metadata that belongs immediately
// below an EOD report's title. Keeping it here makes the report date and its
// stats come from the same ledger day even when an older report is opened.
func eodReportStats(date time.Time) string {
	entries := 0
	done := 0
	meetingHours := ""
	allEntries := AllLedgerEntries()
	cfg := LoadConfig()
	for _, entry := range allEntries {
		if !sameCalendarDate(entry.Date, date) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		entries++
		switch entry.Category {
		case "DONE":
			done++
		case "MEETING_HOURS":
			meetingHours = strings.TrimSpace(entry.Text)
		}
	}
	parts := []string{fmt.Sprintf("%d entries", entries)}
	if done > 0 {
		parts = append(parts, fmt.Sprintf("%d done", done))
	}
	if meetingHours != "" {
		parts = append(parts, meetingHours+" meeting hours")
	}
	if reflection := dayReflection(allEntries, date); reflection != "" {
		parts = append(parts, reflection)
	}
	return strings.Join(parts, " · ")
}

// eodReportForDisplay gives every opened report a date-correct title and
// moves its compact stats line directly below that title. Existing report
// bodies may contain an old or generated H1, so it is removed from the body
// before the canonical title is added.
func eodReportForDisplay(report string, date time.Time) string {
	lines := strings.Split(strings.TrimSpace(report), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		lines = lines[1:]
	}
	var bodyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Stats:") || strings.HasPrefix(trimmed, "*Stats:") {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	body = strings.TrimSpace(augmentEODReport(body, date))
	heading := "# End-of-Day Recap — " + date.Format("Mon Jan 2")
	stats := "*Stats: " + eodReportStats(date) + "*"
	if body == "" {
		return heading + "\n\n" + stats + "\n"
	}
	return heading + "\n\n" + stats + "\n\n" + body + "\n"
}

func showEODReport(a fyne.App, date time.Time, path, report string) {
	showGeneratedReport(a, "Dunnit: EOD Report — "+date.Format("Mon Jan 2"), path,
		eodReportForDisplay(report, date))
}

func showEODAlreadyRunWindow(a fyne.App, date time.Time) {
	w := a.NewWindow("Dunnit: End of Day")
	message := "End of Day has already been handled for " +
		date.Format("Monday, January 2") + "."
	_, reportPath := eodReportPath(date)
	if _, err := os.Stat(reportPath); err == nil {
		message += " The existing EOD report was left unchanged."
	}
	w.SetContent(windowPad(container.NewVBox(
		widget.NewLabelWithStyle(message, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("There is nothing more to do today."),
		widget.NewButton("Close", w.Close),
	)))
	w.Resize(fyne.NewSize(460, 180))
	w.Show()
}

// showEODWindow recreates (in spirit) dun.zsh's dunnit-eod sequence:
// a short daily wrap-up showing everything logged today, an optional
// AI-drafted summary (editable before saving), a productivity score, a meeting-
// hours count, a sentiment rating, goals for tomorrow, and (FR-09,
// extended) a chance to postpone any TODOs/QUESTIONs not resolved
// today. Rather than a chain of separate popups (as the
// original zsh alerter-based flow did), this is one window with all
// the questions -- simpler to implement and to answer.
func showEODWindow(a fyne.App) {
	now := time.Now()
	if endOfDayAlreadyRun(now) {
		showEODAlreadyRunWindow(a, now)
		return
	}
	w := a.NewWindow("Dunnit: End of Day")

	// Today's items, shown first -- read-only, so the user has the
	// full day in view before answering anything below. Render each
	// line separately so inline entry links remain clickable.
	todayBody := container.NewVBox()
	for _, line := range readLedgerLines() {
		todayBody.Add(eodLedgerLineLabel(line))
	}
	if len(todayBody.Objects) == 0 {
		todayBody.Add(widget.NewLabel("Nothing logged yet today."))
	}
	// Wrapped in a Scroll for the form (so a long day doesn't blow up
	// the whole window), but Scroll doesn't inherit its child's
	// MinSize by default -- without an explicit SetMinSize here, it
	// renders at whatever tiny default the form layout gives it.
	todayScroll := container.NewVScroll(todayBody)
	todayScroll.SetMinSize(fyne.NewSize(0, 220)) // room for ~8+ lines

	// The optional AI-drafted summary uses the same summarizeWithLLMCLI
	// pipeline used elsewhere (Summarize/SOM), but only starts after the
	// user taps Generate. It runs in the background since it shells out to
	// the configured LLM CLI; the field is editable while the draft arrives,
	// so the user can always tweak/replace it before Finalize
	// Day. A rendered-markdown preview (summaryPreview) sits below the
	// raw editable text -- the AI draft often comes back with markdown
	// (headers/bold/lists) that's hard to read as literal "**bold**"
	// text in a plain entry field, so this renders it properly via
	// Fyne's built-in widget.NewRichTextFromMarkdown, updating live as
	// the summary is edited.
	summary := widget.NewMultiLineEntry()
	summary.SetPlaceHolder("Tap Generate to create the EOD report summary…")
	summary.Disable()
	summary.SetMinRowsVisible(10)
	summaryPreview := newReportRichText("")
	summaryPreview.Wrapping = fyne.TextWrapWord
	summary.OnChanged = func(text string) {
		setReportRichTextMarkdown(summaryPreview, text)
	}
	summaryPreviewScroll := container.NewVScroll(summaryPreview)
	summaryPreviewScroll.SetMinSize(fyne.NewSize(0, 160))
	copyMarkdownSummaryBtn := widget.NewButton("Copy as Markdown", func() {
		a.Clipboard().SetContent(summary.Text)
	})
	copyRichTextSummaryBtn := widget.NewButton("Copy as rich text", func() {
		copyRichText(a, summary.Text)
	})
	var draftRequest *llmCLIRequest
	generationRequested := false
	generating := false
	var finalizeDay func(bool)
	draftStopBtn := widget.NewButton("Stop generating", func() {
		if draftRequest != nil {
			draftRequest.cancel()
		}
	})
	draftStopBtn.Hide()
	skipBtn := widget.NewButton("Skip", func() { finalizeDay(false) })
	skipBtn.Hide()
	generateBtn := widget.NewButton("Generate", nil)
	generateBtn.OnTapped = func() {
		if generating {
			return
		}
		generationRequested = true
		generating = true
		summary.Enable()
		generateBtn.Disable()
		skipBtn.Show()
		draftStopBtn.Show()
		draftRequest = newLLMCLIRequest()
		request := draftRequest
		go func() {
			ledgerText := gatherLedgerTextForDate(now)
			hasContent := hasRealLedgerContent(ledgerText)
			var draft string
			var err error
			if hasContent {
				draft, err = summarizeWithLLMCLIPromptContext(request.ctx,
					eodSummaryPrompt(), ledgerText)
			}
			request.finish()
			fyne.Do(func() {
				generating = false
				draftStopBtn.Hide()
				generateBtn.Enable()
				if !hasContent {
					summary.SetPlaceHolder("Nothing is logged today to summarize. You can type a report here.")
					return
				}
				if err != nil {
					if request.canceled() {
						return
					}
					log.Println("Error drafting EOD summary:", err)
					summary.SetPlaceHolder("No summary was generated. You can type one here and finalize.")
					return
				}
				if strings.TrimSpace(summary.Text) == "" {
					summary.SetText(draft)
					setReportRichTextMarkdown(summaryPreview, draft)
				}
			})
		}()
	}
	w.SetOnClosed(func() {
		if draftRequest != nil {
			draftRequest.close()
		}
	})
	summaryBox := container.NewVBox(
		summary,
		container.NewHBox(generateBtn, draftStopBtn),
		widget.NewLabelWithStyle("Preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		summaryPreviewScroll,
		container.NewHBox(copyMarkdownSummaryBtn, copyRichTextSummaryBtn),
	)

	productivity := widget.NewSelect([]string{"1", "2", "3", "4", "5"}, nil)
	productivity.SetSelected("3")

	meetingHours := widget.NewEntry()
	meetingHours.SetPlaceHolder("e.g. 2.5")

	sentiment := widget.NewSelect([]string{"Negative", "Neutral", "Positive"}, nil)
	sentiment.SetSelected("Neutral")

	goals := widget.NewMultiLineEntry()
	goals.SetPlaceHolder("Any goals for tomorrow? One per line\u2026")
	goals.SetMinRowsVisible(3)

	// Open TODOs, DOING, and QUESTIONs each get their own Postpone checkbox
	// section. Checking a box sends that item to SOMEDAY before the next
	// Start of Day planning pass.
	todoBox, openTodos, todoChecks := eodOpenItemsSection("TODO")
	doingBox, openDoing, doingChecks := eodOpenItemsSection("DOING")
	questionBox, openQuestions, questionChecks := eodOpenItemsSection("QUESTION")

	items := []*widget.FormItem{
		widget.NewFormItem("Today’s Items", todayScroll),
		widget.NewFormItem("Summary", summaryBox),
		widget.NewFormItem("Productivity (1\u20135)", productivity),
		widget.NewFormItem("Meeting Hours", meetingHours),
		widget.NewFormItem("Sentiment", sentiment),
		widget.NewFormItem("Tomorrow’s Goals", goals),
	}
	if len(openTodos) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open TODOs", todoBox))
	}
	if len(openDoing) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open DOINGs", doingBox))
	}
	if len(openQuestions) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open QUESTIONs", questionBox))
	}
	form := widget.NewForm(items...)
	finalized := false
	finalizeDay = func(writeEODReport bool) {
		if finalized {
			return
		}
		finalized = true
		if draftRequest != nil && generating {
			draftRequest.cancel()
		}
		if generationRequested && writeEODReport && strings.TrimSpace(summary.Text) != "" {
			_, path := eodReportPath(now)
			if err := writeReportFileIfAbsent(path, augmentEODReport(summary.Text, now)); err != nil {
				log.Println("Error saving EOD report:", err)
			}
		}
		recordActivity(productivity.Selected, "PRODUCTIVITY")
		if hrs := strings.TrimSpace(meetingHours.Text); hrs != "" {
			recordActivity(hrs, "MEETING_HOURS")
		}
		recordActivity(sentiment.Selected, "SENTIMENT")

		if strings.TrimSpace(goals.Text) != "" {
			recordTomorrowGoals(strings.Split(goals.Text, "\n"))
		}
		for i, item := range openTodos {
			if todoChecks[i].Checked {
				recordPostponed(item)
			}
		}
		for i, item := range openDoing {
			if doingChecks[i].Checked {
				recordPostponed(item)
			}
		}
		for i, item := range openQuestions {
			if questionChecks[i].Checked {
				recordPostponed(item)
			}
		}
		markEndOfDayRun(now)
		w.Close()
	}
	finalizeBtn := widget.NewButton("Finalize Day", func() { finalizeDay(true) })

	w.SetContent(windowPad(container.NewBorder(nil,
		container.NewHBox(finalizeBtn, skipBtn), nil, nil,
		container.NewVScroll(form))))
	w.Resize(fyne.NewSize(560, 980))
	w.Show()
}
