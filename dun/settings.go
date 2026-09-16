package dun

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func SetThings() {
	fmt.Println("Setting things")
}

// showSettings pops up a window to view/edit config.toml values
// (day_start, day_end, hourly_minute, lunch_time).
func showSettings(a fyne.App) {
	cfg := LoadConfig()
	w := a.NewWindow("Dunnit: Settings")
	dunnitDir := widget.NewEntry()
	dunnitDir.SetText(cfg.DunnitDir)
	dunnitDir.SetPlaceHolder("Leave blank for default")
	llmCLISelect := widget.NewSelect(llmCLISettingOptions(), nil)
	llmCLISelect.SetSelected(llmCLISettingLabel(cfg.LLMCLI))
	browseDir := widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uri != nil {
				dunnitDir.SetText(uri.Path())
			}
		}, w)
	})
	gitSync := widget.NewCheck("", nil)
	gitSync.SetChecked(cfg.GitSyncEnabled)

	dayStart := widget.NewEntry()
	dayStart.SetText(cfg.DayStart)

	dayEnd := widget.NewEntry()
	dayEnd.SetText(cfg.DayEnd)

	nudgeInterval := widget.NewEntry()
	nudgeInterval.SetText(strconv.Itoa(cfg.NudgeIntervalMinutes))

	lunchTime := widget.NewEntry()
	lunchTime.SetText(cfg.LunchTime)

	digestDay := widget.NewSelect([]string{"", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}, nil)
	digestDay.SetSelected(cfg.WeeklyDigestDay)

	digestTime := widget.NewEntry()
	digestTime.SetText(cfg.WeeklyDigestTime)
	digestTime.SetPlaceHolder("HH:MM")

	autoDraft := widget.NewCheck("", nil)
	autoDraft.SetChecked(cfg.AutoDraftDailySummary)

	snoozeMinutes := widget.NewEntry()
	snoozeMinutes.SetText(strconv.Itoa(cfg.SnoozeMinutes))

	skipHolidays := widget.NewCheck("", nil)
	skipHolidays.SetChecked(cfg.SkipUSFederalHolidays)

	extendWorkWeek := widget.NewCheck("", nil)
	extendWorkWeek.SetChecked(cfg.ExtendWorkWeekTo7Days)

	enableOKRs := widget.NewCheck("", nil)
	enableOKRs.SetChecked(cfg.EnableOKRs)

	wastedTimeTracking := widget.NewCheck("", nil)
	wastedTimeTracking.SetChecked(cfg.WastedTimeTrackingEnabled)

	// Faves bucket (categories.go/CategoryLabelsForFaves): multi-
	// select of category codes, in Categories' declared order.
	// widget.NewCheckGroup wants a []string of labels to check off,
	// same as the category codes themselves (not the emoji Label()
	// form) -- plain codes read better in a checkbox list than the
	// emoji-prefixed picker labels.
	var allCategoryCodes []string
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		allCategoryCodes = append(allCategoryCodes, c.Code)
	}
	favesGroup := widget.NewCheckGroup(allCategoryCodes, nil)
	favesGroup.Selected = append([]string{}, cfg.FavoriteCategories...)
	favesGroup.Refresh()

	// Report exclude-tags: entered as freeform space/comma-separated
	// text (e.g. "#home, #personal, #buy, #shop") rather than a
	// per-tag checkbox list -- tags are open-ended/user-coined (see
	// tags.go), unlike categories' small fixed set, so a full
	// checkbox list isn't practical here.
	excludeTagsEntry := widget.NewEntry()
	excludeTagsEntry.SetText(strings.Join(cfg.ReportExcludeTags, ", "))
	excludeTagsEntry.SetPlaceHolder("#home, #personal, #buy, #shop")

	form := widget.NewForm(
		widget.NewFormItem("Dunnit Data Directory", container.NewBorder(nil, nil, nil, browseDir, dunnitDir)),
		widget.NewFormItem("LLM CLI for Reports", llmCLISelect),
		widget.NewFormItem("Enable Git Sync", gitSync),
		widget.NewFormItem("Day Start (HH:MM)", dayStart),
		widget.NewFormItem("Day End (HH:MM)", dayEnd),
		widget.NewFormItem("Nudge Interval (minutes)", nudgeInterval),
		widget.NewFormItem("Lunch Time (HH:MM)", lunchTime),
		widget.NewFormItem("Weekly Digest Day", digestDay),
		widget.NewFormItem("Weekly Digest Time (HH:MM)", digestTime),
		widget.NewFormItem("Auto-draft Daily Summary at EOD", autoDraft),
		widget.NewFormItem("Default Snooze (minutes)", snoozeMinutes),
		widget.NewFormItem("Skip US Federal Holidays", skipHolidays),
		widget.NewFormItem("Extend Work Week to 7 Days", extendWorkWeek),
		widget.NewFormItem("Enable OKRs (Quarter/Year Kickoff+Review)", enableOKRs),
		widget.NewFormItem("Track Wasted Time (WASTED category)", wastedTimeTracking),
		widget.NewFormItem("Report Exclude Tags", excludeTagsEntry),
	)

	// Kickoff/Review section (docs/kickoff-review-design.md): one
	// enable-checkbox pair + one theme dropdown per unit. Built as a
	// second widget.NewForm (rather than folding into the form above)
	// since it's a visually distinct block of settings; both forms
	// share the same OnSubmit-driven save via periodToggles below.
	type unitRow struct {
		period      summaryPeriod
		kickoff     *widget.Check
		review      *widget.Check
		themeSelect *widget.Select
	}
	var unitRows []unitRow
	for _, p := range []summaryPeriod{periodDay, periodWeek, periodMonth, periodQuarter, periodYear} {
		kickoffCheck := widget.NewCheck("", nil)
		kickoffCheck.SetChecked(kickoffEnabled(cfg, p))
		reviewCheck := widget.NewCheck("", nil)
		reviewCheck.SetChecked(reviewEnabled(cfg, p))
		themeSelect := widget.NewSelect(themeOptions(), nil)
		themeSelect.SetSelected(themeDisplayNames[themeFor(cfg, p)])
		unitRows = append(unitRows, unitRow{period: p, kickoff: kickoffCheck, review: reviewCheck, themeSelect: themeSelect})
	}
	periodItems := []*widget.FormItem{}
	for _, row := range unitRows {
		periodItems = append(periodItems,
			widget.NewFormItem(string(row.period)+" Kickoff", row.kickoff),
			widget.NewFormItem(string(row.period)+" Review", row.review),
			widget.NewFormItem(string(row.period)+" Theme", row.themeSelect),
		)
	}
	periodForm := widget.NewForm(periodItems...)

	saveSettings := func() {
		minutes, err := strconv.Atoi(nudgeInterval.Text)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Nudge Interval must be a number: %w", err), w)
			return
		}
		snooze, err := strconv.Atoi(snoozeMinutes.Text)
		if err != nil || snooze <= 0 {
			dialog.ShowError(fmt.Errorf("Default Snooze must be a positive number"), w)
			return
		}
		// Start from the loaded config rather than a blank Config{}
		// so fields not represented in this form (e.g.
		// RecurringMeetings, FR-15) aren't silently wiped out on save.
		newCfg := cfg
		newCfg.DunnitDir = strings.TrimSpace(dunnitDir.Text)
		newCfg.LLMCLI = llmCLISettingFromLabel(llmCLISelect.Selected)
		newCfg.GitSyncEnabled = gitSync.Checked
		newCfg.DayStart = dayStart.Text
		newCfg.DayEnd = dayEnd.Text
		newCfg.NudgeIntervalMinutes = minutes
		newCfg.LunchTime = lunchTime.Text
		newCfg.WeeklyDigestDay = digestDay.Selected
		newCfg.WeeklyDigestTime = digestTime.Text
		newCfg.AutoDraftDailySummary = autoDraft.Checked
		newCfg.SnoozeMinutes = snooze
		newCfg.SkipUSFederalHolidays = skipHolidays.Checked
		newCfg.ExtendWorkWeekTo7Days = extendWorkWeek.Checked
		newCfg.EnableOKRs = enableOKRs.Checked
		newCfg.WastedTimeTrackingEnabled = wastedTimeTracking.Checked
		newCfg.FavoriteCategories = append([]string{}, favesGroup.Selected...)
		var excludeTags []string
		for _, part := range strings.FieldsFunc(excludeTagsEntry.Text, func(r rune) bool { return r == ',' || r == ' ' }) {
			part = strings.TrimSpace(part)
			if part != "" {
				excludeTags = append(excludeTags, part)
			}
		}
		newCfg.ReportExcludeTags = excludeTags
		for _, row := range unitRows {
			setKickoffEnabled(&newCfg, row.period, row.kickoff.Checked)
			setReviewEnabled(&newCfg, row.period, row.review.Checked)
			if theme := themeFromDisplayName(row.themeSelect.Selected); theme != "" {
				setTheme(&newCfg, row.period, theme)
			}
		}
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if os.Getenv("DUNNIT_DIR") == "" {
			configuredDunnitDir = newCfg.DunnitDir
		}
		RebuildTrayMenu()
		w.Close()
	}
	form.OnSubmit = nil

	// Recurring Meetings (FR-15) is its own dialog/config section, but
	// Settings is a natural place to also reach it from -- both are
	// "configure how Dunnit behaves" surfaces.
	recurringMeetingsBtn := widget.NewButton("Recurring Meetings\u2026", func() {
		showMiniCalendarDialog(a, w)
	})

	recurringItemsBtn := widget.NewButton("Recurring Items\u2026", func() {
		showRecurringItemsDialog(a, w)
	})

	saveButton := widget.NewButton("Save", saveSettings)
	content := container.NewVScroll(container.NewVBox(
		newExplanatoryLabel("AI reports use the selected CLI's existing login; Dunnit never asks for or stores tokens."),
		form,
		widget.NewLabelWithStyle("Kickoff / Review", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		periodForm,
		widget.NewLabelWithStyle("Faves (Daybook picker's default bucket)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		favesGroup,
		recurringMeetingsBtn, recurringItemsBtn))
	w.SetContent(container.NewBorder(nil, container.NewPadded(saveButton), nil, nil, windowPad(content)))
	w.Resize(fyne.NewSize(420, 620))
	w.Show()
}
