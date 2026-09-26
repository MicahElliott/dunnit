# Kickoff / Review Design

Status: design agreed 2026-09-01 (continued from
`SESSION-SAVE-2026-09-01-icons-standup-startend-design.md`'s open
Start/End-of-period question). Implementation in progress -- see
"Implementation status" at the bottom, updated as pieces land.

## The 5 units

Day, Week, Month, Quarter, Year. Quarters are traditional calendar
quarters remain traditional calendar quarters (Jan-Mar, Apr-Jun, Jul-Sep,
Oct-Dec); years use the configurable 12-month fiscal-year boundaries in
Settings and are labeled by their ending calendar year, such as FY2026.

## Two kinds of thing, not "Start vs End"

Reframed from last session's "Start of X / End of X" into two
distinctly-shaped concepts:

- **Kickoff** -- forward-looking, requires real user thinking/input,
  encouraged for all 5 units. This is where the "tedious data entry"
  cost lives, so it needs active management (see Overlap below).
- **Review** -- backward-looking, mostly auto-generated, low-friction.
  The user shouldn't be asked to do real data entry to get a Review;
  at most a couple of unavoidable, low-cost decisions (e.g. TODO
  carry-forward checkboxes), never open-ended typing as a requirement.

All 5 units get both a Kickoff and a Review, but every unit's Kickoff
*and* Review are individually toggle-able off in settings (see
Per-unit toggles below) -- nothing is mandatory.

## Kickoff: overlap/stacking, not override

Original idea was "bigger unit's Kickoff replaces smaller unit's
Kickoff when they coincide" (e.g. Monday = Week start, so Day kickoff
gets eliminated as redundant). Rejected: each unit's Kickoff content
is genuinely different, not a superset/subset of another unit's --
Day kickoff cares about *today's* due recurring items and today's open
TODOs readback; Week kickoff cares about weekly goals/weekly recurring
items. Not the same content at different resolution.

Instead: each unit contributes an independent **module** (its own
block of questions/content). When multiple units' Kickoffs coincide on
the same calendar day (Week+Day always on the day a week starts;
Month+Week+Day when the 1st is also handled to fall on a week
boundary; Quarter/Year always coincide with Month since quarter/year
boundaries are always also month boundaries), **stack the applicable
modules into one combined dialog** rather than showing one and
dropping the others. A field-level dedupe (e.g. not asking "goals for
today" separately from "goals for this week" if today literally *is*
the first day of the week) may make sense case by case, but that's a
per-field call, not a whole-module elimination.

### Boundary-coincidence complexity -- kept deliberately simple

Precise "which units coincide today" boundary logic could get
arbitrarily fussy (ISO week-start edge cases, leap years, etc). Rather
than building exact detection-and-merge logic up front, the plan is:

- Do the simple/obvious stacking (Week+Day on week-start day,
  Month+Week+Day when applicable, Quarter/Year+Month always on
  quarter/year boundaries since those always land on month
  boundaries too).
- Give every Kickoff/Review dialog a **Dismiss** action at the top,
  for the rare case a user gets shown, say, an EOM immediately
  followed by an EOQ dialog and doesn't want to deal with both right
  now. Dismiss just closes that dialog; no special "already saw this"
  suppression logic needed initially.
- Revisit smarter dedup/presentment later if it turns out to matter in
  practice, rather than over-engineering it now.

### Missed Kickoffs/Reviews

If the app isn't opened on the actual boundary day (very plausible
above Day/Week granularity), fire the (stacked) Kickoff/Review the
next time the app is used, based on a per-unit "last shown" timestamp
-- same pattern as `RecurringMeeting.lastOccurrence` already used for
standup nudges.

## Review: mostly-automatic, but with a hard rule -- never lose data

A Review must never cause an open item (TODO/QUESTION carry-forward,
etc.) to silently vanish just because the user declined to engage with
that day's Review. Concretely, for EOD specifically (the one Review
type with today's existing manual fields -- productivity/meeting-
hours/sentiment/summary/tomorrow's-goals):

- **Save** button: records whatever optional fields were filled in
  (summary, productivity, sentiment, meeting hours, tomorrow's goals --
  skip any left blank), *and* applies carry-forward per whatever the
  checkboxes are currently set to.
- **Coolio** button (declines optional entry, but user still got the
  free AI-generated summary as FYI): skips recording the optional
  fields entirely (no SUMMARY/PRODUCTIVITY/SENTIMENT/goals lines), but
  still applies carry-forward per the checkboxes -- which default to
  checked, so the safe default (nothing declared resolved) still
  carries forward automatically even with zero interaction. Carry-
  forward is a decision with real consequence, not "entry," so it's
  never bundled into "declining to fill in a form."

