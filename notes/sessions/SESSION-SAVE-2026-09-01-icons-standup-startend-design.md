# Dunnit Session Save -- 2026-09-01 -- icons/UI polish, standup rework, Start/End design question

Status: everything below **except the final open design discussion**
is done, committed, and pushed to `origin/main`. The Start/End-of-
period convergence question is **open, not started** -- this file
exists so a new session can pick up that brainstorm without
re-deriving context.

## 1. Fyne theme icons (done)

Fyne has a built-in icon set (`theme.IconName*`, Material-style,
theme-aware light/dark) -- confirmed via the local Fyne module source
and the user's cloned `fyne-demo` repo. Swapped in for action buttons
with a good conceptual match:

- Discard -> `theme.IconNameContentClear`
- Done -> `theme.IconNameConfirm`
- Delete (Recurring Items list) -> `theme.IconNameDelete`
- Postpone -> `theme.IconNameHistory`
- Edit (both Activity and Reflections sections) -> `theme.IconNameDocumentCreate`

New `newHoverIconButton` in `hoverbutton.go` (icon-based sibling of
`newHoverButton`) so the hover-tooltip/click-swallowing fix applies to
icon buttons too.

**Left as emoji, deliberately:**
- Category glyphs (TODO/GOAL/IDEA/etc in `categories.go`) -- Fyne's
  icon set has no equivalents for these expressive/mnemonic concepts
  (no lightbulb, trophy, seedling icons), so emoji stay as the only
  reasonable option there. This was an explicit design goal restated
  by Micah this session: "I'd like to have emojis only used in CATs."
- 🔥 in `streak.go`'s streak label, and 📝 in `recurring.go`'s help
  text -- both flavor/text emoji, not action-button icons. Micah was
  asked explicitly whether to also strip these for full
  emoji-only-in-CATs consistency and said **no, leave those two**.

Full audit was done via grep for the emoji Unicode ranges across
`dunnit/` -- confirmed no other non-CAT emoji remain outside those two
explicitly-kept spots.

## 2. Daybook "Planned" section collapses to TODO-only (done)

Was showing all of TODO/GOAL/WAITING/QUESTION/FIXME/RISK grouped
together and "feeling overwhelming" per Micah. Now:
- Only TODOs shown by default.
- A "Show all (N more)" / "Show TODOs only" toggle button appears
  (only if there are other-category items) to reveal/hide the rest.
- Toggle state (`showAllPlanned` in `ui.go`) is a plain closure-local
  bool -- persists across refreshes within one Daybook session, resets
  to TODO-only on next app launch (not persisted to config.toml; this
  wasn't discussed/asked for, worth flagging if it turns out
  cross-session persistence is actually wanted).

## 3. SOM digest fixes (done)

- Placeholder text was reused from EOD's ("Generating... please
  wait") -- fixed to "*An AI-generated summary will show up here
  shortly...*", explicit that it's AI content still loading.
- Digest's copilot prompt was reusing `summarizeWithCopilot`'s generic
  "standup or status update" framing, so SOM's own report literally
  said "Standup summary..." even though it's a full month-in-review.
  Fixed: SOM now calls `summarizeWithCopilotPrompt` directly with its
  own prompt, explicitly titled e.g. "Aug 2026 Summary" (via
  `from.Format("Jan 2006")`).
- Added Copy (clipboard) + Save (writes to
  `DunnitDir()/som-YYYYMM.md`) buttons under the digest, mirroring the
  pattern from Standup Summary's generated-output window
  (`showGeneratedStandupSummary` in `standup.go`). **Note**: this is
  the third slightly-different Copy/Save/report-window
  implementation in the codebase now (Standup's
  `showGeneratedStandupSummary`, this new inline SOM version, and
  `ensureDailySummaryDoc`'s daily-summary-doc file convention) --
  flagged below as something the Start/End convergence design should
  probably unify.

## 4. Standup Summary enhancements (done, from earlier in this session
   but included here for completeness since it's directly relevant to
   the open design discussion below)

- Per-item Hide action (👁️-off icon) -- view-only, never writes to
  ledger.
- Bottom "Generate Summary" button -- runs only the still-visible
  (non-hidden) items through a standup-framed
  `summarizeStandupWithCopilot` prompt, shows result via
  `showGeneratedStandupSummary` with Copy/Save (saves to
  `DunnitDir()/dsu-YYYYMMDD.md`).
