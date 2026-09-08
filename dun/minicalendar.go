package dun

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// hmPattern validates a strict "HH:MM" 24-hour time string, since
// parseHM silently returns zeros on any parse failure (which would
// otherwise be indistinguishable from a legitimately-entered
// "00:00").
var hmPattern = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)

// RecurringMeeting is one user-entered recurring meeting slot
// (FR-15) -- purely user-entered, no calendar/.ics/EventKit
// integration. Tag should include the leading "#" (normalized on
// save). Cadence is "weekly" (default/legacy, uses DOW+IntervalWeeks)
// or "daily" (fires every day at Time, or every weekday if
// WeekendPolicy is "skip" -- the common shape for a daily standup,
// letting one entry replace 5 separate weekday rows). DOW is
// 0=Sunday..6=Saturday (matches time.Weekday, only meaningful for
// "weekly") so it sorts/compares naturally against
// time.Now().Weekday(). Time is "HH:MM" 24-hour. IntervalWeeks is
// "every N weeks" (1 = every week, the common case; 2 = biweekly,
// etc; treated as 1 if <= 0, e.g. for entries saved before this field
// existed; not meaningful for "daily"). AnchorDate ("YYYY-MM-DD") is
// the first occurrence's date, used to compute which weeks count for
// IntervalWeeks > 1 -- without it there'd be no way to know which
// week is "week 1" of the cadence. Set once at creation and left
// alone afterward.
type RecurringMeeting struct {
	Tag           string `toml:"tag"`
	Cadence       string `toml:"cadence"` // "weekly" (default/legacy) or "daily"
	DOW           int    `toml:"dow"`
	Time          string `toml:"time"`
	IntervalWeeks int    `toml:"interval_weeks"`
	AnchorDate    string `toml:"anchor_date"`
	WeekendPolicy string `toml:"weekend_policy"` // "include" (default) or "skip" -- daily only
}

// meetingCadenceOptions are the choices for a RecurringMeeting's
// cadence -- "Weekly" is the original/default shape (day-of-week +
// every-N-weeks interval); "Daily" fires every day (or every weekday,
// per WeekendPolicy), replacing what used to require 5 separate
// weekly rows (one per weekday) for something like a daily standup.
var meetingCadenceOptions = []string{"Weekly", "Daily"}

