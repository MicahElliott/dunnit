package dun

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds user-configurable Dunnit preferences, loaded from a TOML
// file at DunnitDir()/config.toml. Ported from the original dunnit zsh
// config-example.zsh (minus dunnits_dir, which is now just DunnitDir()
// itself -- everything dunnit owns lives under one root directory).
type Config struct {
	// DunnitDir is the directory containing ledgers and related data. The
	// DUNNIT_DIR environment variable still takes precedence.
	DunnitDir string `toml:"dunnit_dir"`

	// GitSyncEnabled exposes the optional system-git Push/Pull menu items.
	GitSyncEnabled bool `toml:"git_sync_enabled"`

	// DayStart/DayEnd mark roughly when your working day runs, as
	// "HH:MM" 24-hour strings. Used to decide whether hourly popups
	// should fire at all.
	DayStart string `toml:"day_start"`
	DayEnd   string `toml:"day_end"`

	// HourlyMinute is the minute of every hour when the popup should
	// appear (e.g. 58 means it pops at :58 past each hour).
	// Deprecated: replaced by NudgeIntervalMinutes (FR-04). Kept only
	// so old config.toml files with this key don't fail to decode;
	// no longer read by the scheduler.
	HourlyMinute int `toml:"hourly_minute"`

	// NudgeIntervalMinutes is how often (in minutes) the capture
	// nudge fires during work hours, e.g. 30/45/60/90.
	NudgeIntervalMinutes int `toml:"nudge_interval_minutes"`

	// LunchTime is "HH:MM" for a midday goals-reminder popup.
	LunchTime string `toml:"lunch_time"`

	// RecurringMeetings is the FR-15 mini-calendar: a small,
	// purely user-entered list of weekly recurring meeting slots
	// (tag + day-of-week + time), used by FR-16's pre-meeting
	// nudge. No real calendar/.ics/EventKit integration.
	RecurringMeetings []RecurringMeeting `toml:"recurring_meeting"`

	// WeeklyDigestDay/Time (FR-19) configure when the proactive
	// weekly digest nudge fires, e.g. "Friday" "16:00". Empty
	// DigestDay disables the nudge (default: disabled, since it
	// shells out to gh copilot and Micah may not want it firing
	// unprompted until explicitly configured).
	WeeklyDigestDay  string `toml:"weekly_digest_day"`
	WeeklyDigestTime string `toml:"weekly_digest_time"`

	// AutoDraftDailySummary (FR-18) gates whether EOD's Finalize Day
	// automatically drafts/opens the daily summary doc via gh
	// copilot. Default false -- open design questions remain about
	// whether EOD is even the right trigger timing, and what should
	// differentiate the doc's content from Summarize's existing Day
	// output (see FR-18 questions in
	// docs/open-design-questions.md). Manual drafting via the "Daily
	// Summary Doc..." tray item is unaffected by this flag.
	AutoDraftDailySummary bool `toml:"auto_draft_daily_summary"`

	// SnoozeMinutes (FR-26) is the default duration used by the
	// "Snooze" action (both Daybook's button and the tray menu's
	// default top-level item). Defaults to 15 if unset/invalid. The
	// tray menu also offers a few fixed alternatives (15/30/60) via a
	// submenu regardless of this setting.
	SnoozeMinutes int `toml:"snooze_minutes"`

	// DoNotDisturb (FR-27) is a manual on/off flag the user toggles
	// from the tray menu -- while true, the periodic capture nudge
	// (sched.go) is suppressed entirely, same as an indefinite
	// Snooze. Kept simple deliberately: no OS-level DND/screen-share
	// detection (there's no stable public API for that on macOS, and
	// it was decided not to build a heuristic file-based guess
	// either) -- just a manual toggle the user flips themselves.
	// Persisted so it survives app restarts. Only gates the periodic
	// nudge, not SOD/EOD/meeting nudges, same scope as Snooze.
	DoNotDisturb bool `toml:"do_not_disturb"`

	// RecurringItems is the recurring-TODO/GOAL feature (see
	// RECURRING-ITEMS-DESIGN-SEED.md): a small, hand-maintained list
	// of items to be suggested (not auto-seeded) on a daily/weekly/
	// monthly cadence. Daily/weekly are surfaced in SOD; monthly in
	// SOM. Managed via showRecurringItemsDialog (recurring.go).
	RecurringItems []RecurringItem `toml:"recurring_item"`

	// SkipUSFederalHolidays, when true, treats the 11 US federal
	// holidays (see holidays.go's isUSFederalHoliday) the same as a
	// weekend day -- no hourly/lunch/SOD/EOD nudges. Default false
	// (opt-in), toggled via Settings.
	SkipUSFederalHolidays bool `toml:"skip_us_federal_holidays"`

	// KickoffEnabled/ReviewEnabled gate each of the 5 units'
	// Kickoff/Review surfaces independently (see
	// docs/kickoff-review-design.md) -- the primary lever for "5
	// units is a lot to ask": a unit's toggle off means it's never
	// shown, automatically or via the Kickoff.../Review... tray
	// submenus. Day/Week/Month default on (they cover today's
	// existing SOD/EOD/SOM); Quarter/Year default off since they're
	// new, no-prior-art surfaces a user should opt into.
	KickoffDayEnabled     bool `toml:"kickoff_day_enabled"`
	KickoffWeekEnabled    bool `toml:"kickoff_week_enabled"`
	KickoffMonthEnabled   bool `toml:"kickoff_month_enabled"`
	KickoffQuarterEnabled bool `toml:"kickoff_quarter_enabled"`
	KickoffYearEnabled    bool `toml:"kickoff_year_enabled"`

	ReviewDayEnabled     bool `toml:"review_day_enabled"`
	ReviewWeekEnabled    bool `toml:"review_week_enabled"`
	ReviewMonthEnabled   bool `toml:"review_month_enabled"`
	ReviewQuarterEnabled bool `toml:"review_quarter_enabled"`
	ReviewYearEnabled    bool `toml:"review_year_enabled"`

	// ThemeDay/Week/Month/Quarter/Year hold each unit's standing
	// default Review theme (one of the Theme* constants in
	// period.go) -- individual Review invocations may still override
	// this just for that one instance via a dropdown on the dialog,
	// without changing this stored default.
	ThemeDay     string `toml:"theme_day"`
	ThemeWeek    string `toml:"theme_week"`
	ThemeMonth   string `toml:"theme_month"`
	ThemeQuarter string `toml:"theme_quarter"`
	ThemeYear    string `toml:"theme_year"`

	// ExtendWorkWeekTo7Days, when true, shows the full Mon-Sun 7-day
	// span in Week Kickoff/Review labels instead of the default
	// Mon-Fri 5-day work week. Default false (5-day), toggled via
	// Settings. Display-only -- Week's actual data-gathering range
	// (periodNominalRange/periodDataRange for periodWeek) always
	// covers the full Mon-Sun week regardless of this setting, so a
	// weekend ledger entry is never silently excluded from a Week
	// Review just because the label says "Mon-Fri".
	ExtendWorkWeekTo7Days bool `toml:"extend_work_week_to_7_days"`

	// EnableOKRs gates the Objective/Key-Result modules added to
	// Quarter/Year Kickoff (goal entry) and Review (status scoring) --
	// see docs/kickoff-review-design.md's OKR design. Default false
	// (opt-in): someone who doesn't do formal OKR-style planning
	// never sees these extra sections.
	EnableOKRs bool `toml:"enable_okrs"`

	// FavoriteCategories is a user-chosen list of category codes
	// forming an additional "Faves" quick-filter bucket in Daybook's
	// category picker, alongside the fixed Now/Plan/Reflect groups
	// (see categories.go's Group field) -- unlike those, Faves is
	// entirely user-defined and can mix codes from any group. When
	// non-empty, Faves is the picker's default-active filter shown
	// each time Daybook pops up (replacing "whatever group was last
	// used"), rather than just another option to pick. Default seed:
	// DONE/TODO/IDEA/FIXME/MEETING (Micah's stated preference,
	// 2026-09-02) -- edit via Settings to change.
	FavoriteCategories []string `toml:"favorite_categories"`

	// ReportExcludeTags is a list of "#tag" strings; any ledger line
	// containing one of these tags is excluded from every report/
	// summary generation pipeline (Kickoff/Review digests, Standup,
	// Status Report, Annual Review, Trend View, etc) -- the goal is
	// keeping non-work items (personal errands, etc) out of work-
	// facing reports without needing to keep them out of the ledger
	// itself. Default seed: #home/#personal/#buy/#shop (Micah's
	// stated preference, 2026-09-02) -- edit via Settings to change.
	// Entries are matched as exact #tag tokens (see extractTags),
	// case-sensitive, same as tags are written/matched everywhere
	// else in this codebase.
	ReportExcludeTags []string `toml:"report_exclude_tags"`

	// WastedTimeTrackingEnabled, when true, offers the WASTED
	// category in Daybook's live picker (End/All group filters,
	// Faves). Default false: WASTED is an opt-in "track time you feel
	// was wasted" feature some users won't want surfaced at all. When
	// false, WASTED is still present in Categories (so Help/legend
	// text and historical ledger entries still resolve/display
	// correctly) -- only excluded from picker option lists, same
	// mechanical pattern EODOnly already uses for SUMMARY/
	// PRODUCTIVITY/MEETING_HOURS (see CategoryLabelsForGroup/
	// CategoryLabelsForFaves in categories.go).
	WastedTimeTrackingEnabled bool `toml:"wasted_time_tracking_enabled"`

	// LastCarryForwardDate is "YYYY-MM-DD", the last calendar date
	// on which unresolved open items (TODO/GOAL/WAITING/QUESTION/
	// FIXME/RISK) were copied forward into that day's ledger (see
	// docs/todo-carryforward-design.md). Same pattern as
	// RecurringMeeting's lastOccurrence tracking, just persisted here
	// instead of derived from ledger content -- deliberately NOT
	// inferred from "is today's ledger file empty", since a user who
	// logs an entry before carry-forward has run would make that
	// check wrong. Empty string means carry-forward has never run.
	LastCarryForwardDate string `toml:"last_carry_forward_date"`
}

