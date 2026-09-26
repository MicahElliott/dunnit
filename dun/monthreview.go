package dun

import (
	"log"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ideaSomedayCategories are the categories reviewed in Month Review's
// possibility-triage step. Other open categories remain visible in the
// preparation/Daybook flow and are not repeated here.
var ideaSomedayCategories = map[string]bool{"IDEA": true, "SOMEDAY": true}

type ideaSomedayItem struct {
	Category string
	Text     string
}

// monthReviewEntries returns one report-safe, deduplicated view of the
// month's ledger. Carry-forward copies are one logical item, and configured
// exclusion tags are applied before any Month Review section sees the data.
func monthReviewEntries(from, to time.Time) []LedgerEntry {
	cfg := LoadConfig()
	var entries []LedgerEntry
	for _, entry := range AllLedgerEntries() {
		if entry.Date.Before(from) || entry.Date.After(to) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		entries = append(entries, entry)
	}
	return deduplicateCarryForwardEntries(entries)
}

// gatherIdeaSomedayItems scans the month's effective ledger state rather than
// rendering every repeated carry-forward line. The category icon is added at
// render time from the shared Categories registry.
func gatherIdeaSomedayItems(from, to time.Time) []ideaSomedayItem {
	var out []ideaSomedayItem
	for _, entry := range monthReviewEntries(from, to) {
		if ideaSomedayCategories[entry.Category] {
			out = append(out, ideaSomedayItem{Category: entry.Category, Text: entry.Text})
		}
	}
	return out
}

func monthReviewHiliteCategories() []Category {
	var out []Category
	for _, category := range Categories {
		if category.Group == "hilite" && !category.EODOnly {
			out = append(out, category)
		}
	}
	return out
}

func monthReviewHiliteOptions() []string {
	categories := monthReviewHiliteCategories()
	options := make([]string, len(categories))
	for i, category := range categories {
		options[i] = category.Label()
	}
	return options
}

func selectedMonthReviewHilites(selected []string) map[string]bool {
	wanted := make(map[string]bool, len(selected))
	for _, label := range selected {
		for _, category := range monthReviewHiliteCategories() {
			if category.Label() == label {
				wanted[category.Code] = true
				break
			}
		}
	}
	return wanted
}

func monthReviewHiliteEvidence(entries []LedgerEntry, selected map[string]bool) *fyne.Container {
	box := container.NewVBox()
	if len(selected) == 0 {
		box.Add(newExplanatoryLabel("Select at least one Hilite type to see its entries here."))
		return box
	}
	found := 0
	for _, entry := range entries {
		if !selected[entry.Category] {
			continue
		}
		found++
		box.Add(container.NewHBox(
			widget.NewLabel(entry.Date.Format("Jan 2")),
			itemTextLabel(categoryIconPrefix(entry.Category)+entry.Text),
		))
	}
	if found == 0 {
		box.Add(newExplanatoryLabel("No selected Hilites were logged in this month."))
	}
	return box
}

// showMonthReviewWindow shows the guided backward-looking Month Review after
// the shared quick tidy-up handoff. The report window is created only after
// the user explicitly finishes or skips that preparation step.
func showMonthReviewWindow(a fyne.App, anchor time.Time) {
	cfg := LoadConfig()
	label := periodLabel(cfg, periodMonth, anchor) + periodProgressSuffix(periodMonth, anchor)
	showReportPreparation(a, "Month Review ("+label+")", func() {
		showMonthReviewWindowReady(a, anchor)
	})
}

func showMonthReviewWindowReady(a fyne.App, anchor time.Time) {
	from, to := periodNominalRange(periodMonth, anchor)
	cfg := LoadConfig()
	label := periodLabel(cfg, periodMonth, from) + periodProgressSuffix(periodMonth, from)
	w := a.NewWindow("Dunnit: Month Review — " + label)

	entries := monthReviewEntries(from, to)
	hiliteOptions := monthReviewHiliteOptions()
	var refreshHiliteEvidence func()
	hiliteSelect := widget.NewCheckGroup(hiliteOptions, func([]string) {
		if refreshHiliteEvidence != nil {
			refreshHiliteEvidence()
		}
	})
	// Preserve the broad existing behavior; the user can narrow the focus.
	hiliteSelect.SetSelected(append([]string{}, hiliteOptions...))
	hiliteEvidence := container.NewVBox()
	refreshHiliteEvidence = func() {
		hiliteEvidence.RemoveAll()
		for _, object := range monthReviewHiliteEvidence(entries, selectedMonthReviewHilites(hiliteSelect.Selected)).Objects {
			hiliteEvidence.Add(object)
		}
		hiliteEvidence.Refresh()
	}
	refreshHiliteEvidence()
	selectAllHilitesBtn := widget.NewButton("Select all", func() {
		hiliteSelect.SetSelected(append([]string{}, hiliteOptions...))
		refreshHiliteEvidence()
	})
	clearHilitesBtn := widget.NewButton("Clear", func() {
		hiliteSelect.SetSelected(nil)
		refreshHiliteEvidence()
	})

	// The optional capture row lets the user add a missing Hilite while this
	// review is open. It uses the same emoji-backed category registry as the
	// selector and writes through the normal append-only Daybook path.
	hiliteCaptureSelect := widget.NewSelect(hiliteOptions, nil)
	if len(hiliteOptions) > 0 {
		hiliteCaptureSelect.SetSelected(hiliteOptions[0])
	}
	hiliteCapture := widget.NewMultiLineEntry()
	hiliteCapture.SetPlaceHolder("Add one or more missing Hilites, one per line…")
	hiliteCapture.SetMinRowsVisible(2)
	addHiliteBtn := widget.NewButton("Add Hilite", func() {
		category := categoryCodeFromLabel(hiliteCaptureSelect.Selected)
		if category == "" {
			return
		}
		for _, line := range strings.Split(hiliteCapture.Text, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				if err := recordActivity(line, category); err != nil {
					dialog.ShowError(err, w)
					return
				}
			}
		}
		hiliteCapture.SetText("")
		entries = monthReviewEntries(from, to)
		refreshHiliteEvidence()
	})

	// Generate a themed digest only after the focus and source controls are
	// visible. The selected Hilites are passed into the prompt as an explicit
	// focus section as well as controlling the evidence shown here.
	digestBody := newReportRichText("*Choose Hilites and a report style, then generate.*")
	digestBody.Wrapping = fyne.TextWrapWord
	themeSelect := widget.NewSelect(themeOptions(), nil)
	themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, periodMonth)])
	statusLabel := newExplanatoryLabel("")
	generateBtn := widget.NewButton("Generate report", nil)
	var request *llmCLIRequest
	stopBtn := widget.NewButton("Stop generating", nil)
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
		selectedHilites := selectedMonthReviewHilites(hiliteSelect.Selected)
		generateBtn.Disable()
		request = newLLMCLIRequest()
		stopBtn.OnTapped = func() {
			stopBtn.Disable()
			request.cancel()
		}
		stopBtn.Enable()
		stopBtn.Show()
		statusLabel.SetText("Generating, please wait…")
		setReportRichTextMarkdown(digestBody, "*Generating, please wait…*")
		go func() {
			overrideCfg := LoadConfig()
			setTheme(&overrideCfg, periodMonth, selectedTheme)
			summary, err := generateThemedReviewContextWithHilites(request.ctx, overrideCfg, periodMonth, from, selectedHilites)
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
					statusLabel.SetText("Error generating report — see logs.")
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

	existingBox := container.NewVBox()
	existingPaths, existingThemes := listReviewReportsForPeriod(periodMonth, from)
	if len(existingPaths) > 0 {
		existingBox.Add(newWindowHeading("Saved reports for " + label))
		for i, path := range existingPaths {
			path, th := path, existingThemes[i]
			display := themeDisplayNames[th]
			if display == "" {
				display = "Saved report"
			}
			existingBox.Add(widget.NewButton("Open "+display, func() {
				body, err := os.ReadFile(path)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				showEditableReportWindow(a, "Dunnit: Month Review Report ("+label+")", path, string(body))
			}))
		}
	}

	items := gatherIdeaSomedayItems(from, to)
	triageBox := container.NewVBox()
	if len(items) == 0 {
		triageBox.Add(newExplanatoryLabel("No IDEA or SOMEDAY items were found in this month after deduplication and exclusions."))
	}
	for _, item := range items {
		item := item
		row := widget.NewLabel(categoryIconPrefix(item.Category) + stripCarryForwardSince(item.Text))
		row.Wrapping = fyne.TextWrapWord
		var promoteTodoBtn, promoteGoalBtn, dropBtn *hoverButton
		promoteTodoBtn = newHoverIconButton(theme.Icon(theme.IconNameConfirm), "Make TODO", func() {
			if err := recordActivity(stripCarryForwardSince(item.Text), "TODO"); err != nil {
				dialog.ShowError(err, w)
				return
			}
			promoteTodoBtn.Disable()
			promoteGoalBtn.Disable()
			dropBtn.Disable()
			row.SetText("✅ Made TODO: " + stripCarryForwardSince(item.Text))
		})
		promoteGoalBtn = newHoverIconButton(theme.Icon(theme.IconNameDocumentCreate), "Make GOAL", func() {
			if err := recordActivity(stripCarryForwardSince(item.Text), "GOAL"); err != nil {
				dialog.ShowError(err, w)
				return
			}
			promoteTodoBtn.Disable()
			promoteGoalBtn.Disable()
			dropBtn.Disable()
			row.SetText("🎯 Made GOAL: " + stripCarryForwardSince(item.Text))
		})
		dropBtn = newHoverIconButton(theme.Icon(theme.IconNameDelete), "Discard", func() {
			if err := recordActivity(stripCarryForwardSince(item.Text), "DISCARDED"); err != nil {
				dialog.ShowError(err, w)
				return
			}
			promoteTodoBtn.Disable()
			promoteGoalBtn.Disable()
			dropBtn.Disable()
			row.SetText("🚫 Discarded: " + stripCarryForwardSince(item.Text))
		})
		triageBox.Add(container.NewBorder(nil, nil, nil,
			container.NewHBox(promoteTodoBtn, promoteGoalBtn, dropBtn), row))
	}

	doneBtn := widget.NewButton("Finish review", func() { w.Close() })
	excluded := strings.Join(cfg.ReportExcludeTags, ", ")
	if excluded == "" {
		excluded = "none"
	}

	content := container.NewVBox(
		newWindowHeading("🗓️ Month Review — "+label),
		newExplanatoryLabel("Review the month’s evidence, focus on the Hilites that matter, and then create an editable report."),
		newWindowHeading("Choose reflection evidence"),
		newExplanatoryLabel("Selected Hilite types shape both the evidence shown here and the generated report. Excluded tags: "+excluded+"."),
		hiliteSelect,
		container.NewHBox(selectAllHilitesBtn, clearHilitesBtn),
		hiliteEvidence,
		container.NewHBox(hiliteCaptureSelect, addHiliteBtn),
		hiliteCapture,
		newWindowHeading("Generate report"),
		newExplanatoryLabel("Choose a style, then generate. The result opens in an editable report window."),
		container.NewHBox(widget.NewLabel("Style:"), themeSelect, generateBtn),
		container.NewHBox(statusLabel, stopBtn),
		existingBox,
		digestBody,
		newWindowHeading("Triage ideas and someday items"),
		newExplanatoryLabel("These two categories are possibilities rather than active daily work. Use Daybook during tidy-up for TODOs, DOING, and other open work."),
		triageBox,
		doneBtn,
	)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(620, 760))
	w.Show()
}