- Time window fixed to span across midnight: previously only pulled
  "yesterday's" ledger (or Fri+Sat+Sun on Monday), missing anything
  logged *this morning before the actual standup meeting*. Now
  computed via `standupWindowStart` -- uses the configured `#dsu`
  `RecurringMeeting`'s actual `lastOccurrence` time as the boundary if
  one exists, falling back to the old weekday-aware "yesterday"
  heuristic otherwise.
- Drag-to-reorder was requested but explicitly **not implemented**:
  Fyne has no built-in draggable-list-reorder widget; hand-rolling one
  (custom drag gestures + list reflow) wasn't judged worth it. Items
  stay in chronological order.

## 5. Recurring Meetings: daily cadence added (done, also from earlier
   this session, relevant background for the discussion below)

`RecurringMeeting` gained a `Cadence` field ("weekly" default/legacy
or "daily", plus `WeekendPolicy` mirroring `RecurringItem`'s). Backstory:
Micah asked for a note on the Recurring Meetings window explaining
"daily recurrences are treated as standups, use dialog instead" --
investigation revealed **standups weren't actually a distinct feature
at all**: they were 5 separate weekly `RecurringMeeting` rows (one per
weekday) all tagged `#dsu`, specially detected by exact tag match in
`sched.go`'s pre-meeting-nudge job to trigger `showStandupExport`
instead of the generic `showMeetingPrepDialog`. Micah chose to add
real daily-cadence support so a standup is now just one row instead of
5. The window's top note was updated to his exact requested wording:
"Use these tags throughout your weeks any time a meeting topic thought
comes to mind. They'll be collected and presented to you just before
your meeting starts."

## 6. OPEN DESIGN QUESTION -- Start/End-of-period convergence (NOT STARTED)

Micah's prompt (paraphrased): we have Start of Day (SOD), End of Day
(EOD), Start of Month (SOM), but no Start/End of Week, no Start/End of
Quarter, no Start/End of Year -- as more get added this will sprawl.
Should we converge the design? Should Start/End be a necessary
separation? How should recording summaries (Meeting/Day/Week/Month/
Quarter/Year) be unified? Any Start/End dialog must be very clear
whether it's asking about the *last* period (retrospective) vs the
*upcoming* one (prospective) -- and if retrospective, the AI-generation
step must not fire until *after* all user data entry for that dialog
is done (so the digest can incorporate what the user just typed, if
ever needed -- currently a latent constraint, not yet a live bug,
since EOD/SOM's AI digests today only read already-recorded ledger
entries, never that session's own in-progress form fields).

### Current state as of this session (confirmed by reading the code)

| | Start | End |
|---|---|---|
| Day | SOD (`sod.go`): open TODOs/GOALs readback, quick-add, daily/weekly recurring-item suggestions | EOD (`eod.go`): AI day summary (editable + rendered-markdown preview), productivity/meeting-hours/sentiment prompts, tomorrow's goals, TODO/QUESTION carry-forward |
| Week | **none** -- only a "weekly digest" *nudge* (`sched.go`'s `WeeklyDigestDay`/`WeeklyDigestTime`, opt-in, disabled by default) that just fires `runSummarize(a, periodWeek)` unprompted; not a real dialog, no user-input steps at all | **none** |
| Month | SOM (`som.go`) does **both** halves in one 5-step wizard: prior month's AI digest+triage (End-of-prior-month work) *and* new month's GOALs + monthly recurring-item suggestions (Start-of-new-month work) -- its own name is a bit of a misnomer, it's really "Month Transition" | *(folded into SOM, see above)* |
| Quarter | none (Annual Review-adjacent reports exist, see Reports menu, but no Start/End of Quarter dialog) | none |
| Year | none | none (Annual Review in Reports menu is a separate one-off report, not an End-of-Year *dialog* with prompts) |

Tray menu today (`ui.go`'s `BuildMainWindow`, near the bottom): SOD/
EOD/SOM are flat top-level items alongside Snooze/DND, with Meetings/
Reports/Ledger as submenus. Reports submenu already has: Summarize...,
Standup Summary..., Status Report..., Annual Review..., Trend View....

### Proposed direction floated this session (Micah had NOT yet responded
   when the session ended -- these are ideas to react to /
   critique / revise, not decisions)

1. **Keep Start vs End as a necessary separation** -- they have
   different jobs (End = retrospective/record what just happened;
   Start = prospective/set up what's next), and collapsing them would
   undermine exactly the "is this asking about last vs upcoming"
   clarity Micah wants. Suggested making the distinction *load-bearing*
   in the UI itself (see menu regroup below) rather than fixing it
   with better in-dialog copy alone.

2. **Factor a shared, parameterized engine**: something like
   `showEndOfPeriod(a, period)` / `showStartOfPeriod(a, period)` with a
   small `periodConfig` (label, date-range function, which
   reflection/goal categories apply at that granularity, save-path
   naming, copilot prompt framing) driving both, instead of each of
   EOD/SOM/(future EOW/SOW/EOQ/SOQ/EOY/SOY) hand-rolling its own
   near-duplicate "digest + prompts + Copy/Save + Finalize" scaffold.
   Rationale: already caught real duplication forming this session
   between Standup's `showGeneratedStandupSummary` and the new inline
   SOM Copy/Save code -- three slightly different report-window/
   save-path implementations exist today (`dsu-YYYYMMDD.md` from
   Standup, `som-YYYYMM.md` just added, `ensureDailySummaryDoc`'s
   daily-summary-doc convention).

3. **Menu regroup**: replace the growing flat list with two submenus,
   `Start of...` -> Day/Week/Month/Quarter/Year and `End of...` ->
   Day/Week/Month/Quarter/Year -- scales better than flat items, and
   the grouping itself reinforces the Start-vs-End mental model right
   in the menu structure (addresses the "must be very clear" ask
   partly through information architecture, not just dialog copy).

4. **Unify report saving**: one helper, something like
   `periodReportPath(period, date) string` with a consistent naming
   scheme (e.g. `report-day-20260901.md`, `report-week-2026W36.md`,
   `report-month-202609.md`), and one shared
   `showGeneratedReport(a, parent, title, text)` window (Copy/Save/
   Close) reused everywhere instead of each feature growing its own
   slightly-different copy.

5. **AI-timing rule, stated explicitly**: any End-of-X AI digest must
   be generated *before* the dialog opens for review *unless* the
   digest is meant to incorporate something the user types in that
   same session -- in which case generation must be deferred until
   after Finalize/submit, never fired eagerly at dialog-open time.
   Not a live bug today (EOD/SOM's digests only read already-recorded
   ledger entries), but a constraint to hold to as new tiers get built
   -- especially if a "final reflections" free-text box ever needs to
   feed into its own period's AI digest.

### Where the conversation stopped

Micah was mid-way through being asked to choose a scope for this pass
(three options offered: full convergence now / just add a Week tier
using today's copy-paste style and defer the refactor / refactor
Day+Month onto the shared engine first without adding new tiers yet)
when he asked to checkpoint instead and save for a future session.
**No scope decision has been made.** Good places to restart:

- Does Micah agree with "keep Start/End separate" and the 5-point
  proposal above, or does he have a different shape in mind?
- Is Week the next tier to add, or does Quarter/Year matter first
  (e.g. for a performance-review cycle)?
- Should the shared engine be built test-first/incrementally against
  the *existing* Day/Month dialogs (safer, proves the abstraction
  before adding new surface area) or is it fine to design the engine
  fresh and retrofit later?
- Worth asking Micah for concrete examples of what a Start-of-Week and
  End-of-Week dialog should actually *contain* (which categories/
  prompts matter at weekly granularity that don't already fit neatly
  into Day or Month) before generalizing further -- Week is the one
  tier with zero prior art in this codebase to generalize from.

## Verification performed this session

`make build`, `make vet`, `go test ./...`, `eca__editor_diagnostics`
all clean after every change. Several throwaway test files were
written to verify logic (holiday date calculations for 2026/2027,
daily-meeting-cadence occurrence/nudge timing, legacy-config backward
compatibility, standup window-spanning-midnight logic) then deleted
per repo convention -- no permanent test files added (repo still has
only `undo_test.go` pre-existing this session's work, though Daybook/
categories/standup logic now has meaningfully more implicit test
coverage history in this chat's transcript if ever revisited).

All work is committed to `main` and pushed to `origin/main` (personal
repo, direct-to-main is fine per this repo's `AGENTS.md`). No PR/
branch involved.
