package dun

import (
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
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
	DayOfMonth    int    `toml:"day_of_month"`
	WeekendPolicy string `toml:"weekend_policy"` // "include" (default) or "skip" -- daily only
}

// meetingCadenceOptions are the choices for a RecurringMeeting's
// cadence -- "Weekly" is the original/default shape (day-of-week);
// biweekly odd/even and monthly/quarterly are explicit choices, while
// "Daily" fires every day (or every weekday, per WeekendPolicy).
var meetingCadenceOptions = []string{"daily", "weekly", "biweekly-odd", "biweekly-even", "monthly", "quarterly"}

var meetingCadenceRank = map[string]int{
	"daily":         0,
	"weekly":        1,
	"biweekly-odd":  2,
	"biweekly-even": 3,
	"monthly":       4,
	"quarterly":     5,
}

// dowNames indexes by time.Weekday (0=Sunday..6=Saturday).
var dowNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

func safeDOW(dow int) string {
	if dow < 0 || dow >= len(dowNames) {
		return dowNames[time.Monday]
	}
	return dowNames[dow]
}

func sortRecurringMeetings(meetings []RecurringMeeting) {
	sort.SliceStable(meetings, func(i, j int) bool {
		a, b := meetings[i], meetings[j]
		ra, rb := meetingCadenceRank[a.Cadence], meetingCadenceRank[b.Cadence]
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
		return a.Tag < b.Tag
	})
}

func recurringMeetingDetail(m RecurringMeeting) string {
	switch m.Cadence {
	case "daily":
		detail := "daily " + m.Time
		if m.WeekendPolicy == "skip" {
			detail += " (weekdays only)"
		}
		return detail
	case "monthly", "quarterly":
		return m.Cadence + " day " + strconv.Itoa(m.DayOfMonth) + " " + m.Time
	case "biweekly-odd", "biweekly-even":
		return m.Cadence + " " + safeDOW(m.DOW) + " " + m.Time
	default:
		// IntervalWeeks remains readable for legacy configs, but new
		// weekly entries no longer expose an every-N-weeks control.
		if m.IntervalWeeks > 1 {
			return "weekly legacy " + strconv.Itoa(m.IntervalWeeks) + "-week interval " + safeDOW(m.DOW) + " " + m.Time
		}
		return "weekly " + safeDOW(m.DOW) + " " + m.Time
	}
}