This "never lose an item without an explicit, deliberate decision"
principle should generalize to every unit's Review, not just Day's.

## Three themes (not per-unit-fixed; per-unit-*defaulted*, still all togglable)

- **Personal Notes** -- informal, terse, for-your-eyes-only framing.
  Default theme for Day and Week.
- **Status Report** -- structured "what happened, what's next" framing
  suitable for sharing informally (e.g. a Friday team update). Default
  for Month.
- **Formal Report** -- sober, presentation-ready framing suitable for
  a boss-facing quarterly/annual review meeting. Default for Quarter
  and Year.
- **Brag Preso** -- a fourth, distinct theme (not a Formal Report
  variant) -- punchier, achievement-highlight framing, structured as
  slide-friendly sections. Available as an alternate theme for Month/
  Quarter/Year, not a default for anything initially.

Each unit's Review has a theme setting (defaulting per the above),
independently changeable -- e.g. someone who does send weekly status
updates can bump Week to "Status Report."

## Output formats -- Markdown/HTML native, Pandoc for everything else

Per this repo's KISS/no-new-heavy-deps stance: Dunnit always generates
**Markdown** (source of truth) and can always render **HTML** from it
(cheap, no new dependency -- reuses the existing markdown-preview
pipeline). Each theme gets Markdown/HTML layout templates suited to
its purpose (notes-style, report-style, slide-style-via `---`-
delimited sections for Brag Preso, pandoc-slide convention).

For DOCX/PPTX/PDF/ODT export: **shell out to `pandoc`** if present on
the user's machine (same external-tool-shell-out pattern as `gh
copilot` already used throughout this codebase), as an optional
"Export via Pandoc..." action layered on any generated
Markdown/HTML -- not a bundled dependency. If `pandoc` isn't
installed, show a message pointing the user at installing it, rather
than trying to reimplement document generation.

## Configurable lead-time/window

Each of Month/Quarter/Year's Review can be configured for how many
days before the actual boundary it's allowed to fire (for scheduling
convenience -- so a Quarterly Review doesn't have to be sprung on you
at the exact boundary), and how wide a window it captures.

- Titles/labels always name the **nominal calendar unit** ("Q2 2026
  (Apr-Jun)", "Sep 2026", "2026"), never the actual data-fetch range
  ("Apr 1 - Jun 25") -- see `periodLabel` below.
- If a Review fires before the unit's actual end, the digest is
  generated from whatever's available at that point (option 1 from
  the design discussion -- no deferred/regenerate-in-place complexity)
  -- caveated in the body text if helpful, but not blocking.

Data-fetch windows are also allowed to be **loosely padded** (e.g. a
day or so beyond the nominal boundary in either direction) --
duplicate coverage of the same days across adjacent reports is
acceptable; a coverage *gap* at a boundary is the worse failure mode
and is what padding is meant to avoid. No need for exact interval
math.

## Hierarchical rollup: Reviews feed on prior Reviews, not just raw ledger

Each unit's Review should prefer already-generated, possibly hand-
edited sub-tier Review reports as its source material, falling back to
raw ledger text only for whatever sub-range isn't covered by a saved
sub-report:

- Month's sub-tier is Week.
- Quarter's sub-tier is Month.
- Year's sub-tier is Quarter.
- Day and Week have no sub-tier; always raw ledger.

This respects hand-edits (a user-polished Weekly report is better
source material than re-summarizing raw entries), and keeps AI
prompt/context size down at the higher tiers (a Quarterly digest reads
~3 Monthly reports instead of ~90 days of raw ledger).

- **No auto-refresh on sub-report edits**: if a Weekly report is hand-
  edited *after* its parent Monthly review already consumed it, the
  Monthly review does not automatically regenerate. Known, accepted
  limitation.
- **Partial coverage falls back per-gap**: if only some of a period's
  sub-reports exist, use the ones that do and fall back to raw ledger
  just for the uncovered remainder.
- **Loose date matching is fine** (per the padding note above) -- a
  sub-report just needs to be found as overlapping the requested
  range at all; no need for precise interval math, occasional
  duplicate coverage across a sub-report and the raw-ledger fallback
  is acceptable.

### Sketch

```go
// Period is one of the 5 granularities.
type Period string

const (
	PeriodDay     Period = "day"
	PeriodWeek    Period = "week"
	PeriodMonth   Period = "month"
	PeriodQuarter Period = "quarter"
	PeriodYear    Period = "year"
)

