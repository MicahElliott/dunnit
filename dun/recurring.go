package dun

import (
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// RecurringItem is one user-entered recurring TODO/GOAL (recurring-
// items feature, see RECURRING-ITEMS-DESIGN-SEED.md): a small, hand-
// maintained list of things to be reminded of on a repeating cadence,
// distinct from RecurringMeeting (FR-15). Time is optional: when set,
// the scheduler sends a reminder and opens Daybook at that time; when
// empty, the item remains a Start-of-Day/Start-of-Month suggestion.
//
// Cadence is "daily", "weekly", or "monthly". DOW (0=Sunday..6=Saturday, only meaningful for
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
	Time          string `toml:"time"`           // optional "HH:MM" reminder time
}

var cadenceOptions = []string{"daily", "weekly", "monthly"}

// weekendPolicyOptions are the choices for a "daily" RecurringItem's
// weekend handling, shown as a second dropdown next to Cadence (a
// dropdown rather than a toggle/checkbox for visual consistency with
// weekly's day-of-week and monthly's day-of-month selectors, even
// though a toggle would be a more natural fit for a binary choice).
var weekendPolicyOptions = []string{"Include weekends", "Skip weekends"}

func weekendPolicyFor(policy string) string {
	if policy == "skip" {
		return weekendPolicyOptions[1]
	}
	return weekendPolicyOptions[0]
}

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
		// A timed item has its own scheduler reminder and Daybook popup;
		// showing it here as an earlier SOD/SOM suggestion would create
		// two prompts for the same occurrence.
		if strings.TrimSpace(r.Time) != "" {
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

var recurringCadenceRank = map[string]int{
	"daily":   0,
	"weekly":  1,
	"monthly": 2,
}

// sortRecurringItems keeps the management window and config output in a
// predictable cadence order, with time and recurrence details breaking ties.
func sortRecurringItems(items []RecurringItem) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		ra, rb := recurringCadenceRank[a.Cadence], recurringCadenceRank[b.Cadence]
		if ra != rb {
			return ra < rb
		}
		if a.Time != b.Time {
			return a.Time < b.Time
		}
		if a.DOW != b.DOW {
			return a.DOW < b.DOW
		}
		if a.DayOfMonth != b.DayOfMonth {
			return a.DayOfMonth < b.DayOfMonth
		}
		if a.Category != b.Category {
			return a.Category < b.Category
		}
		return a.Text < b.Text
	})
}

// recurringItemOccurrence returns today's scheduled reminder, if r is due
// today and has a valid optional time.
func recurringItemOccurrence(r RecurringItem, now time.Time) (time.Time, bool) {
	if strings.TrimSpace(r.Time) == "" || !hmPattern.MatchString(strings.TrimSpace(r.Time)) || !r.isDueToday(now) {
		return time.Time{}, false
	}
	hour, minute := parseHM(strings.TrimSpace(r.Time))
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), true
}

