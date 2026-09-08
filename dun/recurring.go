package dun

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RecurringItem is one user-entered recurring TODO/GOAL (recurring-
// items feature, see RECURRING-ITEMS-DESIGN-SEED.md): a small, hand-
// maintained list of things to be reminded of on a repeating cadence,
// distinct from RecurringMeeting (FR-15, which is meeting-specific
// and carries time-of-day precision this feature doesn't need).
//
// Deliberately simple/no time-of-day: Cadence is "daily", "weekly",
// or "monthly". DOW (0=Sunday..6=Saturday, only meaningful for
// "weekly") mirrors RecurringMeeting.DOW. DayOfMonth (1-31, only
// meaningful for "monthly") is clamped to the last day of shorter
// months so e.g. 31 still fires in February. WeekendPolicy (only
// meaningful for "daily") is "include" or "skip" -- empty string
// (e.g. entries saved before this field existed) is treated as
// "include", preserving old behavior.
//
// Per design decision: these are surfaced as *suggestions* (SOD for
// daily/weekly, SOM for monthly) rather than auto-seeded into the
// ledger -- the user explicitly taps "Add" to log one for today,
// avoiding duplicate-looking ledger noise if they already logged the
// same thing by hand.
type RecurringItem struct {
	Category      string `toml:"category"`
	Text          string `toml:"text"`
	Cadence       string `toml:"cadence"` // "daily", "weekly", "monthly"
	DOW           int    `toml:"dow"`
	DayOfMonth    int    `toml:"day_of_month"`
	WeekendPolicy string `toml:"weekend_policy"` // "include" (default) or "skip" -- daily only
}

var cadenceOptions = []string{"daily", "weekly", "monthly"}

// weekendPolicyOptions are the choices for a "daily" RecurringItem's
// weekend handling, shown as a second dropdown next to Cadence (a
// dropdown rather than a toggle/checkbox for visual consistency with
// weekly's day-of-week and monthly's day-of-month selectors, even
// though a toggle would be a more natural fit for a binary choice).
var weekendPolicyOptions = []string{"Include weekends", "Skip weekends"}

// isDueToday reports whether r's cadence puts it due on now's date.
func (r RecurringItem) isDueToday(now time.Time) bool {
	switch r.Cadence {
	case "daily":
		if r.WeekendPolicy == "skip" && (now.Weekday() == time.Saturday || now.Weekday() == time.Sunday) {
			return false
		}
		return true
	case "weekly":
		return int(now.Weekday()) == r.DOW
	case "monthly":
		return now.Day() == clampDayOfMonth(r.DayOfMonth, now)
	default:
		return false
	}
}

// clampDayOfMonth clamps day (1-31) to the last actual day of now's
// month, so e.g. day_of_month=31 still fires in a 28/29/30-day month.
func clampDayOfMonth(day int, now time.Time) int {
	if day < 1 {
		day = 1
	}
	lastOfMonth := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Day()
	if day > lastOfMonth {
		return lastOfMonth
	}
	return day
}

// alreadyLoggedToday reports whether an identical (category, text)
// item already exists among today's open items -- used to dedup
// suggestions so a recurring item already logged by hand today
// doesn't also show up as a suggestion.
func alreadyLoggedToday(r RecurringItem) bool {
	for _, item := range getOpenItems() {
		if item.Category == r.Category && item.Text == r.Text {
			return true
		}
	}
	return false
}

// dueRecurringItems returns the configured recurring items matching
// cadence that are due on now's date and not already logged today.
// cadence == "" matches any cadence.
func dueRecurringItems(cfg Config, now time.Time, cadence string) []RecurringItem {
	var out []RecurringItem
	for _, r := range cfg.RecurringItems {
		if cadence != "" && r.Cadence != cadence {
			continue
		}
		if !r.isDueToday(now) {
			continue
		}
		if alreadyLoggedToday(r) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// recurringItemsSuggestionBox builds a small VBox of "Add" rows for
// the given due items, appending a logged entry (via recordActivity)
// and disabling the row once added. Shared by SOD (daily/weekly) and
// SOM (monthly) so both surface suggestions the same way. Returns nil
// (no box added) if items is empty.
func recurringItemsSuggestionBox(items []RecurringItem, onAdded func()) fyne.CanvasObject {
	if len(items) == 0 {
		return nil
	}
	box := container.NewVBox()
	for _, r := range items {
		r := r // capture
		label := widget.NewLabel(r.Category + ": " + r.Text)
		var addBtn *widget.Button
		addBtn = widget.NewButton("Add", func() {
			log.Println("recurringItemsSuggestionBox Add clicked:", r.Category, r.Text)
			recordActivity(r.Text, r.Category)
			label.SetText("[added] " + r.Category + ": " + r.Text)
			addBtn.Disable()
			if onAdded != nil {
				onAdded()
			}
		})
		box.Add(container.NewBorder(nil, nil, nil, addBtn, label))
	}
	return box
}

// showRecurringItemsDialog lets the user add/edit/delete recurring
// TODO/GOAL/KUDOS entries, persisted in config.toml's recurring_item
// array-of-tables (see Config.RecurringItems). Modeled on
// showMiniCalendarDialog (minicalendar.go) but simpler: no time-of-
// day, just category/text/cadence/day. Each existing item gets its
// own inline delete (trash emoji) rather than a single "Delete
// Selected" button below the list.
func showRecurringItemsDialog(a fyne.App, parent fyne.Window) {
	cfg := LoadConfig()
	items := append([]RecurringItem(nil), cfg.RecurringItems...)

	itemsBox := container.NewVBox()
	var refreshItems func()

	saveAll := func() {
		newCfg := cfg
		newCfg.RecurringItems = items
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, parent)
		}
	}

	refreshItems = func() {
		itemsBox.RemoveAll()
		if len(items) == 0 {
			itemsBox.Add(widget.NewLabel("No recurring items yet."))
		}
		for i, r := range items {
			i := i // capture
			detail := r.Cadence
			switch r.Cadence {
			case "daily":
				if r.WeekendPolicy == "skip" {
					detail = "daily (weekdays only)"
				}
			case "weekly":
				detail = "weekly (" + dowNames[r.DOW] + ")"
			case "monthly":
				detail = "monthly (day " + strconv.Itoa(r.DayOfMonth) + ")"
			}
			row := container.NewBorder(nil, nil, nil,
				newHoverIconButton(theme.Icon(theme.IconNameDelete), "Delete", func() {
					items = append(items[:i], items[i+1:]...)
					saveAll()
					refreshItems()
				}),
				widget.NewLabel(r.Category+": "+r.Text+" \u2014 "+detail))
			itemsBox.Add(row)
		}
		itemsBox.Refresh()
	}
	refreshItems()

	// recurringItemCategories deliberately restricts this feature's
	// category choices to TODO/GOAL/KUDOS -- the categories that
	// actually make sense repeated on a schedule (unlike e.g. WAITING/
	// QUESTION/FIXME/RISK, which are reactive/situational, not
	// something you'd pre-schedule).
	recurringItemCategories := []string{"TODO", "GOAL", "KUDOS"}
	catSelect := widget.NewSelect(recurringItemCategories, nil)
	catSelect.SetSelected("TODO")

	textEntry := widget.NewEntry()
	textEntry.SetPlaceHolder("Item text\u2026")

	cadenceSelect := widget.NewSelect(cadenceOptions, nil)
	cadenceSelect.SetSelected("daily")

	// weekendSelect (only relevant/shown for "daily") mirrors weekly's
	// day-of-week and monthly's day-of-month selectors visually --
	// a dropdown for consistency, even though a checkbox/toggle would
	// be the more natural widget for a plain binary choice.
	weekendSelect := widget.NewSelect(weekendPolicyOptions, nil)
	weekendSelect.SetSelected(weekendPolicyOptions[0])

	dowSelect := widget.NewSelect(dowNames, nil)
	dowSelect.SetSelected(dowNames[time.Monday])
	dowSelect.Hide()

	domEntry := widget.NewEntry()
	domEntry.SetPlaceHolder("1\u201331")
	domWrapper := container.NewGridWrap(fyne.NewSize(50, domEntry.MinSize().Height), domEntry)
	domWrapper.Hide()

	cadenceSelect.OnChanged = func(c string) {
		weekendSelect.Hide()
		dowSelect.Hide()
		domWrapper.Hide()
		switch c {
		case "daily":
			weekendSelect.Show()
		case "weekly":
			dowSelect.Show()
		case "monthly":
			domWrapper.Show()
		}
	}

	var addItem func()
	addBtn := widget.NewButton("Add", func() { addItem() })
	addItem = func() {
		text := strings.TrimSpace(textEntry.Text)
		if text == "" {
			dialog.ShowError(errors.New("text is required"), parent)
			return
		}
		r := RecurringItem{
			Category: catSelect.Selected,
			Text:     text,
			Cadence:  cadenceSelect.Selected,
		}
		switch r.Cadence {
		case "daily":
			if weekendSelect.Selected == weekendPolicyOptions[1] {
				r.WeekendPolicy = "skip"
			}
		case "weekly":
			for i, n := range dowNames {
				if n == dowSelect.Selected {
					r.DOW = i
				}
			}
		case "monthly":
			day, err := strconv.Atoi(strings.TrimSpace(domEntry.Text))
			if err != nil || day < 1 || day > 31 {
				dialog.ShowError(errors.New("day of month must be 1\u201331"), parent)
				return
			}
			r.DayOfMonth = day
		}
		items = append(items, r)
		saveAll()
		refreshItems()
		textEntry.SetText("")
		domEntry.SetText("")
	}
	// Enter in the text field submits, same as the main Daybook entry
	// (ui.go) and SOD's quick-add field.
	textEntry.OnSubmitted = func(string) { addItem() }
	domEntry.OnSubmitted = func(string) { addItem() }

	helpLine := widget.NewLabelWithStyle("📝 These entries will be suggested for you to add in Start of Day / Start of Month, at the selected frequency.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	helpLine.Wrapping = fyne.TextWrapWord

	// entryRow stretches textEntry to fill remaining width (same
	// stretchRowLayout approach as ui.go's doneWrapper), rather than
	// letting Fyne's default layout render it at an oddly narrow
	// width.
	entryRow := container.New(newStretchRowLayout(textEntry), catSelect, textEntry, addBtn)

	content := container.NewVBox(
		helpLine,
		entryRow,
		container.NewHBox(cadenceSelect, weekendSelect, dowSelect, domWrapper),
		widget.NewSeparator(),
		itemsBox,
	)

	w := a.NewWindow("Recurring Items")
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(480, 420))
	w.Show()
}
