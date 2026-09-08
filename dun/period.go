package dun

import (
	"fmt"
	"strings"
	"time"
)

// periodConfig holds the small set of things that differ per unit
// for the Kickoff/Review system (see
// docs/kickoff-review-design.md): which unit rolls up into it
// (SubPeriod, empty if none), and its default Review theme.
// Toggle/theme *settings* live in Config (config.go); this struct is
// static, code-level metadata about each unit's shape, not
// user-editable state.
type periodConfig struct {
	Period       summaryPeriod
	SubPeriod    summaryPeriod // "" if none (Day, Week have no sub-tier)
	DefaultTheme string
}

// periodConfigs is the static table of the 5 units' shape, keyed by
// summaryPeriod. Quarters are traditional calendar quarters
// (Jan-Mar/Apr-Jun/Jul-Sep/Oct-Dec); years are Jan 1 - Dec 31 -- no
// fiscal-year config, at least for now.
var periodConfigs = map[summaryPeriod]periodConfig{
	periodDay:     {Period: periodDay, DefaultTheme: ThemePersonalNotes},
	periodWeek:    {Period: periodWeek, DefaultTheme: ThemePersonalNotes},
	periodMonth:   {Period: periodMonth, SubPeriod: periodWeek, DefaultTheme: ThemeStatusReport},
	periodQuarter: {Period: periodQuarter, SubPeriod: periodMonth, DefaultTheme: ThemeFormalReport},
	periodYear:    {Period: periodYear, SubPeriod: periodQuarter, DefaultTheme: ThemeFormalReport},
}

// Theme names for Review generation (docs/kickoff-review-design.md).
// Kept as plain string constants (rather than a typed enum) since
// they're stored directly as Config TOML values and compared against
// user-editable strings.
const (
	ThemePersonalNotes = "personal_notes"
	ThemeStatusReport  = "status_report"
	ThemeFormalReport  = "formal_report"
	ThemeBragPreso     = "brag_preso"
)

// themeDisplayNames maps each Theme* constant to its human-friendly
// label for dropdowns/menus (e.g. "Personal Notes" instead of the raw
// "personal_notes" TOML value). themeDisplayOrder is the fixed
// presentation order for building a themeSelect-style dropdown.
var themeDisplayNames = map[string]string{
	ThemePersonalNotes: "Personal Notes",
	ThemeStatusReport:  "Status Report",
	ThemeFormalReport:  "Formal Report",
	ThemeBragPreso:     "Brag Preso",
}

var themeDisplayOrder = []string{ThemePersonalNotes, ThemeStatusReport, ThemeFormalReport, ThemeBragPreso}

// themeOptions returns the display-name options (in themeDisplayOrder)
// for a theme-picker widget.Select.
func themeOptions() []string {
	opts := make([]string, len(themeDisplayOrder))
	for i, t := range themeDisplayOrder {
		opts[i] = themeDisplayNames[t]
	}
	return opts
}

// themeFromDisplayName reverses themeDisplayNames, e.g. "Status
// Report" -> ThemeStatusReport. Returns "" if display isn't
// recognized.
func themeFromDisplayName(display string) string {
	for _, t := range themeDisplayOrder {
		if themeDisplayNames[t] == display {
			return t
		}
	}
	return ""
}

// quarterOf returns 1-4 for the traditional calendar quarter
// containing t (Jan-Mar=1, Apr-Jun=2, Jul-Sep=3, Oct-Dec=4).
func quarterOf(t time.Time) int {
	return (int(t.Month())-1)/3 + 1
}

