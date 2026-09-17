package dun

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"github.com/go-co-op/gocron/v2"
)

// parseHM parses "HH:MM" into hour, minute ints. Returns zeros on
// parse failure (caller should treat that as "not configured").
func parseHM(s string) (hour, minute int) {
	fmt.Sscanf(s, "%d:%d", &hour, &minute)
	return
}

// isFirstWeekdayOfMonth reports whether the given date is the first
// weekday (Mon-Fri) of its month -- i.e. day 1, or day 2/3 if day 1
// falls on a weekend. Used to auto-pop the SOM wizard on the actual
// first working day of the month, rather than requiring day 1 itself
// to be a weekday (which would otherwise silently skip SOM entirely
// for any month starting on a Sat/Sun).
func isFirstWeekdayOfMonth(now time.Time) bool {
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	for first.Weekday() == time.Saturday || first.Weekday() == time.Sunday {
		first = first.AddDate(0, 0, 1)
	}
	return now.Day() == first.Day()
}

// isOffDay reports whether now is a day Dunnit's nudges should be
// entirely suppressed: a weekend, or (if cfg.SkipUSFederalHolidays is
// enabled) a US federal holiday, treated identically to a weekend.
func isOffDay(cfg Config, now time.Time) bool {
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return true
	}
	return cfg.SkipUSFederalHolidays && isUSFederalHoliday(now)
}

// withinWorkHours reports whether now falls between the configured
// day_start and day_end (inclusive), Mon-Fri only (see isOffDay --
// also excludes US federal holidays if that setting is enabled).
func withinWorkHours(cfg Config, now time.Time) bool {
	if isOffDay(cfg, now) {
		return false
	}
	startH, startM := parseHM(cfg.DayStart)
	endH, endM := parseHM(cfg.DayEnd)
	start := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, now.Location())
	end := time.Date(now.Year(), now.Month(), now.Day(), endH, endM, 0, 0, now.Location())
	return !now.Before(start) && !now.After(end)
}

// sendAutoPopupNotification gives every scheduled popup the same OS-level
// application identity. The app icon is installed before Schedule runs in
// main, and Fyne passes that icon to each platform notification backend.
func sendAutoPopupNotification(a fyne.App, message string) {
	a.SendNotification(fyne.NewNotification("Dunnit", message))
}