// defaultConfig mirrors the values from dunnit's config-example.zsh.
func defaultConfig() Config {
	return Config{
		DayStart:             "08:00",
		DayEnd:               "17:30",
		NudgeIntervalMinutes: 60,
		LunchTime:            "11:30",
		SnoozeMinutes:        15,

		// Day/Week/Month default on (they cover today's existing
		// SOD/EOD/SOM); Quarter/Year default off, opt-in, since
		// they're new surfaces with no prior art (see
		// docs/kickoff-review-design.md).
		KickoffDayEnabled:     true,
		KickoffWeekEnabled:    true,
		KickoffMonthEnabled:   true,
		KickoffQuarterEnabled: false,
		KickoffYearEnabled:    false,

		ReviewDayEnabled:     true,
		ReviewWeekEnabled:    true,
		ReviewMonthEnabled:   true,
		ReviewQuarterEnabled: false,
		ReviewYearEnabled:    false,

		ThemeDay:     ThemePersonalNotes,
		ThemeWeek:    ThemePersonalNotes,
		ThemeMonth:   ThemeStatusReport,
		ThemeQuarter: ThemeFormalReport,
		ThemeYear:    ThemeFormalReport,

		FavoriteCategories: []string{"DONE", "TODO", "IDEA", "FIXME", "MEETING"},
		ReportExcludeTags:  []string{"#home", "#personal", "#buy", "#shop"},

		WastedTimeTrackingEnabled: false,
	}
}

