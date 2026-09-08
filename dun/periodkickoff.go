package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// periodKickoffTitle/periodRecurringCadence/periodKickoffDefaultCat
// are the small per-unit differences showPeriodKickoffWindow needs --
// everything else about a Kickoff dialog's shape is identical across
// units with no special-cased dialog of their own (i.e. all but Day/
// Month, which keep their existing bespoke SOD/SOM dialogs).
func periodRecurringCadence(period summaryPeriod) string {
	switch period {
	case periodWeek:
		return "weekly"
	case periodMonth:
		return "monthly"
	default:
		return "" // Quarter/Year have no recurring-item cadence (recurring.go's cadenceOptions is daily/weekly/monthly only)
	}
}

// okrKickoffSection builds the optional OKR-entry module (docs/
// kickoff-review-design.md's OKR design) for period's Kickoff --
// Quarter/Year only, gated by cfg.EnableOKRs. Returns nil if not
// applicable, so callers can skip adding it to their layout entirely.
//
// Shape: a "Theme for this period" freeform field (FOCUS category,
// distinct from scored OKRs, no scoring implied), then Objective
// entry (readback of existing Objectives + Add), and Key-Result entry
// scoped to whichever Objective was most recently added *this
// session* -- this sidesteps the adjacency-matching ledger design's
// one real trap (a KR appended after a later Objective already exists
// in the ledger would misattach to that later one, since
// readObjectives associates KR lines to the nearest *preceding*
// OBJECTIVE line by file order) by simply never offering to attach a
// KR to anything but the newest Objective.
func okrKickoffSection(a fyne.App, period summaryPeriod, anchor time.Time) fyne.CanvasObject {
	if period != periodQuarter && period != periodYear {
		return nil
	}
	cfg := LoadConfig()
	if !cfg.EnableOKRs {
		return nil
	}

	focusEntry := widget.NewEntry()
	focusEntry.SetText(readFocus(period, anchor))
	focusEntry.SetPlaceHolder("Theme for this " + strings.ToLower(string(period)) + " (e.g. \u201cConsolidation quarter\u201d)\u2026")
	saveFocusBtn := widget.NewButton("Save", func() {
		text := strings.TrimSpace(focusEntry.Text)
		if text != "" {
			recordFocus(text, period, anchor)
		}
	})
	focusRow := container.New(newStretchRowLayout(focusEntry), focusEntry, saveFocusBtn)

	objBox := container.NewVBox()
	var latestObjectiveText string
	refreshObjectives := func() {
		objBox.RemoveAll()
		objectives := readObjectives(period, anchor)
		if len(objectives) == 0 {
			objBox.Add(widget.NewLabel("No Objectives set yet for this " + strings.ToLower(string(period)) + "."))
		}
		for _, o := range objectives {
			objBox.Add(widget.NewLabelWithStyle(o.Text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, kr := range o.KeyResults {
				objBox.Add(widget.NewLabel("\u2022 " + kr.Text))
			}
			latestObjectiveText = o.Text
		}
		objBox.Refresh()
	}
	refreshObjectives()

	newObjEntry := widget.NewEntry()
	newObjEntry.SetPlaceHolder("New Objective\u2026")
	krEntry := widget.NewEntry()
	krEntry.SetPlaceHolder("Key Result for the Objective above\u2026")

	addObjBtn := widget.NewButton("Add Objective", func() {
		text := strings.TrimSpace(newObjEntry.Text)
		if text == "" {
			return
		}
		recordObjective(text, period, anchor)
		newObjEntry.SetText("")
		refreshObjectives()
	})
	addKRBtn := widget.NewButton("Add Key Result", func() {
		text := strings.TrimSpace(krEntry.Text)
		if text == "" || latestObjectiveText == "" {
			return
		}
		recordKeyResult(text, period, anchor)
		krEntry.SetText("")
		refreshObjectives()
	})

	newObjRow := container.New(newStretchRowLayout(newObjEntry), newObjEntry, addObjBtn)
	krRow := container.New(newStretchRowLayout(krEntry), krEntry, addKRBtn)

	return container.NewVBox(
		widget.NewLabelWithStyle("OKRs for "+periodLabel(cfg, period, anchor), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Theme:"),
		focusRow,
		objBox,
		newObjRow,
		widget.NewLabel("Key Result attaches to the most recently added Objective above:"),
		krRow,
	)
}

// showPeriodKickoffWindow shows a generic Kickoff dialog (docs/
// kickoff-review-design.md) for period's unit containing anchor --
// open TODOs/GOALs readback, a quick-entry field, and (for Week/
// Month, which have a matching recurring-item cadence) recurring-item
// suggestions. Used for Week, Quarter, and Year, which have no
// bespoke dialog of their own (unlike Day/Month's existing SOD/Month
// Kickoff). Deliberately does not duplicate a smaller unit's own
// recurring-item surfacing when Kickoffs stack on a shared boundary
// day (see design doc's "Kickoff: overlap/stacking" section) -- each
// unit only ever surfaces its own cadence's items. anchor lets
// callers (showPeriodPicker) open a period other than "the current
// one", though Kickoff is normally only meaningful for the current/
// upcoming period -- recurring-item due-ness always checks the real
// current time regardless of anchor, since "what's due" is a
// real-world fact, not something that makes sense to ask about a past
// period.
func showPeriodKickoffWindow(a fyne.App, period summaryPeriod, anchor time.Time) {
	now := time.Now()
	cfg := LoadConfig()
	label := periodLabel(cfg, period, anchor) + periodProgressSuffix(period, anchor)
	w := a.NewWindow("Dunnit: " + string(period) + " Kickoff (" + label + ")")

	listBox := container.NewVBox()
	refreshList := func() {
		listBox.RemoveAll()
		cats, grouped := groupOpenItemsByCategory(getOpenItems())
		if len(cats) == 0 {
			listBox.Add(widget.NewLabel("Nothing open right now \u2014 clean slate!"))
		}
		for _, cat := range cats {
			listBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			for _, item := range grouped[cat] {
				listBox.Add(widget.NewLabel("\u2022 " + stripCarryForwardSince(item.Text) + staleBadge(item.Text)))
			}
		}
		listBox.Refresh()
	}
	refreshList()
	listScroll := container.NewVScroll(listBox)
	listScroll.SetMinSize(fyne.NewSize(0, 220))

	recurringBox := container.NewVBox()
	cadence := periodRecurringCadence(period)
	refreshRecurring := func() {
		recurringBox.RemoveAll()
		if cadence == "" {
			return
		}
		due := dueRecurringItems(cfg, now, cadence)
		if box := recurringItemsSuggestionBox(due, refreshList); box != nil {
			recurringBox.Add(widget.NewLabelWithStyle("Recurring Items Due This "+string(period), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			recurringBox.Add(box)
		}
		recurringBox.Refresh()
	}
	refreshRecurring()

	newItemCat := widget.NewSelect(openTrackedCategories, nil)
	newItemCat.SetSelected("GOAL")
	newItemText := widget.NewEntry()
	newItemText.SetPlaceHolder("Add a goal or open item for this " + strings.ToLower(string(period)) + "\u2026")
	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		refreshList()
		refreshRecurring()
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)
	entryRow := container.New(newStretchRowLayout(newItemText), newItemCat, newItemText, addBtn)

	content := container.NewVBox(
		widget.NewLabel("Kicking off "+label+" \u2014 here\u2019s where things stand:"),
		listScroll,
		recurringBox,
		entryRow,
	)
	if okrBox := okrKickoffSection(a, period, anchor); okrBox != nil {
		content.Add(okrBox)
	}
	content.Add(widget.NewButton("Done", func() { w.Close() }))

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(560, 620))
	w.Show()
}