// periodConfig holds what differs per unit: label format, nominal
// boundary calculation, sub-tier (for rollup), default theme,
// settings-toggle keys for Kickoff/Review, lead-time/window config.
type periodConfig struct {
	Period       Period
	SubPeriod    Period // "" if none (Day, Week)
	DefaultTheme string
	// ...boundary calc, toggle keys, lead-time fields, etc.
}

// periodLabel returns the human/report-facing name for a period,
// e.g. "Q2 2026 (Apr-Jun)", "Week of Sep 1", "Sep 2026", "2026" --
// always names the *nominal* calendar unit, independent of whatever
// partial data-fetch window was actually available.
func periodLabel(period Period, anchor time.Time) string

// periodDataRange returns the [from, to] window to pull source
// material from for a period's Review -- padded loosely per the
// looseness note above, not an exact boundary.
func periodDataRange(period Period, anchor time.Time, cfg periodConfig) (from, to time.Time)

// reviewSourceMaterial is what gets fed to the copilot prompt for a
// Review: already-generated sub-tier reports (their saved, possibly
// hand-edited markdown) found to overlap the range, plus raw ledger
// text for whatever sub-range isn't covered by one.
type reviewSourceMaterial struct {
	SubReports []string // markdown bodies of covered sub-period reports, in order
	RawLedger  string   // ledger text for whatever's left uncovered
}

