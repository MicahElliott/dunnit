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

// startOfDayPending reports whether today's configured Day Kickoff has not
// run yet. Weekends and configured holidays follow the scheduler's behavior.
func startOfDayPending(cfg Config, now time.Time) bool {
	if !cfg.KickoffDayEnabled || isOffDay(cfg, now) {
		return false
	}
	return cfg.LastStartOfDayDate != now.Format("2006-01-02")
}

var sodContextCategories = map[string]bool{
	"GOAL":     true,
	"RISK":     true,
	"WAITING":  true,
	"QUESTION": true,
}

func dayReflection(entries []LedgerEntry, date time.Time) string {
	productivity := ""
	sentiment := ""
	for _, e := range entries {
		if !e.Date.Equal(date) {
			continue
		}
		switch e.Category {
		case "PRODUCTIVITY":
			productivity = strings.TrimSpace(e.Text)
		case "SENTIMENT":
			sentiment = strings.TrimSpace(e.Text)
		}
	}
	var parts []string
	if productivity != "" {
		parts = append(parts, "productivity "+productivity+"/5")
	}
	if sentiment != "" {
		parts = append(parts, strings.ToLower(sentiment))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " · ")
}

func sodReport(date time.Time) string {
	_, path := eodReportPath(date)
	contents, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(contents))
}

// markStartOfDayRun records that today's Day Kickoff/Start of Day routine
// was opened. A config read error is left untouched so a malformed config
// cannot be replaced with defaults while recording this UI state.
func markStartOfDayRun() {
	cfg, err := loadConfig()
	if err != nil {
		log.Println("Skipping Start of Day marker:", err)
		return
	}
	today := time.Now().Format("2006-01-02")
	if cfg.LastStartOfDayDate == today {
		return
	}
	cfg.LastStartOfDayDate = today
	if err := writeConfig(cfg); err != nil {
		log.Println("Error saving Start of Day marker:", err)
	}
}

// showSODWindow opens the daily planning checkpoint: it carries forward the
// newest prior day's TODO/DOING items, shows the last active day's context and
// EOD reflection, and offers a quick-entry field for today's plan.
func showSODWindow(a fyne.App) {
	now := time.Now()
	carrySource, _ := carryForwardDailyPlan(now)
	entries := AllLedgerEntries()
	lastActive, hasLastActive := lastActiveLedgerDate(entries, now)

	w := a.NewWindow("Dunnit: Start of Day")

	planBox := container.NewVBox()
	refreshPlan := func() {
		planBox.RemoveAll()
		var plan []OpenItem
		for _, item := range getOpenItems() {
			if dailyCarryCategories[item.Category] {
				plan = append(plan, item)
			}
		}
		if len(plan) == 0 {
			planBox.Add(widget.NewLabel("No TODOs carried in yet. Add one below or from Daybook."))
		} else {
			for _, item := range plan {
				planBox.Add(widget.NewLabel("\u2022 " + stripCarryForwardSince(item.Text) + staleBadge(item.Text)))
			}
		}
		planBox.Refresh()
	}
	refreshPlan()

	planHeading := "Today's plan"
	if !carrySource.IsZero() {
		planHeading = "Carried into today from " + carrySource.Format("Mon Jan 2")
	}
	var planNote *widget.Label
	if !carrySource.IsZero() {
		planNote = widget.NewLabel("These items are carrying into today. Edit or remove them in Daybook.")
		planNote.Wrapping = fyne.TextWrapWord
	} else {
		planNote = widget.NewLabel("Add a TODO below or from Daybook.")
	}
	planScroll := container.NewVScroll(planBox)
	planScroll.SetMinSize(fyne.NewSize(0, 150))

	contextBox := container.NewVBox()
	if hasLastActive {
		contextItems := openItemsAtDate(entries, lastActive, now)
		contextCount := 0
		for _, item := range contextItems {
			if sodContextCategories[item.Category] {
				contextBox.Add(widget.NewLabel("\u2022 " + item.Category + ": " + stripCarryForwardSince(item.Text)))
				contextCount++
			}
		}
		if contextCount == 0 {
			contextBox.Add(widget.NewLabel("No open reminders from the last active day."))
		}
	} else {
		contextBox.Add(widget.NewLabel("No previous active day yet."))
	}
	contextScroll := container.NewVScroll(contextBox)
	contextScroll.SetMinSize(fyne.NewSize(0, 100))

	staleItems := staleDailyPlanItems(now)
	staleBox := container.NewVBox()
	if len(staleItems) > 0 {
		staleBox.Add(widget.NewLabelWithStyle("Stale TODOs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		staleBox.Add(widget.NewLabel("These have been open for at least seven days."))
		for _, item := range staleItems {
			staleBox.Add(widget.NewLabel("\u2022 " + stripCarryForwardSince(item.Text) + staleBadge(item.Text)))
		}
		staleBox.Add(widget.NewButton("Move stale TODOs to SOMEDAY", func() {
			for _, item := range staleItems {
				recordPostponed(item)
			}
			staleBox.RemoveAll()
			staleBox.Refresh()
			refreshPlan()
		}))
	}

	reportBox := container.NewVBox()
	if hasLastActive {
		if reflection := dayReflection(entries, lastActive); reflection != "" {
			reportBox.Add(widget.NewLabel("Last active day: " + lastActive.Format("Mon Jan 2") + " · " + reflection))
		}
		if report := sodReport(lastActive); report != "" {
			reportText := widget.NewRichTextFromMarkdown(report)
			reportText.Wrapping = fyne.TextWrapWord
			reportScroll := container.NewVScroll(reportText)
			reportScroll.SetMinSize(fyne.NewSize(0, 120))
			reportBox.Add(widget.NewLabelWithStyle("Last EOD report", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			reportBox.Add(reportScroll)
		}
	}

	// Recurring items (daily/weekly cadence) are suggestions the user
	// explicitly adds, rather than entries seeded without consent.
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
		if box := recurringItemsSuggestionBox(dailyWeekly, refreshPlan); box != nil {
			recurringBox.Add(widget.NewLabelWithStyle("Recurring Items Due Today", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
			recurringBox.Add(box)
		}
		recurringBox.Refresh()
	}
	refreshRecurring()

	newItemCat := widget.NewSelect([]string{"TODO", "DOING"}, nil)
	newItemCat.SetSelected("TODO")
	newItemText := newTagAutoEntry()
	newItemText.SetPlaceHolder("Add a TODO for today…")
	newItemSuggestions := newItemText.SuggestionBox()
	addItem := func() {
		text := strings.TrimSpace(newItemText.Text)
		if text == "" {
			return
		}
		recordActivity(text, newItemCat.Selected)
		newItemText.SetText("")
		refreshPlan()
		refreshRecurring()
	}
	newItemText.OnSubmitted = func(string) { addItem() }
	addBtn := widget.NewButton("Add", addItem)
	entryRow := container.New(newStretchRowLayout(newItemText), newItemCat, newItemText, addBtn)

	done := func() {
		markStartOfDayRun()
		if trayRefreshAll != nil {
			trayRefreshAll()
		} else if refreshStartOfDayNotice != nil {
			refreshStartOfDayNotice()
		}
		w.Close()
	}

	content := container.NewVBox(
		widget.NewLabelWithStyle("Let's get your day planned.", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		streakLabel(),
		reportBox,
		widget.NewLabelWithStyle(planHeading, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		planNote,
		planScroll,
		staleBox,
		widget.NewLabelWithStyle("From the last active day", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		contextScroll,
		recurringBox,
		entryRow,
		newItemSuggestions,
		widget.NewButton("Done", done),
	)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(560, 640))
	w.Show()
}