// periodLabel returns the human/report-facing name for a period,
// naming the *nominal* calendar unit containing anchor -- always
// independent of whatever partial data-fetch window a Review actually
// managed to pull (see periodDataRange), so titles never read
// something awkward like "covers Apr 1 - Jun 25". cfg is consulted
// for periodWeek's ExtendWorkWeekTo7Days setting (5-day Mon-Fri
// display by default, 7-day Mon-Sun if enabled) -- display-only, see
// Config.ExtendWorkWeekTo7Days's doc comment.
//
// Examples: periodLabel(cfg, periodDay, t) -> "Mon Sep 1, 2026";
// periodLabel(cfg, periodWeek, t) -> "Week of Aug 31 - Sep 4 (W37)"
// (5-day default) or "Week of Aug 31 - Sep 6 (W37)" (7-day, if
// ExtendWorkWeekTo7Days) -- week start is Monday, matching
// time.Time.ISOWeek's convention used elsewhere in this codebase;
// periodLabel(cfg, periodMonth, t) -> "Sep 2026";
// periodLabel(cfg, periodQuarter, t) -> "Q3 2026 (Jul-Sep)";
// periodLabel(cfg, periodYear, t) -> "2026".
func periodLabel(cfg Config, period summaryPeriod, anchor time.Time) string {
	switch period {
	case periodDay:
		return anchor.Format("Mon Jan 2, 2006")
	case periodWeek:
		return weekLabel(anchor, cfg.ExtendWorkWeekTo7Days)
	case periodMonth:
		return anchor.Format("Jan 2006")
	case periodQuarter:
		q := quarterOf(anchor)
		monthNames := [4]string{"Jan\u2013Mar", "Apr\u2013Jun", "Jul\u2013Sep", "Oct\u2013Dec"}
		return fmt.Sprintf("Q%d %d (%s)", q, anchor.Year(), monthNames[q-1])
	case periodYear:
		return fmt.Sprintf("%d", anchor.Year())
	default:
		return string(period)
	}
}

// weekLabel returns "Week of <start> – <end> — W<iso week>" for the
// week containing anchor (en-dash between the date range, em-dash
// before the week number, no parens) -- end is start+4 days (Fri) by
// default, or start+6 days (Sun) if extend7 is true. If start and end
// fall in the same month, end is shown as just the day number (e.g.
// "Sep 7 – 11") rather than repeating the month name; otherwise both
// months are spelled out (e.g. "Aug 31 – Sep 4").
func weekLabel(anchor time.Time, extend7 bool) string {
	start := weekStart(anchor)
	lastDayOffset := 4
	if extend7 {
		lastDayOffset = 6
	}
	end := start.AddDate(0, 0, lastDayOffset)
	_, isoWeek := start.ISOWeek()

	var rangeText string
	if start.Month() == end.Month() {
		rangeText = start.Format("Jan 2") + " \u2013 " + end.Format("2")
	} else {
		rangeText = start.Format("Jan 2") + " \u2013 " + end.Format("Jan 2")
	}
	return fmt.Sprintf("Week of %s \u2014 W%d", rangeText, isoWeek)
}

// weekStart returns the Monday (00:00) of the ISO week containing t,
// matching the week-boundary convention time.Time.ISOWeek already
// uses elsewhere in this codebase (e.g. getLedger's directory
// scheme).
func weekStart(t time.Time) time.Time {
	// time.Weekday: Sunday=0 .. Saturday=6. Convert to ISO
	// (Monday=0..Sunday=6) so Monday is treated as day 0 of the week.
	isoWeekday := (int(t.Weekday()) + 6) % 7
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	return d.AddDate(0, 0, -isoWeekday)
}

// periodOffsetAnchor returns a representative time.Time for the unit
// offset periods away from the one containing anchor -- e.g.
// periodOffsetAnchor(periodMonth, now, -1) returns a time in last
// month, periodOffsetAnchor(periodQuarter, now, 0) returns a time in
// the current quarter. Used by the period-picker preamble (docs/
// kickoff-review-design.md) to let a user choose which of "this
// period (so far)"/"last period"/a short back-list to open a
// Kickoff/Review for, instead of every caller hardcoding "the
// previous period" the way Month Review originally did.
func periodOffsetAnchor(period summaryPeriod, anchor time.Time, offset int) time.Time {
	switch period {
	case periodDay:
		return anchor.AddDate(0, 0, offset)
	case periodWeek:
		return anchor.AddDate(0, 0, offset*7)
	case periodMonth:
		return anchor.AddDate(0, offset, 0)
	case periodQuarter:
		return anchor.AddDate(0, offset*3, 0)
	case periodYear:
		return anchor.AddDate(offset, 0, 0)
	default:
		return anchor
	}
}

