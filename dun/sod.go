package dun

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// showSODWindow shows a Start-of-Day readback (FR-13): today's still-
// open items (TODO/GOAL/WAITING/QUESTION/FIXME/RISK, per
// openTrackedCategories) grouped by category, with a quick-entry
// field to log a fresh one before the day gets going. Mirrors
// showEODWindow's one-window-with-everything approach rather than a
// chain of separate popups.
func showSODWindow(a fyne.App) {
	runCarryForwardIfNeeded()
	w := a.NewWindow("Dunnit: Start of Day")

	listBox := container.NewVBox()

	// listAsText renders the currently-shown open items as plain
	// text, for the Copy button -- the list itself isn't selectable
	// (it's built of widget.Labels, not a text widget), so this is
	// the simplest way to let the user get this content into a doc/
	// clipboard elsewhere.
	var listAsText func() string

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

		listAsText = func() string {
			var b strings.Builder
			for i, cat := range cats {
				if i > 0 {
					b.WriteString("\n")
				}
				b.WriteString(categoryPlural(cat) + "\n")
				for _, item := range grouped[cat] {
					b.WriteString("- " + stripCarryForwardSince(item.Text) + "\n")
				}
			}
			return b.String()
		}
	}
	refreshList()

	// Scroll area for the list is given a tall fixed min-height
	// (enough for ~10 lines) so a short list doesn't render as a tiny
	// sliver, and a long list scrolls internally rather than growing
	// the whole window unboundedly.
	listScroll := container.NewVScroll(listBox)
	listScroll.SetMinSize(fyne.NewSize(0, 260))

	// Recurring items (daily/weekly cadence) suggested for today --
	// see RECURRING-ITEMS-DESIGN-SEED.md. Surfaced as suggestions the
	// user explicitly taps "Add" for, not auto-seeded.
	recurringBox := container.NewVBox()
	refreshRecurring := func() {
		recurringBox.RemoveAll()
		due := dueRecurringItems(LoadConfig(), time.Now(), "")
		var dailyWeekly []RecurringItem
		for _, r := range due {
			if r.Cadence != "monthly" {
				dailyWeekly = append(dailyWeekly, r)
			}
		}
		if box := recurringItemsSuggestionBox(dailyWeekly, refreshList); box != nil {
			recurringBox.Add(widget.NewLabelWithStyle("Recurring Items Due Today", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			recurringBox.Add(box)
		}
		recurringBox.Refresh()
	}
	refreshRecurring()

	newItemCat := widget.NewSelect(openTrackedCategories, nil)
	newItemCat.SetSelected("TODO")
	newItemText := newTagAutoEntry()
	newItemText.SetPlaceHolder("Add a new open item for today\u2026")

	// Tag autocomplete (FR-10), same tagAutoEntry pattern as the main
	// Daybook entry (ui.go) and Recurring Meetings' tag field
	// (minicalendar.go) -- see tagautoentry.go's doc comment for why
	// this is an inline suggestion list, not a canvas overlay
	// (PopUpMenu/PopUp were both tried and both broke keyboard input).
	newItemSuggestions := newItemText.SuggestionBox()

	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		refreshList() // rebuild the list in place, rather than closing/reopening the window
		refreshRecurring()
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)

	copyBtn := widget.NewButton("Copy", func() {
		if listAsText != nil {
			a.Clipboard().SetContent(listAsText())
		}
	})

	entryRow := container.New(newStretchRowLayout(newItemText), newItemCat, newItemText, addBtn)

	// Note: new items logged here go straight to the ledger, but
	// Daybook's own "Planned" section only refreshes when Daybook
	// itself is shown/interacted with -- so if Daybook is already
	// open in the background, it won't reflect these until it's
	// closed and reopened (or otherwise refreshed).
	syncNote := widget.NewLabel("Note: new items here will show up in Daybook\u2019s Planned section next time it\u2019s opened.")
	syncNote.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		widget.NewLabel("Good morning! Here\u2019s where things stand:"),
		streakLabel(),
		listScroll,
		copyBtn,
		recurringBox,
		entryRow,
		newItemSuggestions,
		syncNote,
		widget.NewButton("Done", func() { w.Close() }),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