// Schedule sets up the recurring popups (hourly activity prompt, and a
// lunchtime goals reminder), reading times from config.toml. It shows
// (raises) the given main window rather than just sending a passive
// notification, since the whole point is to prompt for input.
func Schedule(a fyne.App, w fyne.Window) gocron.Scheduler {
	cfg := LoadConfig()

	s, err := gocron.NewScheduler()
	if err != nil {
		fmt.Println("Error creating scheduler:", err)
		return s
	}

	// intervalDuration is how often this job fires, from
	// cfg.NudgeIntervalMinutes (FR-04; falls back to 60 if unset/
	// invalid, e.g. an old config.toml predating this key). FR-01: if
	// the user already logged an entry more recently than this
	// interval, skip the nudge -- they're clearly already engaged, no
	// need to interrupt.
	intervalMinutes := nudgeIntervalMinutes(cfg)
	intervalDuration := time.Duration(intervalMinutes) * time.Minute

	_, err = s.NewJob(
		gocron.DurationJob(intervalDuration),
		gocron.NewTask(func() {
			now := time.Now()
			if !withinWorkHours(cfg, now) {
				return
			}
			if last := LastActivityAt(); !last.IsZero() && now.Sub(last) < intervalDuration {
				return
			}
			if !SnoozedUntil().IsZero() {
				return
			}
			if IsDoNotDisturb() {
				return
			}
			sendAutoPopupNotification(a, "What are you working on?")
			fyne.Do(func() {
				ShowDaybook(w, true)
			})
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling interval job:", err)
	}

	if lh, lm := parseHM(cfg.LunchTime); lh != 0 || lm != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(lh), uint(lm), 0))),
			gocron.NewTask(func() {
				if !withinWorkHours(cfg, time.Now()) {
					return
				}
				sendAutoPopupNotification(a, "How are your goals coming along?")
				fyne.Do(func() {
					ShowDaybook(w, true)
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling lunchtime job:", err)
		}
	}

	// FR-13: Start-of-Day nudge, fires once per workday near
	// cfg.DayStart, showing today's open TODOs/DOING/GOALs (readback) and a
	// chance to add more before the day gets going. FR-14: if today
	// is also the first weekday of the month, show the SOM wizard
	// instead (its step 4 already covers the same "current GOALs"
	// readback SOD would show, so no need for both). Uses
	// isFirstWeekdayOfMonth rather than a plain "day == 1" check so
	// SOM still fires on the actual first working day even when the
	// 1st falls on a weekend (otherwise SOM would silently never fire
	// that month).
	if sh, sm := parseHM(cfg.DayStart); sh != 0 || sm != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(sh), uint(sm), 0))),
			gocron.NewTask(func() {
				now := time.Now()
				cfg := LoadConfig()
				if isOffDay(cfg, now) {
					return
				}
				if isFirstWeekdayOfMonth(now) {
					sendAutoPopupNotification(a, "A new month is ready for review and planning.")
					fyne.Do(func() {
						if startOfDayPending(cfg, now) {
							showSODWindow(a)
						}
						showMonthReviewWindow(a, periodOffsetAnchor(periodMonth, now, -1))
						showMonthKickoffWindow(a, now)
					})
					return
				}
				if !startOfDayPending(cfg, now) {
					return
				}
				sendAutoPopupNotification(a, "Good morning! Here’s where things stand.")
				fyne.Do(func() {
					showSODWindow(a)
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling start-of-day job:", err)
		}
	}

	if eh, em := parseHM(cfg.DayEnd); eh != 0 || em != 0 {
		_, err = s.NewJob(
			gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(uint(eh), uint(em), 0))),
			gocron.NewTask(func() {
				now := time.Now()
				if isOffDay(LoadConfig(), now) || endOfDayAlreadyRun(now) {
					return
				}
				sendAutoPopupNotification(a, "End of day! Let’s wrap up.")
				fyne.Do(func() {
					showEODWindow(a)
				})
			}),
		)
		if err != nil {
			fmt.Println("Error scheduling end-of-day job:", err)
		}
	}

	// FR-16: pre-meeting nudge, checking every 15 min (per Micah's
	// call -- simplest fixed interval, not tied to NudgeIntervalMinutes)
	// whether any FR-15 recurring meeting's next occurrence starts
	// within the next ~15 min. firedFor dedupes so the same occurrence
	// doesn't nudge repeatedly across multiple 15-min checks while
	// still inside the window. A "#dsu" tag (FR-17) triggers the
	// deterministic standup export instead of the generic Meeting
	// Prep dialog; every other tag still gets Meeting Prep (FR-12).
	// Post-Meeting Capture is deliberately a separate after-start check
	// below so a meeting summary never opens at the same time as prep.
	firedFor := map[string]time.Time{} // occurrence key -> occurrence time already nudged for
	firedPostFor := map[string]time.Time{}
	_, err = s.NewJob(
		gocron.DurationJob(15*time.Minute),
		gocron.NewTask(func() {
			now := time.Now()
			cfg := LoadConfig()
			for i, m := range cfg.RecurringMeetings {
				if dueForPreMeetingNudge(m, now, 15*time.Minute) {
					occ := nextOccurrence(m, now)
					key := fmt.Sprintf("%d:%s", i, occ.Format(time.RFC3339))
					if fired, ok := firedFor[key]; !ok || !fired.Equal(occ) {
						firedFor[key] = occ
						sendAutoPopupNotification(a, "Upcoming meeting "+m.Tag+" at "+m.Time)
						m := m
						fyne.Do(func() {
							if strings.EqualFold(m.Tag, "#dsu") {
								showStandupExport(a)
							} else {
								showMeetingPrepDialogForTag(a, m.Tag)
							}
						})
					}
				}
				if dueForPostMeetingNudge(m, now) {
					occ := lastOccurrence(m, now)
					key := fmt.Sprintf("%d:%s", i, occ.Format(time.RFC3339))
					if fired, ok := firedPostFor[key]; !ok || !fired.Equal(occ) {
						firedPostFor[key] = occ
						sendAutoPopupNotification(a, "Meeting summary for "+m.Tag)
						m := m
						fyne.Do(func() { showPostMeetingCapture(a, m.Tag) })
					}
				}
			}
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling pre-meeting nudge job:", err)
	}

	// Timed recurring items use the native notification channel and raise
	// Daybook with the item ready to review. Untimed items retain their SOD/
	// SOM suggestion behavior. A two-minute tolerance absorbs scheduler
	// drift while the occurrence key prevents duplicate alerts.
	firedRecurringItems := map[string]time.Time{}
	_, err = s.NewJob(
		gocron.DurationJob(time.Minute),
		gocron.NewTask(func() {
			now := time.Now()
			cfg := LoadConfig()
			for i, item := range cfg.RecurringItems {
				if !dueForRecurringItemReminder(item, now, 2*time.Minute) {
					continue
				}
				occ, ok := recurringItemOccurrence(item, now)
				if !ok {
					continue
				}
				key := fmt.Sprintf("%d:%s", i, occ.Format(time.RFC3339))
				if fired, ok := firedRecurringItems[key]; ok && fired.Equal(occ) {
					continue
				}
				firedRecurringItems[key] = occ
				sendAutoPopupNotification(a, "Recurring item: "+item.Category+" "+item.Text)
				item := item
				fyne.Do(func() { ShowRecurringItemReminder(w, item) })
			}
		}),
	)
	if err != nil {
		fmt.Println("Error scheduling recurring item reminder job:", err)
	}

	// FR-19: proactive weekly digest, fires once on the configured
	// weekly_digest_day/time (e.g. Friday 16:00) and shows a Week-
	// period Summarize report unprompted. Disabled by default (no
	// weekly_digest_day configured) since it shells out to configured LLM CLI
	// on a schedule -- opt-in via Settings/config.toml. The monthly
	// version is intentionally not a separate mechanism here; it's
	// folded into FR-14's SOM wizard once that exists.
	if wd, ok := parseWeekday(cfg.WeeklyDigestDay); ok {
		if dh, dm := parseHM(cfg.WeeklyDigestTime); dh != 0 || dm != 0 {
			_, err = s.NewJob(
				gocron.WeeklyJob(1, gocron.NewWeekdays(wd), gocron.NewAtTimes(gocron.NewAtTime(uint(dh), uint(dm), 0))),
				gocron.NewTask(func() {
					sendAutoPopupNotification(a, "Your weekly digest is ready.")
					fyne.Do(func() {
						w.Show()
						runSummarize(a, periodWeek)
					})
				}),
			)
			if err != nil {
				fmt.Println("Error scheduling weekly digest job:", err)
			}
		}
	}

	s.Start()
	return s
}

// parseWeekday parses a day-name string ("Monday".."Sunday", case-
// insensitive) into a time.Weekday. ok is false for an empty or
// unrecognized string (treated as "digest not configured").
func parseWeekday(s string) (day time.Weekday, ok bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "sunday":
		return time.Sunday, true
	case "monday":
		return time.Monday, true
	case "tuesday":
		return time.Tuesday, true
	case "wednesday":
		return time.Wednesday, true
	case "thursday":
		return time.Thursday, true
	case "friday":
		return time.Friday, true
	case "saturday":
		return time.Saturday, true
	default:
		return 0, false
	}
}
