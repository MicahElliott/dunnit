# Dunnit Session Save -- 2026-09-02 -- Kickoff/Review implementation, next: design Quarter/Year Kickoff content

Status: everything below is done, committed, and pushed to
`origin/main`. The next piece of work -- designing what Quarter/Year
Kickoff should actually *contain* -- is **open, not started**. This
file exists so a new session can pick that up without re-deriving
context.

## Background

This session continued from
`SESSION-SAVE-2026-09-01-icons-standup-startend-design.md`, which
ended with an open design question about unifying Start-of-X/End-of-X
across Day/Week/Month/Quarter/Year. That question was resolved into
the **Kickoff/Review** model, written up in full at
`docs/kickoff-review-design.md` -- **read that file first**, it's the
authoritative design doc and has a live "Implementation status"
checklist at the bottom.

## What got built this session (all committed/pushed to `main`)

In rough chronological order:

1. **`dunnit/report.go`**: shared `periodReportPath`/`showGeneratedReport`
   helpers, unifying 3 near-duplicate report-window implementations
   (Standup, SOM, dailysummary.go).
2. **`dunnit/period.go`**: `summaryPeriod` extended with `periodYear`;
   `periodConfig`, `periodLabel`, `periodNominalRange`,
   `periodDataRange`, theme constants, `themePromptFraming`,
   `generateThemedReview`, `reviewLengthConstraint`, `kickoffEnabled`/
   `reviewEnabled`/`themeFor` + their mutating counterparts
   `setKickoffEnabled`/`setReviewEnabled`/`setTheme`, `themeOptions`/
   `themeDisplayNames`/`themeFromDisplayName` (human-readable theme
   labels), `periodRecurringCadence`, `weekLabel`.
3. **`dunnit/config.go`**: 10 new bool toggles (Kickoff/Review enabled
   per unit -- Day/Week/Month default on, Quarter/Year default off),
   5 theme string fields, `ExtendWorkWeekTo7Days` bool (default false
   = Mon-Fri week-label display; display-only, doesn't affect actual
   data range).
4. **`dunnit/review.go`**: `gatherReviewSourceMaterial` -- the
   hierarchical rollup (Month feeds on saved Week reports + raw-ledger
   fallback for gaps; Quarter feeds on Month reports; Year feeds on
   Quarter reports; Day/Week always raw ledger only).
5. **`dunnit/periodkickoff.go`** / **`dunnit/periodreview.go`**:
   generic `showPeriodKickoffWindow(a, period)` /
   `showPeriodReviewWindow(a, period)` -- used by Week, Quarter, and
   Year (Day/Month keep their bespoke SOD/EOD/SOM dialogs). Kickoff =
   open-items readback + quick-add + (for Week/Month only, since
   Quarter/Year have no recurring-item cadence) recurring-item
   suggestions. Review = theme dropdown + explicit "Generate" button
   (no eager AI call) + generated report opens in an editable Markdown
   popup (`showEditableReportWindow` in `report.go`, with live preview,
   Save, and "Copy as HTML" via `goldmark`) + TODO/QUESTION
   carry-forward checkboxes applied unconditionally on Done (never-
   lose-data guarantee, same as EOD).
6. **`dunnit/ui.go`**: tray menu regrouped into `Kickoff...`/
   `Review...` submenus (all 5 units, Quarter/Year hidden unless their
   Config toggle is on). Menu-building extracted into standalone
   `buildTrayMenu(a, w4)` + new `RebuildTrayMenu()`, so Settings
   changes take effect immediately without an app restart. Also added
   `"Standup Summary..."` to the Meetings submenu (was only in Reports
   before, hard to discover as a meeting-prep-adjacent workflow).
7. **`dunnit/settings.go`**: new "Kickoff / Review" section -- one
   enable-checkbox pair + theme dropdown per unit, plus an "Extend
   Work Week to 7 Days" checkbox. Save calls `RebuildTrayMenu()`.
