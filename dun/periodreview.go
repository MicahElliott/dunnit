package dun

import (
	"log"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// okrReviewSection builds the optional OKR-scoring module (docs/
// kickoff-review-design.md's OKR design) for period's Review --
// Quarter/Year only, gated by cfg.EnableOKRs. Read-back of each open
// Key Result plus a status dropdown (defaulted to its current/latest
// status) and an optional short note; nothing is written until
// applyFn is called (wired to the Review window's Done button,
// alongside the existing TODO/QUESTION carry-forward, so OKR scoring
// follows the same "never lose data, decisions applied on Done"
// pattern -- see docs/kickoff-review-design.md's Review section).
// Returns (nil, nil) if not applicable.
func okrReviewSection(period summaryPeriod, anchor time.Time) (box fyne.CanvasObject, applyFn func()) {
	if period != periodQuarter && period != periodYear {
		return nil, nil
	}
	cfg := LoadConfig()
	if !cfg.EnableOKRs {
		return nil, nil
	}
	objectives := readObjectives(period, anchor)
	if len(objectives) == 0 {
		return nil, nil
	}

	type krRow struct {
		text   string
		status *widget.Select
		note   *widget.Entry
	}
	var rows []krRow
	vbox := container.NewVBox(
		widget.NewLabelWithStyle("Score This Period’s Key Results", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)
	for _, o := range objectives {
		if len(o.KeyResults) == 0 {
			continue
		}
		vbox.Add(widget.NewLabelWithStyle(o.Text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		for _, kr := range o.KeyResults {
			statusSelect := widget.NewSelect(okrStatusOptions, nil)
			statusSelect.SetSelected(kr.Status)
			noteEntry := widget.NewEntry()
			noteEntry.SetText(kr.Note)
			noteEntry.SetPlaceHolder("Optional note\u2026")
			rows = append(rows, krRow{text: kr.Text, status: statusSelect, note: noteEntry})
			vbox.Add(widget.NewLabel("\u2022 " + kr.Text))
			vbox.Add(container.New(newStretchRowLayout(noteEntry), statusSelect, noteEntry))
		}
	}

	apply := func() {
		for _, r := range rows {
			// Only append a new KEYRESULT-STATUS line if the status or
			// note actually changed from what's already on record, so
			// re-opening Review without touching anything doesn't spam
			// the ledger with identical status lines.
			recordKeyResultStatus(r.text, r.status.Selected, r.note.Text, period, anchor)
		}
	}
	return vbox, apply
}

// showPeriodReviewWindow shows a generic Review dialog (docs/kickoff-
// review-design.md) for period's unit containing anchor:
// mostly-automatic, generated on-demand (the user picks a theme
// first, then taps Generate) via generateThemedReview (period.go),
// saved via reviewReportPath's theme-aware naming convention so
// multiple differently-themed reports can coexist for the same
// period, and so a larger unit's Review can roll it up. Existing
// saved reports for this exact period are listed up front (view/
// reopen), without blocking a fresh Generate alongside them. Also
// offers TODO/QUESTION carry-forward checkboxes, applied
// unconditionally on Done regardless of whether a report was ever
// generated or saved -- same never-lose-data guarantee as EOD. Used
// for Week, Quarter, and Year, which have no bespoke dialog of their
// own (unlike Day/Month's existing EOD/Month Review); anchor lets
// callers (showPeriodPicker) open a period other than "last one",
// including one still in progress (periodProgressSuffix).
func showPeriodReviewWindow(a fyne.App, period summaryPeriod, anchor time.Time) {
	cfg := LoadConfig()
	label := periodLabel(cfg, period, anchor) + periodProgressSuffix(period, anchor)
	showReportPreparation(a, string(period)+" Review ("+label+")", func() {
		showPeriodReviewWindowReady(a, period, anchor)
	})
}

func showPeriodReviewWindowReady(a fyne.App, period summaryPeriod, anchor time.Time) {
	cfg := LoadConfig()
	label := periodLabel(cfg, period, anchor) + periodProgressSuffix(period, anchor)
	w := a.NewWindow("Dunnit: " + string(period) + " Review (" + label + ")")

	themeSelect := widget.NewSelect(themeOptions(), nil)
	themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, period)])

	statusLabel := newExplanatoryLabel("Pick a theme, then tap Generate.")

	// Existing reports for this exact period, listed up front so
	// Generate is never the only option -- a user can reopen/view a
	// prior draft instead of (or in addition to) regenerating.
	existingBox := container.NewVBox()
	existingPaths, existingThemes := listReviewReportsForPeriod(period, anchor)
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
				showEditableReportWindow(a,
					"Dunnit: "+periodSummaryTitle(period, anchor),
					path, string(body))
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
		go func() {
			overrideCfg := cfg
			setTheme(&overrideCfg, period, selectedTheme)
			summary, err := generateThemedReviewContext(request.ctx, overrideCfg, period, anchor)
			request.finish()
			fyne.Do(func() {
				generateBtn.Enable()
				stopBtn.Hide()
				if request.canceled() {
					statusLabel.SetText("Stopped.")
					return
				}
				if err != nil {
					log.Println("Error generating "+string(period)+" Review:", err)
					statusLabel.SetText("Error generating report \u2014 see logs.")
					dialog.ShowError(err, w)
					return
				}
				statusLabel.SetText("Generated.")
				showEditableReportWindow(a,
					"Dunnit: "+periodSummaryTitle(period, anchor),
					reviewReportPath(period, anchor, selectedTheme), summary)
			})
		}()
	}

	// Postpone-opt-out: same eodOpenItemsSection as EOD -- unresolved
	// TODOs/DOING/QUESTIONs are offered by the next Start of Day, so
	// this section lets the user explicitly send an item to SOMEDAY
	// (Postpone) instead. Applied on Done
	// regardless of whether a report was ever generated -- generating/
	// saving a report and postponing open items are independent
	// actions.
	todoBox, openTodos, todoChecks := eodOpenItemsSection("TODO")
	doingBox, openDoing, doingChecks := eodOpenItemsSection("DOING")
	questionBox, openQuestions, questionChecks := eodOpenItemsSection("QUESTION")
	carryForwardBox := container.NewVBox()
	if len(openTodos) > 0 {
		carryForwardBox.Add(widget.NewLabelWithStyle("Postpone Open TODOs (checked = send to SOMEDAY)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		carryForwardBox.Add(todoBox)
	}
	if len(openDoing) > 0 {
		carryForwardBox.Add(widget.NewLabelWithStyle("Postpone Open DOINGs (checked = send to SOMEDAY)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		carryForwardBox.Add(doingBox)
	}
	if len(openQuestions) > 0 {
		carryForwardBox.Add(widget.NewLabelWithStyle("Postpone Open QUESTIONs (checked = send to SOMEDAY)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		carryForwardBox.Add(questionBox)
	}

	okrBox, applyOKRs := okrReviewSection(period, anchor)

	doneBtn := widget.NewButton("Done", func() {
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
		if applyOKRs != nil {
			applyOKRs()
		}
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabel(string(period)+" Review: "+label),
		container.NewBorder(nil, nil, widget.NewLabel("Theme:"), generateBtn, themeSelect),
		container.NewHBox(statusLabel, stopBtn),
		existingBox,
		carryForwardBox,
	)
	if okrBox != nil {
		content.Add(okrBox)
	}
	content.Add(doneBtn)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(520, 500))
	w.Show()
}
