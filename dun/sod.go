package dun

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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
	"FIXME":    true,
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

func lastEODReport(now time.Time) (date time.Time, report, path string) {
	var latest time.Time
	err := filepath.Walk(DunnitDir(), func(candidate string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := filepath.Base(candidate)
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "eod-"), ".md")
		if datePart == name || !strings.HasPrefix(name, "eod-") || !strings.HasSuffix(name, ".md") {
			return nil
		}
		candidateDate, parseErr := time.ParseInLocation("Mon-20060102", datePart, time.Local)
		if parseErr != nil || candidateDate.Format("Mon-20060102") != datePart || !candidateDate.Before(now) || !candidateDate.After(latest) {
			return nil
		}
		contents, readErr := os.ReadFile(candidate)
		if readErr != nil || strings.TrimSpace(string(contents)) == "" {
			return nil
		}
		latest = candidateDate
		date = candidateDate
		report = strings.TrimSpace(string(contents))
		path = candidate
		return nil
	})
	if err != nil {
		log.Println("Error scanning for the last EOD report:", err)
	}
	return date, report, path
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
	// Ledger files may have been changed by git sync or another Dunnit
	// process since the five-minute index refresh. SOD is a daily boundary,
	// so its date/context decisions should always use the current files.
	InvalidateLedgerCaches()
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
				planBox.Add(itemTextLabel(categoryIconPrefix(item.Category) + openItemDisplayText(item.Text)))
			}
		}
		planBox.Refresh()
	}
	refreshPlan()

	planHeading := "Today’s plan"
	if !carrySource.IsZero() {
		planHeading = "Carried into today from " + carrySource.Format("Mon Jan 2")
	}
	var planNote *widget.Label
	if !carrySource.IsZero() {
		planNote = newExplanatoryLabel("These items are carrying into today. Edit or remove them in Daybook.")
	} else {
		planNote = newExplanatoryLabel("Add a TODO below or from Daybook.")
	}
	contextBox := container.NewVBox()
	if hasLastActive {
		contextItems := openItemsAtDate(entries, lastActive, now)
		contextCount := 0
		for _, item := range contextItems {
			if sodContextCategories[item.Category] {
				contextBox.Add(itemTextLabel(categoryIconPrefix(item.Category) + item.Category + ": " + openItemDisplayText(item.Text)))
				contextCount++
			}
		}
		if contextCount == 0 {
			contextBox.Add(widget.NewLabel("No open reminders from the last active day."))
		}
	} else {
		contextBox.Add(widget.NewLabel("No previous active day yet."))
	}
	staleBox := container.NewVBox()
	var refreshStale func()
	refreshStale = func() {
		staleBox.RemoveAll()
		staleItems := staleDailyPlanItems(time.Now())
		staleBox.Add(widget.NewLabelWithStyle(
			fmt.Sprintf("Stale TODOs (open %d+ days)", staleReviewDays),
			fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		staleBox.Add(newExplanatoryLabel(fmt.Sprintf(
			"Review scans the previous %d calendar days. Daily carry searches the previous %d days. These items remain active in Daybook until you complete, postpone, or discard them.",
			staleReviewLookbackDays, dailyCarryLookbackDays)))
		if len(staleItems) == 0 {
			staleBox.Add(widget.NewLabel("Nothing needs a stale-item decision."))
			staleBox.Refresh()
			return
		}
		for _, stale := range staleItems {
			item := stale.OpenItem
			actions := container.NewHBox(
				newHoverIconButton(theme.Icon(theme.IconNameDelete), "Delete", func() {
					recordDiscarded(item)
					refreshStale()
					refreshPlan()
				}),
				newHoverIconButton(theme.Icon(theme.IconNameHistory), "Postpone", func() {
					recordPostponed(item)
					refreshStale()
					refreshPlan()
				}),
				newHoverIconButton(theme.Icon(theme.IconNameConfirm), "Done", func() {
					recordConvertedDone(item)
					refreshStale()
					refreshPlan()
				}),
			)
			staleBox.Add(container.NewBorder(nil, nil, nil, actions,
				itemTextLabel(categoryIconPrefix(item.Category)+openItemDisplayText(item.Text)+
					" · since "+stale.Since.Format("Jan 2, 2006"))))
		}
		staleBox.Refresh()
	}
	refreshStale()

	reportBox := container.NewVBox()
	if hasLastActive {
		if reflection := dayReflection(entries, lastActive); reflection != "" {
			reportBox.Add(widget.NewLabel("Last active day: " + lastActive.Format("Mon Jan 2") + " · " + reflection))
		}
	}
	if reportDate, report, reportPath := lastEODReport(now); !reportDate.IsZero() {
		reportBox.Add(widget.NewLabelWithStyle("Last EOD report — "+reportDate.Format("Mon Jan 2"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		reportBox.Add(newExplanatoryLabel("The full report opens in its own window, with its date and stats at the top."))
		reportBox.Add(widget.NewButton("See full EOD report", func() {
			showEODReport(a, reportDate, reportPath, report)
		}))
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
		widget.NewLabelWithStyle("Let’s get your day planned.", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		streakLabel(),
		reportBox,
		widget.NewLabelWithStyle(planHeading, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		planNote,
		planBox,
		staleBox,
		widget.NewLabelWithStyle("Open context from the last active day (not copied into today’s plan)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newExplanatoryLabel("WAITING, RISK, QUESTION, FIXME, and GOAL stay here for context; only TODO and DOING become today’s active plan."),
		contextBox,
		recurringBox,
		entryRow,
		newItemSuggestions,
		widget.NewButton("Done", done),
	)

	w.SetContent(windowPad(container.NewVScroll(content)))
	w.Resize(fyne.NewSize(560, 640))
	w.Show()
}