// DunnitDir is the single root directory for everything dunnit owns:
// ledger files (DunnitDir()/<year>/<month>/w<week>/ledger-*.txt) and
// config.toml. Overridable via the DUNNIT_DIR env var; defaults to
// ~/.config/dunnit.
func DunnitDir() string {
	if dir := os.Getenv("DUNNIT_DIR"); dir != "" {
		return dir
	}
	if dir := configuredDunnitDir; dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dunnit")
}

var configuredDunnitDir string

func configPath() string {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".config", "dunnit")
	if dir := os.Getenv("DUNNIT_DIR"); dir != "" {
		root = dir
	}
	return filepath.Join(root, "config.toml")
}

// LoadConfig reads config.toml, creating it with defaults on first run
// if it doesn't exist yet.
func LoadConfig() Config {
	cfg := defaultConfig()
	path := configPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(DunnitDir(), 0755); err != nil {
			log.Println("Error creating dunnit dir:", err)
			return cfg
		}
		if err := writeConfig(cfg); err != nil {
			log.Println("Error writing default config:", err)
		}
		return cfg
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		log.Println("Error reading config, using defaults:", err)
		return defaultConfig()
	}
	if os.Getenv("DUNNIT_DIR") == "" && cfg.DunnitDir != "" {
		configuredDunnitDir = cfg.DunnitDir
	}
	return cfg
}

func writeConfig(cfg Config) error {
	f, err := os.Create(configPath())
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