// dueForRecurringItemReminder tolerates scheduler drift up to window after
// the configured time, while firedForRecurringItems prevents duplicate alerts.
func dueForRecurringItemReminder(r RecurringItem, now time.Time, window time.Duration) bool {
	occurrence, ok := recurringItemOccurrence(r, now)
	if !ok {
		return false
	}
	delta := occurrence.Sub(now)
	return delta <= 0 && delta >= -window
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

// recurringEntryLabel renders the user-entered part with Daybook's green tag
// treatment and de-emphasizes the recurrence details after the em dash.
func recurringEntryLabel(text, detail string) fyne.CanvasObject {
	detailText := canvas.NewText(" — "+detail, theme.Color(theme.ColorNameForeground))
	detailText.TextStyle = fyne.TextStyle{Italic: true}
	return container.New(newTightRowLayout(), itemTextLabel(text), detailText)
}

// showRecurringItemsDialog lets the user add/edit/delete recurring
// TODO/GOAL entries, persisted in config.toml's recurring_item
// array-of-tables (see Config.RecurringItems). Each existing item gets
// inline edit and delete controls, with an optional time for an active
// Daybook reminder.
func showRecurringItemsDialog(a fyne.App, parent fyne.Window) {
	cfg := LoadConfig()
	items := append([]RecurringItem(nil), cfg.RecurringItems...)

	itemsBox := container.NewVBox()
	var refreshItems func()
	var beginRecurringItemEdit func(int)

	saveAll := func() {
		sortRecurringItems(items)
		newCfg := cfg
		newCfg.RecurringItems = items
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, parent)
		}
	}

	refreshItems = func() {
		sortRecurringItems(items)
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
				detail = "weekly (" + safeDOW(r.DOW) + ")"
			case "monthly":
				detail = "monthly (day " + strconv.Itoa(r.DayOfMonth) + ")"
			}
			if r.Time != "" {
				detail += " at " + r.Time
			}
			editBtn := widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() {
				beginRecurringItemEdit(i)
			})
			row := container.NewBorder(nil, nil, nil,
				container.NewHBox(editBtn, widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDelete), func() {
					items = append(items[:i], items[i+1:]...)
					saveAll()
					refreshItems()
				})),
				recurringEntryLabel(r.Category+": "+r.Text, detail))
			itemsBox.Add(row)
		}
		itemsBox.Refresh()
	}
	refreshItems()

	// Recurring items are repeated actions or aims. KUDOS is a notable
	// event recorded when it happens, while the other plan categories are
	// either reactive or already covered by the recurring-meeting flow.
	recurringItemCategories := []string{"TODO", "GOAL"}
	catSelect := widget.NewSelect(recurringItemCategories, nil)
	catSelect.SetSelected("TODO")

	textEntry := widget.NewEntry()
	textEntry.SetPlaceHolder("Item text\u2026")

	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("HH:MM (optional)")
	timeWrapper := container.NewGridWrap(fyne.NewSize(132, timeEntry.MinSize().Height), timeEntry)

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

	editingIndex := -1
	var addBtn *widget.Button
	cancelEditBtn := widget.NewButton("Cancel", nil)
	cancelEditBtn.Hide()
	resetForm := func() {
		editingIndex = -1
		catSelect.SetOptions(recurringItemCategories)
		catSelect.SetSelected("TODO")
		textEntry.SetText("")
		timeEntry.SetText("")
		cadenceSelect.SetSelected("daily")
		weekendSelect.SetSelected(weekendPolicyOptions[0])
		dowSelect.SetSelected(dowNames[time.Monday])
		domEntry.SetText("")
		cadenceSelect.OnChanged("daily")
		addBtn.SetText("Add")
		cancelEditBtn.Hide()
	}
	beginRecurringItemEdit = func(index int) {
		if index < 0 || index >= len(items) {
			return
		}
		r := items[index]
		editingIndex = index
		catOptions := append([]string(nil), recurringItemCategories...)
		foundCategory := false
		for _, option := range catOptions {
			if option == r.Category {
				foundCategory = true
				break
			}
		}
		if !foundCategory {
			// Keep legacy values such as KUDOS editable without
			// offering them for new recurring items.
			catOptions = append(catOptions, r.Category)
		}
		catSelect.SetOptions(catOptions)
		catSelect.SetSelected(r.Category)
		textEntry.SetText(r.Text)
		timeEntry.SetText(r.Time)
		cadenceSelect.SetSelected(r.Cadence)
		weekendSelect.SetSelected(weekendPolicyFor(r.WeekendPolicy))
		if r.DOW >= 0 && r.DOW < len(dowNames) {
			dowSelect.SetSelected(dowNames[r.DOW])
		}
		domEntry.SetText(strconv.Itoa(r.DayOfMonth))
		cadenceSelect.OnChanged(r.Cadence)
		addBtn.SetText("Save")
		cancelEditBtn.Show()
	}
	var addItem func()
	addBtn = widget.NewButton("Add", func() { addItem() })
	addItem = func() {
		text := strings.TrimSpace(textEntry.Text)
		if text == "" {
			dialog.ShowError(errors.New("text is required"), parent)
			return
		}
		reminderTime := strings.TrimSpace(timeEntry.Text)
		if reminderTime != "" && !hmPattern.MatchString(reminderTime) {
			dialog.ShowError(errors.New("time must be HH:MM (24-hour) or blank"), parent)
			return
		}
		r := RecurringItem{
			Category: catSelect.Selected,
			Text:     text,
			Cadence:  cadenceSelect.Selected,
			Time:     reminderTime,
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
		if editingIndex >= 0 && editingIndex < len(items) {
			items[editingIndex] = r
		} else {
			items = append(items, r)
		}
		saveAll()
		refreshItems()
		resetForm()
	}
	cancelEditBtn.OnTapped = resetForm
	// Enter in the text field submits, same as the main Daybook entry
	// (ui.go) and SOD's quick-add field.
	textEntry.OnSubmitted = func(string) { addItem() }
	domEntry.OnSubmitted = func(string) { addItem() }
	timeEntry.OnSubmitted = func(string) { addItem() }

	helpLine := widget.NewLabelWithStyle("📝 Untimed entries are suggested in Start of Day / Start of Month. Add an optional HH:MM time for a native reminder and a prefilled Daybook popup.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	helpLine.Wrapping = fyne.TextWrapWord
	helpLine.SizeName = theme.SizeNameCaptionText

	heading := widget.NewLabelWithStyle("🔁 Recurring Items", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// entryRow stretches textEntry to fill remaining width (same
	// stretchRowLayout approach as ui.go's doneWrapper), rather than
	// letting Fyne's default layout render it at an oddly narrow
	// width.
	entryRow := container.New(newStretchRowLayout(textEntry), catSelect, textEntry)
	actionsRow := container.NewHBox(cadenceSelect, weekendSelect, dowSelect, domWrapper, timeWrapper, addBtn, cancelEditBtn)
	itemsScroll := container.NewVScroll(itemsBox)
	itemsScroll.SetMinSize(fyne.NewSize(0, 170))

	content := container.NewVBox(
		heading,
		helpLine,
		entryRow,
		actionsRow,
		container.NewPadded(widget.NewSeparator()),
		itemsScroll,
	)

	w := a.NewWindow("Dunnit: Recurring Items")
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(520, 440))
	w.Show()
}