// showMiniCalendarDialog lets the user add/edit/delete recurring
// meeting entries (FR-15), persisted in config.toml's
// recurring_meeting array-of-tables (see Config.RecurringMeetings).
func showMiniCalendarDialog(a fyne.App, parent fyne.Window) {
	cfg := LoadConfig()
	meetings := append([]RecurringMeeting(nil), cfg.RecurringMeetings...)
	meetingsBox := container.NewVBox()
	var refreshMeetings func()
	var beginMeetingEdit func(int)

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
	cadenceSelect.SetSelected("weekly")

	dowSelect := widget.NewSelect(dowNames, nil)
	dowSelect.SetSelected(dowNames[time.Monday])

	// weekendSelect (only relevant/shown for "Daily") mirrors
	// recurring.go's weekendPolicyOptions dropdown for the same
	// feature on RecurringItem, for consistency between the two
	// recurring-something dialogs.
	weekendSelect := widget.NewSelect(weekendPolicyOptions, nil)
	weekendSelect.SetSelected(weekendPolicyOptions[0])
	weekendSelect.Hide()

	// timeEntry is wrapped in a fixed-size GridWrap
	// containers (same fix as minsInput in ui.go) -- otherwise Fyne's
	// default layout can render a plain widget.Entry at an oddly
	// narrow width alongside other fixed-width siblings in an HBox.
	timeEntry := widget.NewEntry()
	timeEntry.SetPlaceHolder("HH:MM")
	timeWrapper := container.NewGridWrap(fyne.NewSize(88, timeEntry.MinSize().Height), timeEntry)

	domEntry := widget.NewEntry()
	domEntry.SetPlaceHolder("day 1-31")
	domWrapper := container.NewGridWrap(fyne.NewSize(70, domEntry.MinSize().Height), domEntry)
	domWrapper.Hide()

	cadenceSelect.OnChanged = func(c string) {
		daily := c == "daily"
		monthly := c == "monthly" || c == "quarterly"
		if daily {
			dowSelect.Hide()
			weekendSelect.Show()
		} else {
			weekendSelect.Hide()
			if monthly {
				dowSelect.Hide()
				domWrapper.Show()
			} else {
				dowSelect.Show()
				domWrapper.Hide()
			}
		}
	}

	saveAll := func() {
		sortRecurringMeetings(meetings)
		newCfg := cfg
		newCfg.RecurringMeetings = meetings
		if err := writeConfig(newCfg); err != nil {
			dialog.ShowError(err, parent)
		}
	}

	editingIndex := -1
	var addBtn *widget.Button
	cancelEditBtn := widget.NewButton("Cancel", nil)
	cancelEditBtn.Hide()
	resetForm := func() {
		editingIndex = -1
		tagEntry.SetText("")
		cadenceSelect.SetSelected("weekly")
		dowSelect.SetSelected(dowNames[time.Monday])
		weekendSelect.SetSelected(weekendPolicyOptions[0])
		timeEntry.SetText("")
		domEntry.SetText("")
		cadenceSelect.OnChanged("weekly")
		addBtn.SetText("Add")
		cancelEditBtn.Hide()
	}
	beginMeetingEdit = func(index int) {
		if index < 0 || index >= len(meetings) {
			return
		}
		m := meetings[index]
		editingIndex = index
		tagEntry.SetText(m.Tag)
		cadenceSelect.SetSelected(m.Cadence)
		if m.DOW >= 0 && m.DOW < len(dowNames) {
			dowSelect.SetSelected(dowNames[m.DOW])
		}
		weekendSelect.SetSelected(weekendPolicyFor(m.WeekendPolicy))
		timeEntry.SetText(m.Time)
		domEntry.SetText(strconv.Itoa(m.DayOfMonth))
		cadenceSelect.OnChanged(m.Cadence)
		addBtn.SetText("Save")
		cancelEditBtn.Show()
	}
	var addItem func()
	addBtn = widget.NewButton("Add", func() {
		addItem()
	})
	addItem = func() {
		tag := normalizeTag(tagEntry.Text)
		if tag == "" {
			dialog.ShowError(errors.New("tag is required"), parent)
			return
		}
		if !hmPattern.MatchString(timeEntry.Text) {
			dialog.ShowError(errors.New("time must be HH:MM (24-hour)"), parent)
			return
		}
		var meeting RecurringMeeting
		if cadenceSelect.Selected == "daily" {
			weekendPolicy := ""
			if weekendSelect.Selected == weekendPolicyOptions[1] {
				weekendPolicy = "skip"
			}
			meeting = RecurringMeeting{
				Tag:           tag,
				Cadence:       "daily",
				Time:          timeEntry.Text,
				WeekendPolicy: weekendPolicy,
			}
		} else if cadenceSelect.Selected == "monthly" || cadenceSelect.Selected == "quarterly" {
			day, err := strconv.Atoi(strings.TrimSpace(domEntry.Text))
			if err != nil || day < 1 || day > 31 {
				dialog.ShowError(errors.New("day of month must be 1-31"), parent)
				return
			}
			meeting = RecurringMeeting{Tag: tag, Cadence: cadenceSelect.Selected, Time: timeEntry.Text, DayOfMonth: day, AnchorDate: time.Now().Format("2006-01-02")}
		} else {
			dow := 0
			for i, n := range dowNames {
				if n == dowSelect.Selected {
					dow = i
				}
			}
			meeting = RecurringMeeting{Tag: tag, Cadence: cadenceSelect.Selected, DOW: dow, Time: timeEntry.Text, IntervalWeeks: 1, AnchorDate: time.Now().Format("2006-01-02")}
		}
		if editingIndex >= 0 && editingIndex < len(meetings) {
			meeting.AnchorDate = meetings[editingIndex].AnchorDate
			meetings[editingIndex] = meeting
		} else {
			meetings = append(meetings, meeting)
		}
		saveAll()
		refreshMeetings()
		resetForm()
	}
	cancelEditBtn.OnTapped = resetForm
	tagEntry.OnSubmitted = func(string) { addItem() }
	timeEntry.OnSubmitted = func(string) { addItem() }
	domEntry.OnSubmitted = func(string) { addItem() }
	refreshMeetings = func() {
		sortRecurringMeetings(meetings)
		meetingsBox.RemoveAll()
		if len(meetings) == 0 {
			meetingsBox.Add(widget.NewLabel("No recurring meetings yet."))
		}
		for i, m := range meetings {
			i := i
			editBtn := widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() { beginMeetingEdit(i) })
			deleteBtn := widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDelete), func() {
				meetings = append(meetings[:i], meetings[i+1:]...)
				saveAll()
				refreshMeetings()
			})
			meetingsBox.Add(container.NewBorder(nil, nil, nil, container.NewHBox(editBtn, deleteBtn), recurringEntryLabel(m.Tag, recurringMeetingDetail(m))))
		}
		meetingsBox.Refresh()
	}
	refreshMeetings()

	heading := widget.NewLabelWithStyle("🗓️ Recurring Meetings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	helpLine := widget.NewLabelWithStyle("📝 Use these tags throughout your weeks any time a meeting topic thought comes to mind. They’ll be collected and presented to you just before your meeting starts. And summaries will be shown after.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	helpLine.Wrapping = fyne.TextWrapWord
	helpLine.SizeName = theme.SizeNameCaptionText
	actionsRow := container.NewHBox(cadenceSelect, dowSelect, domWrapper, timeWrapper, weekendSelect, addBtn, cancelEditBtn)
	meetingsScroll := container.NewVScroll(meetingsBox)
	meetingsScroll.SetMinSize(fyne.NewSize(0, 170))

	content := container.NewVBox(
		heading,
		helpLine,
		tagEntry,
		tagSuggestions,
		actionsRow,
		container.NewPadded(widget.NewSeparator()),
		meetingsScroll,
	)

	w := a.NewWindow("Dunnit: Recurring Meetings")
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(520, 440))
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
	if m.Cadence == "monthly" || m.Cadence == "quarterly" {
		monthStep := 1
		if m.Cadence == "quarterly" {
			monthStep = 3
		}
		month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		if m.Cadence == "quarterly" {
			if anchor, err := time.ParseInLocation("2006-01-02", m.AnchorDate, now.Location()); err == nil {
				months := (month.Year()-anchor.Year())*12 + int(month.Month()-anchor.Month())
				if months < 0 {
					month = time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, now.Location())
					months = 0
				}
				if remainder := months % monthStep; remainder != 0 {
					month = month.AddDate(0, monthStep-remainder, 0)
				}
			}
		}
		day := clampDayOfMonth(m.DayOfMonth, month)
		candidate := time.Date(month.Year(), month.Month(), day, hh, mm, 0, 0, now.Location())
		if !candidate.After(now) {
			nextMonth := candidate.AddDate(0, monthStep, 0)
			day = clampDayOfMonth(m.DayOfMonth, time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location()))
			candidate = time.Date(nextMonth.Year(), nextMonth.Month(), day, hh, mm, 0, 0, now.Location())
		}
		return candidate
	}

	interval := m.IntervalWeeks
	if interval <= 0 {
		interval = 1
	}

	dow := m.DOW
	if dow < 0 || dow >= len(dowNames) {
		dow = int(time.Monday)
	}
	daysAhead := (dow - int(now.Weekday()) + 7) % 7
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location()).AddDate(0, 0, daysAhead)
	if candidate.Before(now) {
		candidate = candidate.AddDate(0, 0, 7)
	}

	if m.Cadence == "biweekly-odd" || m.Cadence == "biweekly-even" {
		for {
			_, week := candidate.ISOWeek()
			odd := week%2 == 1
			if (m.Cadence == "biweekly-odd" && odd) || (m.Cadence == "biweekly-even" && !odd) {
				return candidate
			}
			candidate = candidate.AddDate(0, 0, 7)
		}
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

// meetingOccursOnDate reports whether a recurring meeting occurs on date.
// It is used by lastOccurrence so all supported cadence types share the
// same after-meeting timing rules.
func meetingOccursOnDate(m RecurringMeeting, date time.Time) bool {
	switch m.Cadence {
	case "daily":
		return m.WeekendPolicy != "skip" || (date.Weekday() != time.Saturday && date.Weekday() != time.Sunday)
	case "monthly":
		return date.Day() == clampDayOfMonth(m.DayOfMonth, date)
	case "quarterly":
		if date.Day() != clampDayOfMonth(m.DayOfMonth, date) {
			return false
		}
		anchor, err := time.ParseInLocation("2006-01-02", m.AnchorDate, date.Location())
		if err != nil {
			return int(date.Month())%3 == 1
		}
		months := (date.Year()-anchor.Year())*12 + int(date.Month()-anchor.Month())
		return months >= 0 && months%3 == 0
	default:
		if int(date.Weekday()) != m.DOW {
			return false
		}
		if m.Cadence == "biweekly-odd" || m.Cadence == "biweekly-even" {
			_, week := date.ISOWeek()
			return (m.Cadence == "biweekly-odd" && week%2 == 1) || (m.Cadence == "biweekly-even" && week%2 == 0)
		}
		interval := m.IntervalWeeks
		if interval <= 0 {
			interval = 1
		}
		if interval == 1 {
			return true
		}
		anchor, err := time.ParseInLocation("2006-01-02", m.AnchorDate, date.Location())
		if err != nil {
			return true
		}
		dateUTC := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		anchorUTC := time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, time.UTC)
		days := int(dateUTC.Sub(anchorUTC).Hours() / 24)
		return days >= 0 && days%(7*interval) == 0
	}
}

// lastOccurrence returns the most recent past occurrence of m at or
// before now (the mirror of nextOccurrence). Used by the post-meeting
// nudge, which looks backward instead of forward.
func lastOccurrence(m RecurringMeeting, now time.Time) time.Time {
	hour, minute := parseHM(m.Time)
	for daysBack := 0; daysBack <= 3660; daysBack++ {
		date := now.AddDate(0, 0, -daysBack)
		if !meetingOccursOnDate(m, date) {
			continue
		}
		candidate := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, now.Location())
		if !candidate.After(now) {
			return candidate
		}
	}
	return time.Time{}
}

// dueForPostMeetingNudge reports whether the last meeting occurrence is in
// the 15-45 minute post-meeting window. Duration is not tracked, so this
// window is the practical approximation for a meeting summary prompt.
func dueForPostMeetingNudge(m RecurringMeeting, now time.Time) bool {
	last := lastOccurrence(m, now)
	if last.IsZero() {
		return false
	}
	delta := now.Sub(last)
	return delta > 15*time.Minute && delta <= 45*time.Minute
}
