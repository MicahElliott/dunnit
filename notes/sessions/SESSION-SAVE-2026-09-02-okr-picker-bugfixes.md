# Dunnit Session Save -- 2026-09-02 -- Pending/Open Items After OKR + Bugfix Session

Status: everything from this session is committed and pushed to
`origin/main` (commits `e4f407f`, `99b69a4`, `5dd4dc3`, on top of the
earlier `ce19d60`/`fdcc585`/... Kickoff/Review work). This file is a
punch-list of things explicitly raised/pondered during the session but
deliberately deferred, not started, or flagged as needing a decision --
so a new session can pick up without re-deriving context. Read
`docs/kickoff-review-design.md` first for the underlying Kickoff/
Review model if any of this needs re-deriving.

## 1. Full report/summary "navigator" (bigger version of the period picker)

`showPeriodPicker` (new this session, `dunnit/periodpicker.go`) is a
deliberately minimal "this period / last period / a few more back"
picker for Week/Month/Quarter/Year Review. Explicitly **not** the
fuller thing floated during design discussion: a real navigator that
lets you browse *all* saved reports across every unit and every theme
(e.g. "show me every Quarter Review I've ever generated, regardless of
theme, and let me open any of them"). `listReviewReportsForPeriod`
(review.go) already does exact-period-match lookup with theme parsing
out of the filename, which could be a building block, but no browsing
UI exists yet. Worth scoping as its own design conversation, not a
quick bolt-on.

## 2. OKR "1:1" content -- still totally unscoped

When OKR support was discussed, Micah floated wanting Kickoff to
possibly support "themes/1x1" content in addition to OKRs, but
explicitly said he didn't know what that would look like yet. Nothing
was built for this -- only the OKR Objective/KeyResult/FOCUS ("theme
for this period") pieces got built. If Micah ever has a concrete
real-world 1:1 template/process to echo, that's the next design
conversation for Kickoff content, separate from OKRs.

## 3. `hoverbutton.go`'s tooltip-popup bug -- only patched at the call sites reported buggy

The double-click root cause (Fyne's overlay hit-testing swallowing a
click once a hover tooltip popup is showing) was fixed by moving
Daybook's Planned/Activity/Reflections inline buttons off
`newHoverIconButton` entirely, onto plain icon-only
`widget.NewButtonWithIcon`. Two other call sites still use
`newHoverIconButton` and were **not** touched since they weren't
reported as buggy:
- `standup.go:204` ("Hide" button)
- `recurring.go:187` ("Delete" button)

These are plausibly exposed to the same underlying Fyne bug (the root
cause is generic, not Daybook-specific) -- just not yet reported as
annoying in practice. Worth a quick look/fix if the double-click
symptom is ever seen there too, or possibly worth preemptively fixing
for consistency now that the pattern's understood. Not done this
session since Micah's report was specifically scoped to Daybook's
three sections.

## 4. Spacing -- fixed root cause, but not yet visually confirmed

Found and fixed a real bug this session: `BuildMainWindow` was calling
`a.Settings().SetTheme(theme.LightTheme())`, which silently clobbered
the compact theme set in `MakeUI`, meaning **two full tightening
passes earlier in the session had zero actual effect** -- the compact
theme was never live. Fixed by having `compactTheme` (`dunnit/theme.go`)
wrap `LightTheme` directly and only being set once, in
`BuildMainWindow`. Padding/InnerPadding/LineSpacing are now at
near-floor values (1/2/1 vs defaults 4/8/4).

**This has not been visually confirmed by Micah yet** since the fix
landed near the end of the session -- next session should start by
having him actually look at Daybook (and a Kickoff/Review window) and
confirm whether it now looks tight enough, too tight, or if there's
still a specific widget/section that looks spacey despite the global
theme fix (in which case that widget likely has its own hardcoded
layout/padding independent of the theme, and needs a targeted look).

## 5. Trend View was deliberately left out of the report tag-filter

`Config.ReportExcludeTags` (new this session) is wired into
`concatLedgerFiles`/`concatLedgerFilesFiltered` (covers Kickoff/
Review, Status Report, Annual Review) and `gatherStandupLines`
(Standup, which reads ledger lines directly). `trend.go`
(`gatherTrendPoints`) was **intentionally not filtered** -- reasoning
was that Trend View surfaces numeric stats (productivity/sentiment
scores over time), not prose fed into an AI-generated report someone
else might read, so the "keep non-work items out of shared reports"
motivation doesn't obviously apply. This was a judgment call, not
something Micah explicitly confirmed -- worth a quick "does this match
your intent?" check next session, especially if a #home-tagged
PRODUCTIVITY/SENTIMENT entry ever shows up somewhere unwanted in a
Trend chart.

## 6. Known older gaps, still open (pre-dating this session, from docs/kickoff-review-design.md's checklist)

- Pandoc export action (DOCX/PPTX/PDF) -- Markdown/HTML only so far.
- Reports menu's standalone "Annual Review..." not yet retired/folded
  into `Review... -> Year` (the design doc calls for this once Year
  Review exists, which it now does).

## Verification performed this session

Every change built clean (`make build && make vet && go test ./...`)
and `eca__editor_diagnostics` clean, incrementally through the session
and in a final pass before each of the 3 commits. `gofmt -l` clean on
every touched file (pre-existing drift in `holidays.go`, untouched by
this session, is the only repo-wide gofmt complaint). All work is
committed to `main` and pushed to `origin/main` (personal repo,
direct-to-main is fine per this repo's `AGENTS.md`) -- no PR/branch
involved. No UI was visually tested by the agent (no runtime access);
several fixes (spacing, double-click, Enter/Esc on Edit dialog) are
logically verified via code reading but **not yet confirmed working by
Micah in the running app** -- worth prioritizing that check early next
session before building more on top.
