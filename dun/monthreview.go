package dun

import (
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ideaSomedayCategories are the categories reviewed in Month Review's
// IDEA/SOMEDAY triage step.
var ideaSomedayCategories = map[string]bool{"IDEA": true, "SOMEDAY": true}

// ideaSomedayItem is one IDEA/SOMEDAY line found in the month being
// reviewed, pending a promote/drop decision.
type ideaSomedayItem struct {
	Category string
	Text     string
}

// gatherIdeaSomedayItems scans ledger files dated within [from, to]
// for IDEA/SOMEDAY lines, in first-seen order. No resolved/dropped
// tracking beyond this run -- triage is a one-time pass per
// invocation (append-only design: promoting/dropping never touches
// the original lines).
func gatherIdeaSomedayItems(from, to time.Time) []ideaSomedayItem {
	var out []ideaSomedayItem
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil || date.Before(from) || date.After(to) {
			continue
		}
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !ideaSomedayCategories[cat] {
				continue
			}
			out = append(out, ideaSomedayItem{Category: cat, Text: text})
		}
	}
	return out
}

// showMonthReviewWindow shows Month's Review dialog for the month
// containing anchor -- entirely backward-looking (docs/kickoff-
// review-design.md's Kickoff/Review split, replacing the old
// showSOMWindow which conflated this with Month's Kickoff), though
// "backward" also covers a still-in-progress month if anchor is the
// current month (periodProgressSuffix frames that case explicitly
// rather than implying the month already ended):
//  1. AI-generated digest, behind an explicit Generate button (no
//     eager call) -- same pattern as showPeriodReviewWindow, saved
//     via reviewReportPath's theme-aware naming so multiple themed
//     reports can coexist for the same month; existing saved reports
//     for this exact month are listed up front (view/reopen).
//  2. Triage the month's open IDEA/SOMEDAY lines: promote each to
//     TODO or GOAL, or drop (append-only).
//  3. Explicit IMPACT/MILESTONE prompts for the month (backward-
//     looking reflections, not forward planning -- moved here from
//     the old SOM step 3).
//
// Forward-looking content (GOALs for the new month, monthly recurring
// items) lives in showMonthKickoffWindow instead.
func showMonthReviewWindow(a fyne.App, anchor time.Time) {
	from, to := periodNominalRange(periodMonth, anchor)
	cfg := LoadConfig()
	label := periodLabel(cfg, periodMonth, from) + periodProgressSuffix(periodMonth, from)
	w := a.NewWindow("Dunnit: Month Review \u2014 Looking Back at " + label)

	// Step 1: digest, behind a Generate button.
	digestBody := newReportRichText("*Pick a theme, then tap Generate.*")
	digestBody.Wrapping = fyne.TextWrapWord
	themeSelect := widget.NewSelect(themeOptions(), nil)
	themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, periodMonth)])
	statusLabel := widget.NewLabel("")

	existingBox := container.NewVBox()
	existingPaths, existingThemes := listReviewReportsForPeriod(periodMonth, from)
	if len(existingPaths) > 0 {
		existingBox.Add(widget.NewLabelWithStyle("Already Saved for "+label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for i, path := range existingPaths {
			path, th := path, existingThemes[i]
			display := themeDisplayNames[th]
			if display == "" {
				display = "(untitled)"
			}
			existingBox.Add(widget.NewButton("View: "+display, func() {
				body, err := os.ReadFile(path)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				showEditableReportWindow(a, "Dunnit: Month Review Report ("+label+")", path, string(body))
			}))
		}
	}

	generateBtn := widget.NewButton("Generate", nil)
	var request *llmCLIRequest
	stopBtn := widget.NewButton("Stop", nil)
	stopBtn.Hide()
	w.SetOnClosed(func() {
		if request != nil {
			request.close()
		}
	})
	generateBtn.OnTapped = func() {
		selectedTheme := themeFromDisplayName(themeSelect.Selected)
		if selectedTheme == "" {
			return
		}
		generateBtn.Disable()
		request = newLLMCLIRequest()
		stopBtn.OnTapped = func() {
			stopBtn.Disable()
			request.cancel()
		}
		stopBtn.Enable()
		stopBtn.Show()
		statusLabel.SetText("Generating, please wait\u2026")
		setReportRichTextMarkdown(digestBody, "*Generating, please wait\u2026*")
		go func() {
			overrideCfg := cfg
			setTheme(&overrideCfg, periodMonth, selectedTheme)
			summary, err := generateThemedReviewContext(request.ctx, overrideCfg, periodMonth, from)
			request.finish()
			fyne.Do(func() {
				generateBtn.Enable()
				stopBtn.Hide()
				if request.canceled() {
					statusLabel.SetText("Stopped.")
					return
				}
				if err != nil {
					log.Println("Error generating Month Review:", err)
					statusLabel.SetText("Error generating report \u2014 see logs.")
					dialog.ShowError(err, w)
					return
				}
				statusLabel.SetText("Generated.")
				setReportRichTextMarkdown(digestBody, summary)
				showEditableReportWindow(a,
					"Dunnit: Month Review Report ("+label+")",
					reviewReportPath(periodMonth, from, selectedTheme), summary)
			})
		}()
	}

	// Step 2: IDEA/SOMEDAY triage
	items := gatherIdeaSomedayItems(from, to)
	triageBox := container.NewVBox()
	if len(items) == 0 {
		triageBox.Add(widget.NewLabel("No open IDEA/SOMEDAY items from this month."))
	}
	handled := make([]bool, len(items))
	for i, item := range items {
		i, item := i, item // capture
		row := widget.NewLabel(item.Category + ": " + item.Text)
		promoteTodoBtn := widget.NewButton("→ TODO", func() {
			recordActivity(item.Text, "TODO")
			handled[i] = true
			row.SetText("Promoted to TODO — " + item.Text)
		})
		promoteGoalBtn := widget.NewButton("→ GOAL", func() {
			recordActivity(item.Text, "GOAL")
			handled[i] = true
			row.SetText("Promoted to GOAL — " + item.Text)
		})
		dropBtn := widget.NewButton("Drop", func() {
			handled[i] = true
			row.SetText("Dropped — " + item.Text)
		})
		triageBox.Add(container.NewBorder(nil, nil, nil,
			container.NewHBox(promoteTodoBtn, promoteGoalBtn, dropBtn), row))
	}

	// Step 3: IMPACT/MILESTONE prompts (backward-looking -- reflecting
	// on the month just ending, not planning ahead).
	impactEntry := widget.NewMultiLineEntry()
	impactEntry.SetPlaceHolder("Any IMPACT items this month? One per line\u2026")
	impactEntry.SetMinRowsVisible(2)
	milestoneEntry := widget.NewMultiLineEntry()
	milestoneEntry.SetPlaceHolder("Any MILESTONE items this month? One per line\u2026")
	milestoneEntry.SetMinRowsVisible(2)

	doneBtn := widget.NewButton("Done", func() {
		for _, line := range strings.Split(impactEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "IMPACT")
			}
		}
		for _, line := range strings.Split(milestoneEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "MILESTONE")
			}
		}
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Looking Back: "+label+" Digest", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, widget.NewLabel("Theme:"), generateBtn, themeSelect),
		container.NewHBox(statusLabel, stopBtn),
		existingBox,
		digestBody,
		widget.NewLabelWithStyle("Looking Back: Review IDEA/SOMEDAY Items", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		triageBox,
		widget.NewLabelWithStyle("Looking Back: IMPACT / MILESTONE This Month", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		impactEntry,
		milestoneEntry,
		doneBtn,
	)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(520, 600))
	w.Show()
}