// periodNominalRange returns the exact calendar boundaries [from, to]
// of the unit containing anchor -- e.g. for periodQuarter, the first
// instant of the quarter through the last instant of its last day.
// Used both for periodLabel-adjacent bookkeeping and as the basis
// periodDataRange pads outward from.
func periodNominalRange(period summaryPeriod, anchor time.Time) (from, to time.Time) {
	loc := anchor.Location()
	switch period {
	case periodDay:
		from = time.Date(anchor.Year(), anchor.Month(), anchor.Day(), 0, 0, 0, 0, loc)
		to = from.AddDate(0, 0, 1).Add(-time.Second)
	case periodWeek:
		from = weekStart(anchor)
		to = from.AddDate(0, 0, 7).Add(-time.Second)
	case periodMonth:
		from = time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
		to = from.AddDate(0, 1, 0).Add(-time.Second)
	case periodQuarter:
		q := quarterOf(anchor)
		startMonth := time.Month((q-1)*3 + 1)
		from = time.Date(anchor.Year(), startMonth, 1, 0, 0, 0, 0, loc)
		to = from.AddDate(0, 3, 0).Add(-time.Second)
	case periodYear:
		from = time.Date(anchor.Year(), time.January, 1, 0, 0, 0, 0, loc)
		to = from.AddDate(1, 0, 0).Add(-time.Second)
	default:
		from, to = anchor, anchor
	}
	return from, to
}

// periodDataRangePadDays is how many extra days beyond the nominal
// boundary periodDataRange pads its returned window by, in either
// direction. Deliberately loose (see docs/kickoff-review-design.md's
// "Configurable lead-time/window" section) -- duplicate coverage of
// the same days across adjacent reports is an acceptable tradeoff for
// avoiding a coverage gap at a boundary; no need for exact interval
// math anywhere that consumes this.
const periodDataRangePadDays = 1

// periodDataRange returns the [from, to] window to actually pull
// source material from for period's Review of the unit containing
// anchor -- the nominal calendar boundaries, padded loosely by
// periodDataRangePadDays on each side, and clamped so `to` never
// extends past `now` -- letting a Review be run for a period that's
// still in progress (docs/kickoff-review-design.md's "mid-period"
// support) without pulling in dates that haven't happened yet.
// Titles/labels should always use periodLabel/periodProgressSuffix
// instead of this range, since this range is intentionally
// imprecise.
func periodDataRange(period summaryPeriod, anchor time.Time) (from, to time.Time) {
	from, to = periodNominalRange(period, anchor)
	pad := time.Duration(periodDataRangePadDays) * 24 * time.Hour
	from, to = from.Add(-pad), to.Add(pad)
	now := time.Now()
	if to.After(now) {
		to = now
	}
	return from, to
}

// periodIsInProgress reports whether the unit containing anchor
// hasn't fully elapsed yet as of now -- i.e. this is a "so far" look
// at a period still underway, not a completed-period retrospective.
func periodIsInProgress(period summaryPeriod, anchor time.Time) bool {
	_, to := periodNominalRange(period, anchor)
	return to.After(time.Now())
}

// periodProgressSuffix returns a short parenthetical to append to a
// Review/Kickoff window's title/label when the unit is still in
// progress (e.g. "(through Sep 2)"), or "" for a fully-elapsed period
// -- so a mid-period Review is clearly framed as partial rather than
// silently implying the period already ended.
func periodProgressSuffix(period summaryPeriod, anchor time.Time) string {
	if !periodIsInProgress(period, anchor) {
		return ""
	}
	return " (through " + time.Now().Format("Jan 2") + ")"
}

// kickoffEnabled/reviewEnabled/themeFor read a unit's Kickoff/Review
// toggle or theme setting out of cfg, keyed by period -- small
// switch-based accessors so callers work in terms of summaryPeriod
// rather than reaching into Config's individual per-unit fields
// directly.
func kickoffEnabled(cfg Config, period summaryPeriod) bool {
	switch period {
	case periodDay:
		return cfg.KickoffDayEnabled
	case periodWeek:
		return cfg.KickoffWeekEnabled
	case periodMonth:
		return cfg.KickoffMonthEnabled
	case periodQuarter:
		return cfg.KickoffQuarterEnabled
	case periodYear:
		return cfg.KickoffYearEnabled
	default:
		return false
	}
}