// gatherReviewSourceMaterial builds the rollup for period's Review of
// [from, to]:
//  1. Determine period's sub-tier via periodConfig (Day/Week have
//     none -- always raw ledger only).
//  2. List saved report files for that sub-tier kind (globs
//     period-appropriate report directories via the shared path
//     helpers), parse each's embedded date, keep ones whose
//     nominal range intersects [from, to] at all (loose).
//  3. Track, loosely, which whole sub-periods were found covered.
//  4. For any day within [from, to] not covered by a found
//     sub-report, fall back to that day's raw ledger text
//     (gatherLedgerTextForRange).
//  5. Return both lists separately, so the prompt can frame them
//     differently ("Prior summaries:" vs "Additional raw entries not
//     yet summarized:").
func gatherReviewSourceMaterial(period Period, from, to time.Time) reviewSourceMaterial
```

## Keeping prompts/outputs small

Two levers, both needed:

1. **Structural**: the rollup above is the main lever -- feeding prior
   summaries instead of raw ledger keeps input size down at higher
   tiers automatically.
2. **Explicit prompt instruction**: every Review-generating
   `summarizeWithLLMCLIPrompt` call should append a shared,
   period-scaled length constraint (e.g. "Keep the total output under
   roughly N words; be terse, favor bullet points over prose" with N
   smaller for Day, larger for Year) -- so length discipline doesn't
   depend on each call site's prompt wording being separately tuned
   well.

## Per-unit toggles

Every unit's Kickoff and Review (10 toggles total: 5 units x
{Kickoff, Review}) is independently on/off in Settings. Someone who
doesn't do formal quarterly planning simply turns Quarter Kickoff/
Review off and never sees that surface. This is the primary lever for
"forcing 5 units on everyone is a lot to ask," more so than any amount
of smart merging/dedup logic.

## Scope note on this doc vs. current implementation

This document describes the target design. As of 2026-09-01, actual
code still has SOD/EOD/SOM as three structurally different, mostly
hand-rolled dialogs (see `dunnit/sod.go`, `dunnit/eod.go`,
`dunnit/som.go`) -- SOM in particular still conflates End-of-prior-
month and Start-of-new-month in one wizard, which this design would
eventually split into Month's Review and Month's Kickoff as separate
(but maybe still-adjacent-in-menu) flows. `dunnit/report.go`'s
shared report-path/`showGeneratedReport` helpers (added 2026-09-01,
see `SESSION-SAVE-2026-09-01-icons-standup-startend-design.md`) are
the first concrete step toward this design and remain compatible with
it (the sketch above reuses the shared report naming convention for
rollup lookups).

## Implementation status

- [x] Shared report Copy/Save/path helpers (`dunnit/report.go`)
- [x] `Period`/`periodConfig` type and per-unit config
      (`dunnit/period.go`, reuses existing `summaryPeriod` +
      `periodYear` addition)
- [x] Per-unit Kickoff/Review settings toggles (`Config` fields in
      `dunnit/config.go` -- no dedicated Settings-window UI yet, only
      config.toml-editable so far; menu regroup reads them)
- [x] `periodLabel` / `periodDataRange` (`dunnit/period.go`)
- [x] `gatherReviewSourceMaterial` rollup + ledger fallback
      (`dunnit/review.go`)
- [x] Shared length-constraint prompt suffix
      (`reviewLengthConstraint`, wired into SOM + EOD)
- [x] Theme system (Personal Notes / Status Report / Formal Report /
      Brag Preso) + per-unit default + override (`themePromptFraming`,
      `generateThemedReview`; wired into SOM with a theme dropdown)
- [ ] Pandoc export action
- [x] Week Kickoff + Review (`dunnit/periodkickoff.go`,
      `dunnit/periodreview.go`) -- first tier with no prior art
- [x] Quarter Kickoff + Review (generic `showPeriodKickoffWindow`/
      `showPeriodReviewWindow`, gated off by default)
- [x] Year Kickoff + Review (same generic dialogs, gated off by
      default)
- [ ] **Quarter/Year-specific Kickoff *content*** -- important
      distinction: Quarter/Year currently reuse the exact same generic
      dialog as Week (open-items readback + quick-add; themed AI
      digest + carry-forward for Review). No OKR/theme-setting-style
      prompts or other content unique to Quarter/Year granularity has
      been designed or built yet -- this was floated as a follow-up
      idea but never actually scoped. Still open.
- [ ] Split SOM into Month Review + Month Kickoff
- [x] Menu regroup: `Kickoff.../Review...` submenus in `ui.go`, now
      listing all 5 units (Quarter/Year hidden until their Config
      toggles are enabled)
- [x] Settings-window UI for the 10 toggles + 5 theme defaults
      (`showSettings` in `settings.go`), with `RebuildTrayMenu()`
      applying toggle changes to the tray menu immediately, no
      restart needed
- [ ] Reconcile Reports menu's "Annual Review..." into `Review... ->
      Year` once Year Review exists

## Menu/UI shape -- decided 2026-09-01

1. **Tray menu**: replace today's flat SOD/EOD/SOM items with two
   submenus, `Kickoff...` -> Day/Week/Month/Quarter/Year and
   `Review...` -> Day/Week/Month/Quarter/Year.
2. **Theme selection**: both. Settings holds each unit's *default*
   theme (persisted, standing preference); the Review dialog itself
   also shows a quick theme dropdown that can override just that one
   generated instance without changing the standing default.
3. **Annual Review reconciliation**: the existing Reports-menu
   "Annual Review" report and the new automatic Year Review are the
   same concept and should be reconciled into one -- Year's Review
   (boundary-triggered or manually invoked via the Review submenu)
   *is* the Annual Review going forward, rather than keeping two
   parallel implementations. The Reports menu's standalone "Annual
   Review..." entry should be retired/folded into `Review... -> Year`
   once Year Review is implemented (ad hoc/arbitrary-date-range
   invocation, per last session's point 6, still available -- just
   through the unified Review flow, not a second separate feature).

## Week label format and app-wide typography -- decided 2026-09-02

- **Week label**: `"Week of Aug 31 – Sep 4 — W36"` -- en-dash between
  the date range, em-dash before the ISO week number, no parens. Same-
  month weeks collapse the end date to just the day number (`"Week of
  Sep 7 – 11 — W37"`). Implemented as `weekLabel` in `period.go`, used
  by `periodLabel(cfg, periodWeek, anchor)`.
- **`Config.ExtendWorkWeekTo7Days`**: new toggle (default `false` =
  Mon-Fri 5-day display) switching Week's *label* to the full Mon-Sun
  7-day span. Display-only -- Week's actual data-gathering range is
  unaffected either way, so a weekend ledger entry is never excluded
  from a Week Review just because the label reads "Mon-Fri".
- **App-wide typography default**: all new user-facing strings (window
  titles, labels, placeholders, button/dialog text) should use proper
  typography going forward -- em-dash "—" for sentence dashes, en-dash
  "–" for numeric/date ranges, curly quotes/apostrophes ("’" not "'"),
  real bullets "•" instead of "- " list-item prefixes, real ellipsis
  "…" instead of "...". Applies to UI strings only, not Go comments
  (which stay plain ASCII, existing repo convention) and not the
  existing "..." menu-item-suffix convention (e.g. `"Settings..."`,
  which is a deliberate "opens a dialog" marker, left alone) or "->"
  arrow glyphs, or Markdown source fed to `ParseMarkdown`/
  `NewRichTextFromMarkdown` (must stay ASCII to parse). Swept
  app-wide 2026-09-02 (see git history around that date).