// dowNames indexes by time.Weekday (0=Sunday..6=Saturday).
var dowNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// showMiniCalendarDialog lets the user add/edit/delete recurring
// meeting entries (FR-15), persisted in config.toml's
// recurring_meeting array-of-tables (see Config.RecurringMeetings).
func showMiniCalendarDialog(a fyne.App, parent fyne.Window) {
	cfg := LoadConfig()
	meetings := append([]RecurringMeeting(nil), cfg.RecurringMeetings...)

	list := widget.NewList(
		func() int { return len(meetings) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			m := meetings[i]
			var detail string
			if m.Cadence == "daily" {
				detail = "daily " + m.Time
				if m.WeekendPolicy == "skip" {
					detail += " (weekdays only)"
				}
			} else {
				interval := m.IntervalWeeks
				if interval <= 0 {
					interval = 1
				}
				every := "every week"
				if interval > 1 {
					every = fmt.Sprintf("every %d weeks", interval)
				}
				detail = dowNames[m.DOW] + " " + m.Time + " (" + every + ")"
			}
			o.(*widget.Label).SetText(m.Tag + " \u2014 " + detail)
		},
	)

	tagEntry := newTagAutoEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #dsu, #boss)")

	// Tag autocomplete, same tagAutoEntry approach as the main entry
	// field (FR-10, see ui.go/tagautoentry.go) -- suggests previously-
	// used tags from ledger history as the user types "#...", as an
	// inline sibling suggestion list (not a canvas overlay -- see
	// tagautoentry.go's doc comment for why overlays broke keyboard
	// input entirely).
	tagSuggestions := tagEntry.SuggestionBox()

	cadenceSelect := widget.NewSelect(meetingCadenceOptions, nil)
	cadenceSelect.SetSelected("Weekly")

	dowSelect := widget.NewSelect(dowNames, nil)
	dowSelect.SetSelected(dowNames[time.Monday])

	// weekendSelect (only relevant/shown for "Daily") mirrors
	// recurring.go's weekendPolicyOptions dropdown for the same
	// feature on RecurringItem, for consistency between the two
	// recurring-something dialogs.
	weekendSelect := widget.NewSelect(weekendPolicyOptions, nil)
	weekendSelect.SetSelected(weekendPolicyOptions[0])
	weekendSelect.Hide()

	// timeEntry/intervalEntry are wrapped in fixed-size GridWrap
	// containers (same fix as minsInput in ui.go) -- otherwise Fyne's
	// default layout can render a plain widget.Entry at an oddly
	// narrow width alongside other fixed-width siblings in an HBox.
	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("HH:MM")
	timeWrapper := container.NewGridWrap(fyne.NewSize(80, timeEntry.MinSize().Height), timeEntry)

	intervalEntry := widget.NewEntry()
	intervalEntry.SetPlaceHolder("1")
	intervalEntry.SetText("1")
	intervalWrapper := container.NewGridWrap(fyne.NewSize(50, intervalEntry.MinSize().Height), intervalEntry)
	intervalLabel := widget.NewLabel("every")
	weekLabel := widget.NewLabel("week(s)")

	cadenceSelect.OnChanged = func(c string) {
		daily := c == "Daily"
		if daily {
			dowSelect.Hide()
			intervalLabel.Hide()
			intervalWrapper.Hide()
			weekLabel.Hide()
			weekendSelect.Show()
		} else {
			dowSelect.Show()
			intervalLabel.Show()
			intervalWrapper.Show()
			weekLabel.Show()
			weekendSelect.Hide()
		}
	}

	saveAll := func() {
		newCfg := cfg
		newCfg.RecurringMeetings = meetings
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, parent)
		}
	}

	addBtn := widget.NewButton("Add", func() {
		tag := normalizeTag(tagEntry.Text)
		if tag == "" {
			dialog.ShowError(errors.New("tag is required"), parent)
			return
		}
		if !hmPattern.MatchString(timeEntry.Text) {
			dialog.ShowError(errors.New("time must be HH:MM (24-hour)"), parent)
			return
		}
		if cadenceSelect.Selected == "Daily" {
			weekendPolicy := ""
			if weekendSelect.Selected == weekendPolicyOptions[1] {
				weekendPolicy = "skip"
			}
			meetings = append(meetings, RecurringMeeting{
				Tag:           tag,
				Cadence:       "daily",
				Time:          timeEntry.Text,
				WeekendPolicy: weekendPolicy,
			})
			saveAll()
			list.Refresh()
			tagEntry.SetText("")
			timeEntry.SetText("")
			return
		}
		interval, err := strconv.Atoi(strings.TrimSpace(intervalEntry.Text))
		if err != nil || interval <= 0 {
			dialog.ShowError(errors.New("every N weeks must be a positive number"), parent)
			return
		}
		dow := 0
		for i, n := range dowNames {
			if n == dowSelect.Selected {
				dow = i
			}
		}
		meetings = append(meetings, RecurringMeeting{
			Tag:           tag,
			Cadence:       "weekly",
			DOW:           dow,
			Time:          timeEntry.Text,
			IntervalWeeks: interval,
			AnchorDate:    time.Now().Format("2006-01-02"),
		})
		saveAll()
		list.Refresh()
		tagEntry.SetText("")
		timeEntry.SetText("")
		intervalEntry.SetText("1")
	})

	var selected widget.ListItemID = -1
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	deleteBtn := widget.NewButton("Delete Selected", func() {
		if selected < 0 || selected >= len(meetings) {
			return
		}
		meetings = append(meetings[:selected], meetings[selected+1:]...)
		saveAll()
		list.Refresh()
		selected = -1
	})

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Recurring Meetings"),
			widget.NewRichTextFromMarkdown("*Use these tags throughout your weeks any time a meeting topic thought comes to mind. They\u2019ll be collected and presented to you just before your meeting starts.*"),
			tagEntry,
			tagSuggestions,
			container.NewHBox(cadenceSelect, dowSelect, timeWrapper, intervalLabel, intervalWrapper, weekLabel, weekendSelect, addBtn),
		),
		deleteBtn,
		nil, nil,
		list,
	)

	w := a.NewWindow("Recurring Meetings")
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(560, 420))
	w.Show()
}