func reviewEnabled(cfg Config, period summaryPeriod) bool {
	switch period {
	case periodDay:
		return cfg.ReviewDayEnabled
	case periodWeek:
		return cfg.ReviewWeekEnabled
	case periodMonth:
		return cfg.ReviewMonthEnabled
	case periodQuarter:
		return cfg.ReviewQuarterEnabled
	case periodYear:
		return cfg.ReviewYearEnabled
	default:
		return false
	}
}

// setKickoffEnabled/setReviewEnabled are the mutating counterparts to
// kickoffEnabled/reviewEnabled, used by Settings to write back a
// user's toggle choices per unit.
func setKickoffEnabled(cfg *Config, period summaryPeriod, value bool) {
	switch period {
	case periodDay:
		cfg.KickoffDayEnabled = value
	case periodWeek:
		cfg.KickoffWeekEnabled = value
	case periodMonth:
		cfg.KickoffMonthEnabled = value
	case periodQuarter:
		cfg.KickoffQuarterEnabled = value
	case periodYear:
		cfg.KickoffYearEnabled = value
	}
}

func setReviewEnabled(cfg *Config, period summaryPeriod, value bool) {
	switch period {
	case periodDay:
		cfg.ReviewDayEnabled = value
	case periodWeek:
		cfg.ReviewWeekEnabled = value
	case periodMonth:
		cfg.ReviewMonthEnabled = value
	case periodQuarter:
		cfg.ReviewQuarterEnabled = value
	case periodYear:
		cfg.ReviewYearEnabled = value
	}
}

// themeFor returns cfg's configured default theme for period, falling
// back to periodConfigs' DefaultTheme if unset (e.g. an older
// config.toml written before theme fields existed, decoded to "").
func themeFor(cfg Config, period summaryPeriod) string {
	var theme string
	switch period {
	case periodDay:
		theme = cfg.ThemeDay
	case periodWeek:
		theme = cfg.ThemeWeek
	case periodMonth:
		theme = cfg.ThemeMonth
	case periodQuarter:
		theme = cfg.ThemeQuarter
	case periodYear:
		theme = cfg.ThemeYear
	}
	if theme != "" {
		return theme
	}
	return periodConfigs[period].DefaultTheme
}

// setTheme writes value into cfg's per-unit theme field for period --
// the mutating counterpart to themeFor, used by callers building a
// one-off override Config (e.g. showPeriodReviewWindow's per-instance
// theme dropdown) without touching the persisted config.toml.
func setTheme(cfg *Config, period summaryPeriod, value string) {
	switch period {
	case periodDay:
		cfg.ThemeDay = value
	case periodWeek:
		cfg.ThemeWeek = value
	case periodMonth:
		cfg.ThemeMonth = value
	case periodQuarter:
		cfg.ThemeQuarter = value
	case periodYear:
		cfg.ThemeYear = value
	}
}

// reviewLengthWords is the rough target word-count ceiling to instruct
// the copilot prompt with for each unit's Review, scaled so a Day
// digest stays tight while a Year digest is allowed more room to
// cover a full year's worth of rolled-up material. Deliberately a
// ceiling stated in the prompt itself (not just achieved structurally
// via the rollup in review.go) -- see docs/kickoff-review-design.md's
// "Keeping prompts/outputs small" section: structural rollup and
// explicit prompt instruction are both needed, since rollup alone
// doesn't stop an LLM from padding its own output with prose.
var reviewLengthWords = map[summaryPeriod]int{
	periodDay:     150,
	periodWeek:    250,
	periodMonth:   400,
	periodQuarter: 600,
	periodYear:    800,
}

// reviewLengthConstraint returns a short instruction fragment, scaled
// to period, to append to any Review-generating copilot prompt --
// e.g. summarizeWithCopilotPrompt(someInstructions+reviewLengthConstraint(periodMonth), ...).
// Kept as a standalone appendable fragment (not baked into
// summarizeWithCopilotPrompt itself) so non-Review callers (e.g.
// Standup's summarizeStandupWithCopilot, which already has its own
// "Be concise" framing) aren't forced to adopt it.
func reviewLengthConstraint(period summaryPeriod) string {
	words, ok := reviewLengthWords[period]
	if !ok {
		words = 300
	}
	return fmt.Sprintf(" Keep the total output under roughly %d words; be terse, "+
		"favor bullet points over prose.", words)
}