8. **Typography sweep** (app-wide, all `dunnit/*.go` user-facing
   strings): em-dash `—` for sentence dashes, en-dash `–` for
   date/numeric ranges, curly quotes/apostrophes, real bullets `•`,
   real ellipsis `…`. Deliberately left alone: Go comments (stay
   ASCII), the existing `"..."` menu-item-suffix convention, `"->"`
   arrow glyphs, Markdown source fed to parsers. **This is now the
   default going forward for any new UI strings.**
9. **Week label format** (specifically requested wording): `"Week of
   Aug 31 – Sep 4 — W36"` -- en-dash in the range, em-dash before the
   ISO week number, no parens; collapses to `"Week of Sep 7 – 11 —
   W37"` when start/end share a month. Implemented as `weekLabel` in
   `period.go`.

All of this passed `make build && make vet && go test ./...` and
`eca__editor_diagnostics` clean at every commit. Several throwaway
test files were written to verify logic (rollup lookup, label
formatting, theme round-tripping, Quarter/Year plumbing) then deleted
per repo convention -- no permanent test files added this session.

## Known gaps / explicitly NOT done (see `docs/kickoff-review-design.md`'s checklist for the full list)

- **Quarter/Year Kickoff have no unique content** -- they currently
  reuse the exact same generic dialog as Week (open-items readback +
  quick-add; no recurring-item suggestions since Quarter/Year have no
  matching cadence). **This is the next task, see below.**
- Pandoc export action (DOCX/PPTX/PDF) not built -- MD/HTML only so
  far (goldmark-based "Copy as HTML").
- SOM (`dunnit/som.go`) still conflates End-of-prior-month Review and
  Start-of-new-month Kickoff in one 5-step wizard -- not split.
- Reports menu's "Annual Review..." not yet reconciled/folded into
  `Review... -> Year`.

## NEXT TASK: Design Quarter/Year Kickoff content

Micah wants to design this in a **new session** (this file is the
handoff). Starting points for that conversation:

- Read `docs/kickoff-review-design.md` in full first, especially the
  "Three themes" and "Menu/UI shape" sections, for the established
  vocabulary (Kickoff vs Review, the 4 themes, per-unit toggles).
- The generic `showPeriodKickoffWindow` (in `dunnit/periodkickoff.go`)
  is the current placeholder implementation for Quarter/Year -- read
  it to see exactly what it does today (open-items readback + quick-
  add only, no recurring-item box since `periodRecurringCadence`
  returns `""` for both).
- Ideas floated in earlier conversation (never scoped/agreed) worth
  raising again: OKR-setting prompts, "theme for the quarter/year"
  framing, something tied to performance-review cycles. No decisions
  were made on any of this -- fully open.
- Whatever gets designed should probably still fit the existing
  `periodConfig`/`showPeriodKickoffWindow` architecture if possible
  (e.g. as additional optional fields/sections gated by period, rather
  than forking Quarter/Year into their own bespoke dialogs) -- but
  don't treat that as a constraint if the content genuinely needs a
  different shape; ask Micah rather than assuming.
- Worth asking Micah directly: does he actually use quarterly/annual
  goal-setting himself (e.g. for a work performance-review cycle), and
  if so what does that process concretely look like today (outside
  Dunnit) that a Quarter/Year Kickoff could usefully echo?

## Verification performed this session

`make build`, `make vet`, `go test ./...`, `eca__editor_diagnostics`
all clean after every change, both incrementally per-commit and in a
final check before this save. `gofmt -l` confirmed clean on every
touched file (one incidental gofmt cleanup of pre-existing drift in
`ui.go`, unrelated to this session's changes, included for free when
`gofmt -w` was run). One sub-agent-run typography sweep was reviewed
carefully by hand afterward (diff-by-diff) since the sub-agent
self-reported two files it worried it had corrupted mid-edit -- both
were confirmed fixed (a corrupted comment in `ui.go`, a missing
trailing newline in `recurring.go`) before committing.

All work is committed to `main` and pushed to `origin/main` (personal
repo, direct-to-main is fine per this repo's `AGENTS.md`). No PR/
branch involved.