// nextOccurrence returns the next time (today or later) that m is
// scheduled, given now. For "daily" cadence, that's simply today's
// (or tomorrow's, if today's time has passed) occurrence at m.Time,
// skipping weekends if WeekendPolicy is "skip" -- no DOW/
// IntervalWeeks involved. For "weekly" (default/legacy), IntervalWeeks
// > 1 only counts weeks that are an exact multiple of IntervalWeeks
// away from AnchorDate's week count; candidates in between are
// skipped. Falls back to every-week behavior if AnchorDate is
// missing/unparseable (e.g. legacy entries).
func nextOccurrence(m RecurringMeeting, now time.Time) time.Time {
	hh, mm := parseHM(m.Time)

	if m.Cadence == "daily" {
		candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		for m.WeekendPolicy == "skip" && (candidate.Weekday() == time.Saturday || candidate.Weekday() == time.Sunday) {
			candidate = candidate.AddDate(0, 0, 1)
		}
		return candidate
	}

	interval := m.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}

	daysAhead := (m.DOW - int(now.Weekday()) + 7) % 7
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location()).AddDate(0, 0, daysAhead)
	if candidate.Before(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	if interval == 1 {
		return candidate
	}

	anchor, err := time.ParseInLocation("2006-01-02", m.AnchorDate, now.Location())
	if err != nil {
		return candidate
	}
	for {
		weeksSinceAnchor := int(candidate.Sub(anchor).Hours() / (24 * 7))
		if weeksSinceAnchor >= 0 && weeksSinceAnchor%interval == 0 {
			return candidate
		}
		candidate = candidate.AddDate(0, 0, 7)
	}
}

// dueForPreMeetingNudge reports whether m's next occurrence starts
// within the next window duration from now (FR-16 -- "~15 min
// before"). Since the scheduler check itself runs periodically (every
// 15 min, per FR-16 v1), window should be set a bit wider than the
// check interval to avoid missing a meeting between checks.
func dueForPreMeetingNudge(m RecurringMeeting, now time.Time, window time.Duration) bool {
	next := nextOccurrence(m, now)
	delta := next.Sub(now)
	return delta > 0 && delta <= window
}

// lastOccurrence returns the most recent past occurrence of m at or
// before now (the mirror of nextOccurrence). Used by FR-36's post-
// meeting nudge, which looks backward instead of forward.
func lastOccurrence(m RecurringMeeting, now time.Time) time.Time {
	next := nextOccurrence(m, now)
	if next.Equal(now) {
		return next
	}
	// nextOccurrence never returns a time <= now, so the actual most
	// recent past occurrence is exactly one cadence period before
	// whatever it returns when starting the search from just after
	// that prior occurrence. Simplest robust approach: step backward
	// one cadence unit at a time (a day for "daily", a week at a time
	// respecting IntervalWeeks for "weekly") until we find an
	// occurrence at or before now.
	if m.Cadence == "daily" {
		candidate := next.AddDate(0, 0, -1)
		for candidate.After(now) || (m.WeekendPolicy == "skip" && (candidate.Weekday() == time.Saturday || candidate.Weekday() == time.Sunday)) {
			candidate = candidate.AddDate(0, 0, -1)
		}
		return candidate
	}
	interval := m.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}
	candidate := next.AddDate(0, 0, -7*interval)
	for candidate.After(now) {
		candidate = candidate.AddDate(0, 0, -7*interval)
	}
	return candidate
}

// dueForPostMeetingNudge was FR-36's original post-meeting-window
// check (fired 15-45 min after a meeting's start, since duration
// isn't tracked). Removed 2026-09-03: Post-Meeting Capture now opens
// alongside Meeting Prep at meeting *start* instead (sched.go), so
// there's no separate after-the-fact window to check for. Left as a
// comment (not deleted outright) in case a future "actually track
// meeting end" feature wants this shape again -- if reintroduced,
// this exact function signature is a good starting point.