// themePromptFraming returns the instruction text for a Review's
// copilot prompt, per theme (docs/kickoff-review-design.md's "Three
// themes" section) -- title is the report's periodLabel-derived
// title (e.g. "Sep 2026", "Q3 2026 (Jul-Sep)"), unitNoun is the plain
// English name of the unit being covered (e.g. "day", "week",
// "month", "quarter", "year") for framing the instruction naturally.
// Does not include reviewLengthConstraint -- callers append that
// separately, since it's period-scaled rather than theme-scaled.
func themePromptFraming(theme string, unitNoun, title string) string {
	switch theme {
	case ThemeStatusReport:
		return fmt.Sprintf(
			"Summarize this ledger of a %s's activity entries into a "+
				"status report titled %q, using exactly these section "+
				"headers (Markdown ##): \"What Happened\", \"What's Next\", "+
				"\"Blockers\" (omit Blockers if there are none worth "+
				"mentioning). Neutral, third-person tone, suitable to "+
				"paste into a team update.", unitNoun, title)
	case ThemeFormalReport:
		return fmt.Sprintf(
			"Summarize this ledger of a %s's activity entries into a "+
				"formal report titled %q, using exactly these section "+
				"headers (Markdown ##): \"Summary\", \"Key Accomplishments\", "+
				"\"Challenges\", \"Goals for Next Period\". Sober, "+
				"professional tone, written as if for a manager/reviewer "+
				"audience.", unitNoun, title)
	case ThemeBragPreso:
		return fmt.Sprintf(
			"Summarize this ledger of a %s's activity entries into a "+
				"slide presentation titled %q, formatted as Pandoc-style "+
				"Markdown slides: each slide is a level-1 Markdown heading "+
				"(# Slide Title) followed by a couple of short bullet "+
				"points, with slides separated by a line containing only "+
				"\"---\". Each slide should highlight one achievement or "+
				"notable win, framed for impact. Upbeat tone, minimal text "+
				"per slide.", unitNoun, title)
	default: // ThemePersonalNotes, or unrecognized -- fall back to informal
		return fmt.Sprintf(
			"Summarize this ledger of a %s's activity entries into "+
				"brief personal notes titled %q. Informal, first-person, "+
				"casual tone -- freeform paragraphs or loose bullet points, "+
				"no fixed section headers needed.", unitNoun, title)
	}
}

// unitNoun returns the plain English noun for period ("day", "week",
// "month", "quarter", "year"), for use in prompt framing text.
func unitNoun(period summaryPeriod) string {
	return strings.ToLower(string(period))
}

// generateThemedReview runs the full themed-Review copilot pipeline
// for period's unit containing anchor: gathers rollup source material
// via gatherReviewSourceMaterial (review.go), builds the theme-framed
// prompt (themePromptFraming + reviewLengthConstraint), and returns
// the resulting markdown. Returns a placeholder message (nil error)
// if there's no source material at all to summarize.
func generateThemedReview(cfg Config, period summaryPeriod, anchor time.Time) (string, error) {
	from, to := periodDataRange(period, anchor)
	material := gatherReviewSourceMaterial(period, from, to)

	var combined strings.Builder
	if len(material.SubReports) > 0 {
		combined.WriteString("Prior summaries covering parts of this period:\n\n")
		for _, r := range material.SubReports {
			combined.WriteString(r)
			combined.WriteString("\n\n")
		}
	}
	if strings.TrimSpace(material.RawLedger) != "" {
		combined.WriteString("Additional raw entries not yet summarized:\n\n")
		combined.WriteString(material.RawLedger)
	}
	if okrText := okrSummaryText(cfg, period, anchor); okrText != "" {
		combined.WriteString("\n\nOKR status for this period:\n\n")
		combined.WriteString(okrText)
	}
	if strings.TrimSpace(combined.String()) == "" {
		return fmt.Sprintf("(no entries found for %s)", periodLabel(cfg, period, anchor)), nil
	}

	theme := themeFor(cfg, period)
	title := periodLabel(cfg, period, anchor)
	instructions := themePromptFraming(theme, unitNoun(period), title) + reviewLengthConstraint(period)
	return summarizeWithCopilotPrompt(instructions, combined.String())
}
