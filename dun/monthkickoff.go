package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showMonthKickoffWindow shows Month's Kickoff dialog for the month
// containing anchor -- entirely forward-looking (docs/kickoff-review-
// design.md's Kickoff/Review split, replacing the old showSOMWindow
// which conflated this with Month's Review):
//  1. Current GOALs readback + new/updated GOALs entry for the new
//     month.
//  2. Monthly recurring-item suggestions (always computed against the
//     real current time regardless of anchor, since "what's due" is a
//     real-world fact -- Kickoff isn't normally opened for a past
//     month, but anchor is still accepted for symmetry with Review).
//
// No digest, no "what happened last month" section -- that's Month
// Review's job (showMonthReviewWindow). Any IDEA/SOMEDAY items
// promoted to GOAL during Month Review are picked up here for free
// via the live getOpenItems() readback below, so look-back naturally
// feeds forward without any direct hand-off code between the two
// windows.
func showMonthKickoffWindow(a fyne.App, anchor time.Time) {
	now := time.Now()
	cfg := LoadConfig()
	label := periodLabel(cfg, periodMonth, anchor)
	w := a.NewWindow("Dunnit: Month Kickoff \u2014 Planning " + label)

	// Current GOALs readback + entry for the new month.
	var currentGoals []OpenItem
	for _, item := range getOpenItems() {
		if item.Category == "GOAL" {
			currentGoals = append(currentGoals, item)
		}
	}
	currentGoalsBox := container.NewVBox()
	if len(currentGoals) == 0 {
		currentGoalsBox.Add(widget.NewLabel("(no current GOALs logged yet)"))
	}
	for _, item := range currentGoals {
		currentGoalsBox.Add(widget.NewLabel("\u2022 " + item.Text))
	}
	newGoalsEntry := widget.NewMultiLineEntry()
	newGoalsEntry.SetPlaceHolder("New/updated GOALs for this month? One per line\u2026")
	newGoalsEntry.SetMinRowsVisible(2)

	// Monthly recurring items, surfaced as a checklist (see
	// RECURRING-ITEMS-DESIGN-SEED.md) -- each due monthly item is a
	// suggestion the user explicitly taps "Add" for, not auto-seeded.
	recurringBox := container.NewVBox()
	dueMonthly := dueRecurringItems(cfg, now, "monthly")
	if box := recurringItemsSuggestionBox(dueMonthly, nil); box != nil {
		recurringBox.Add(box)
	} else {
		recurringBox.Add(widget.NewLabel("(no monthly recurring items due)"))
	}

	doneBtn := widget.NewButton("Commence "+now.Month().String(), func() {
		for _, line := range strings.Split(newGoalsEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				recordActivity(line, "GOAL")
			}
		}
		w.Close()
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Looking Ahead: "+label+" GOALs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		currentGoalsBox,
		newGoalsEntry,
		widget.NewLabelWithStyle("Looking Ahead: Monthly Recurring Items", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		recurringBox,
		doneBtn,
	)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(520, 480))
	w.Show()
}
