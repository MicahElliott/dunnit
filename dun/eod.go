package dun

import (
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

const minAutoEODDoneEntries = 3

func doneEntriesForDate(date time.Time) int {
	path := ledgerFileForDate(date)
	if path == "" {
		return 0
	}
	count := 0
	for _, line := range readLedgerLinesFrom(path) {
		category, _, ok := parseLedgerLine(line)
		if ok && category == "DONE" {
			count++
		}
	}
	return count
}

func autoEODReportEligible(date time.Time) bool {
	return doneEntriesForDate(date) >= minAutoEODDoneEntries
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
		c := widget.NewCheck(stripCarryForwardSince(item.Text)+staleBadge(item.Text), nil)
		c.SetChecked(false)
		checks[i] = c
		box.Add(c)
	}
	return box, items, checks
}

// showEODWindow recreates (in spirit) dun.zsh's dunnit-eod sequence:
// a short daily wrap-up showing everything logged today, an AI-drafted
// summary (editable before saving), a productivity score, a meeting-
// hours count, a sentiment rating, goals for tomorrow, and (FR-09,
// extended) a chance to postpone any TODOs/QUESTIONs not resolved
// today. Rather than a chain of separate popups (as the
// original zsh alerter-based flow did), this is one window with all
// the questions -- simpler to implement and to answer.
func showEODWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: End of Day")

	// Today's items, shown first -- read-only, so the user has the
	// full day in view before answering anything below. Uses a plain
	// widget.Label rather than a disabled MultiLineEntry: disabling
	// an Entry recolors its text via theme.ColorNameDisabled, which
	// (at least with the LightTheme this app forces via
	// a.Settings().SetTheme) renders too close to the background to
	// read -- the box looked entirely blank even though the text was
	// there. Label has no such disabled-state recoloring.
	todayBody := widget.NewLabel(strings.Join(readLedgerLines(), "\n"))
	todayBody.Wrapping = fyne.TextWrapWord
	// Wrapped in a Scroll for the form (so a long day doesn't blow up
	// the whole window), but Scroll doesn't inherit its child's
	// MinSize by default -- without an explicit SetMinSize here, it
	// renders at whatever tiny default the form layout gives it.
	todayScroll := container.NewVScroll(todayBody)
	todayScroll.SetMinSize(fyne.NewSize(0, 220)) // room for ~8+ lines

	// AI-drafted summary: fed today's ledger text via the same
	// summarizeWithLLMCLI pipeline used elsewhere (Summarize/SOM),
	// rather than asking the user to hand-write one. Runs in the
	// background since it shells out to configured LLM CLI; the field starts
	// with a placeholder and is editable once (or before) the draft
	// arrives, so the user can always tweak/replace it before Finalize
	// Day. A rendered-markdown preview (summaryPreview) sits below the
	// raw editable text -- the AI draft often comes back with markdown
	// (headers/bold/lists) that's hard to read as literal "**bold**"
	// text in a plain entry field, so this renders it properly via
	// Fyne's built-in widget.NewRichTextFromMarkdown, updating live as
	// the summary is edited.
	summary := widget.NewMultiLineEntry()
	summary.SetPlaceHolder("Generating an AI summary of today, please wait (feel free to edit once it arrives, or type your own now)\u2026")
	summary.SetMinRowsVisible(10)
	summaryPreview := widget.NewRichTextFromMarkdown("")
	summaryPreview.Wrapping = fyne.TextWrapWord
	summary.OnChanged = func(text string) {
		summaryPreview.ParseMarkdown(text)
	}
	summaryPreviewScroll := container.NewVScroll(summaryPreview)
	summaryPreviewScroll.SetMinSize(fyne.NewSize(0, 160))
	copyMarkdownSummaryBtn := widget.NewButton("Copy as Markdown", func() {
		a.Clipboard().SetContent(summary.Text)
	})
	copyHTMLSummaryBtn := widget.NewButton("Copy as HTML", func() {
		copyRichText(a, summary.Text)
	})
	draftRequest := newLLMCLIRequest()
	w.SetOnClosed(draftRequest.close)
	draftStopBtn := draftRequest.stopButton()
	draftStopBtn.Hide()
	summaryBox := container.NewVBox(
		summary,
		draftStopBtn,
		widget.NewLabelWithStyle("Preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		summaryPreviewScroll,
		container.NewHBox(copyMarkdownSummaryBtn, copyHTMLSummaryBtn),
	)
	go func() {
		today := time.Now()
		if !autoEODReportEligible(today) {
			return
		}
		ledgerText := gatherLedgerTextForDate(today)
		if !hasRealLedgerContent(ledgerText) {
			return
		}
		fyne.Do(draftStopBtn.Show)
		draft, err := summarizeWithLLMCLIPromptContext(draftRequest.ctx,
			"Summarize this ledger of a day's activity entries into "+
				"a brief impact report suitable for a personal end-of-day "+
				"recap. Be concise and group related work together."+
				reviewLengthConstraint(periodDay), ledgerText)
		draftRequest.finish()
		fyne.Do(func() {
			draftStopBtn.Hide()
			if err != nil {
				if draftRequest.canceled() {
					return
				}
				log.Println("Error drafting EOD summary:", err)
				return
			}
			if strings.TrimSpace(summary.Text) == "" {
				summary.SetText(draft)
				summaryPreview.ParseMarkdown(draft)
			}
		})
	}()

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
		widget.NewFormItem("Today's Items", todayScroll),
		widget.NewFormItem("Summary", summaryBox),
		widget.NewFormItem("Productivity (1\u20135)", productivity),
		widget.NewFormItem("Meeting Hours", meetingHours),
		widget.NewFormItem("Sentiment", sentiment),
		widget.NewFormItem("Tomorrow's Goals", goals),
	}
	if len(openTodos) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open TODOs", todoBox))
	}
	if len(openDoing) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open DOING", doingBox))
	}
	if len(openQuestions) > 0 {
		items = append(items, widget.NewFormItem("Postpone Open QUESTIONs", questionBox))
	}
	form := widget.NewForm(items...)
	form.SubmitText = "Finalize Day"
	form.OnSubmit = func() {
		if strings.TrimSpace(summary.Text) != "" {
			_, path := eodReportPath(time.Now())
			if err := writeReportFile(path, summary.Text); err != nil {
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
		// FR-18: draft (if not already present) today's hand-editable
		// EOD report, now that the day's PRODUCTIVITY/
		// SENTIMENT lines above have just been recorded. Gated behind
		// AutoDraftDailySummary (default off) -- open design
		// questions remain about EOD-vs-other trigger timing and how
		// this doc's content should differ from Summarize's existing
		// Day output; see docs/open-design-questions.md. Manual
		// drafting via the "EOD Report..." tray item always
		// works regardless of this setting. Runs in the background
		// since it shells out to configured LLM CLI; opens in $EDITOR when
		// ready rather than blocking Finalize Day.
		if LoadConfig().AutoDraftDailySummary && autoEODReportEligible(time.Now()) {
			request := newLLMCLIRequest()
			progress := a.NewWindow("Dunnit: Drafting EOD Report\u2026")
			progress.SetOnClosed(request.close)
			progress.SetContent(windowPad(llmCLIProgressContent("Drafting the EOD report with the configured LLM CLI, please wait\u2026", request)))
			progress.Resize(fyne.NewSize(420, 140))
			progress.Show()
			go func() {
				path, _, err := ensureEODReportContext(request.ctx, time.Now())
				request.finish()
				fyne.Do(func() {
					progress.Close()
					if request.canceled() {
						return
					}
					if err != nil {
						log.Println("Error drafting EOD report:", err)
						return
					}
					if path != "" {
						openInEditor(path)
					}
				})
			}()
		}
		w.Close()
	}

	w.SetContent(windowPad(container.NewVScroll(form)))
	w.Resize(fyne.NewSize(560, 980))
	w.Show()
}
