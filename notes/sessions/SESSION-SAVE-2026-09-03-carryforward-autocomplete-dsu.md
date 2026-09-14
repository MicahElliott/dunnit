# Dunnit Session Save -- 2026-09-03 -- Tag-Autocomplete Fix, TODO Carry-Forward, Window Polish, DSU Improvements

Status: everything described below is implemented and verified
(`make build`/`go vet ./...`/`go test ./...`/`gofmt -l`/
`eca__editor_diagnostics` all clean throughout), but **nothing has
been visually clicked through in the running app** by Micah for most
of it yet -- see "Not yet visually confirmed" below for the priority
list. This is a plain session-recap file (not committed/tracked --
personal-project scratch, per this repo's usual style), not a design
doc; see `docs/todo-carryforward-design.md` for the one true design
doc this session produced.

## What got built this session (chronological)

1. **Tag-autocomplete bug, actually fixed this time.** Root cause:
   ANY Fyne canvas overlay (`widget.PopUpMenu`, and then plain
   `widget.PopUp` when that was tried) makes the canvas's focus
   manager route ALL keyboard input -- typing, arrows, even Escape --
   to the overlay's own (empty) focus manager the instant it exists,
   regardless of explicit `canvas.Focus()` calls. Two prior attempts
   both hit this same wall from different angles. Real fix: new
   `tagAutoEntry` widget (`dunnit/tagautoentry.go`) that renders
   suggestions as a plain **inline sibling widget** (a VBox of
   buttons shown/hidden below the entry in the normal layout) --
   never an overlay -- with real Up/Down/Enter/Escape keyboard
   navigation, preserving Dunnit's mouseless-workflow goal. Migrated
   all 3 call sites: Daybook's main entry (`ui.go`), Start-of-Day's
   new-item field (`sod.go`), Recurring Meetings' tag field
   (`minicalendar.go`).

2. **TODO/GOAL/etc. carry-forward** (the big feature this session,
   full design in `docs/todo-carryforward-design.md`): discovered
   `getOpenItems()` only ever scanned *today's* ledger file, so
   anything logged on a prior day and never resolved silently
   vanished from Planned the next day -- an oversight, not a design
   choice. Considered and rejected a rolling lookback window (breaks
   "today's ledger = today's concerns," needs file-aware
   Done/Postpone bookkeeping); chose **copy-forward at first touch of
   the day** instead:
   - `dunnit/carryforward.go`: `priorOpenItems()` scans all ledger
     history (via the existing `AllLedgerEntries()` index) for
     unresolved items, `runCarryForwardIfNeeded()` copies them into
     today's ledger annotated `(since YYYY-MM-DD)` (plain, greppable,
     parallel to the existing `(via CATEGORY)` convention -- no new
     tag/marker syntax), idempotent per calendar day via
     `Config.LastCarryForwardDate`.
   - Wired at 3 touchpoints with **no wizard/dialog gate**:
     `recordActivity`, `BuildMainWindow`, `showSODWindow` -- whichever
     fires first does it, so a user who logs DONEs before ever
     opening Kickoff still gets carry-forward.
   - Staleness badge (`\u26a0 Nd`, 4-day threshold, `staleBadge()`)
     shown in Planned/SOD/Kickoff wherever an open item's text
     renders.
   - **Postpone redefined**: since unresolved items now carry forward
     automatically by default, Postpone's job flipped from "carry
     this forward" to **"stop carrying this forward, park it in
     SOMEDAY on purpose."** EOD's and period-Review's old "Carry
     Forward Open TODOs/QUESTIONs" checkboxes were repurposed into
     Postpone-opt-out checkboxes (default unchecked); the old
     `carryForwardItem` function was removed entirely.
   - **SOMEDAY browser built** (`dunnit/somedaybrowser.go`): since
     Postponed items stop appearing in daily UI, added "SOMEDAY
     Items..." under the Ledger tray menu, plus a "Browse SOMEDAY
     Items..." button at the bottom of Planned's expanded "Show all"
     view. Lists every still-unhandled SOMEDAY item with `-> TODO`/
     `-> GOAL`/`Discard` actions, using the same `(via SOMEDAY)`
     resolution-suffix convention as everything else.
   - Cold-start guard (first-ever run surfacing a user's *entire*
     ledger backlog at once) was discussed and deliberately **not
     built** -- no plausible way a brand-new user without pre-existing
     ledger history hits it.
   - Tests: `carryforward_test.go`, `somedaybrowser_test.go` -- all
     passing.

3. **Window edge padding audit + fix.** A sub-agent audit found only
   Daybook (`BuildMainWindow`) had outer edge padding; all ~43 other
   `.SetContent(...)` call sites across 24 files rendered flush
   against window edges. Added a shared `windowPad()` helper
   (`dunnit/windowpad.go`, same `layout.NewCustomPaddedLayout(10,10,10,10)`
   Daybook already used) and applied it everywhere. Daybook itself
   refactored to use the shared helper too (no behavior change there).

4. **Tray menu**: briefly extracted "Recurring Meetings.../Recurring
   Items..." into their own top-level "Recurring" submenu, then
   **reverted per Micah's explicit request** -- both entries are back
   in their original locations only (Meetings submenu / Ledger
   submenu), no separate Recurring menu exists.

5. **Post-Meeting Capture**: now opens at meeting **start** (alongside
   Meeting Prep) instead of the old 15-45-min-after nudge, so it can
   be filled in live during the meeting; removed the now-dead
   post-meeting-window scheduling logic
   (`dueForPostMeetingNudge` deleted, left as a comment for history).
   TODO gets its own larger multi-line box at the bottom of the
   dialog (renamed out of the inline per-category fields since it's
   the category most likely to need several entries per meeting);
   TIL/GOAL/RISK are all multi-line too now (one item per line, blank
   lines skipped).

6. **DSU/Standup improvements** (several rounds):
   - **Root-cause fix for "Report Exclude Tags" not working**:
     `lineHasExcludedTag` matched tags **case-sensitively** --
     `#Home` logged vs `#home` configured in Settings silently didn't
     match. Fixed to case-insensitive matching; regression test added
     (`summarize_test.go`).
   - **Scrum-style prompt reframing**: `summarizeStandupWithCopilot`'s
     prompt now explicitly structures output under "What did I do
     yesterday / What will I do today / Risks & blockers," with an
     explicit "what do you need help with" callout and "focus on
     concrete results, not a busy-sounding activity log" instruction.
     Today's still-open TODOs/GOALs (`getOpenItems()`) are now fed in
     as real material for the "today" section (previously nothing
     backed that section at all).
   - **Editable pre-generation step**: Standup Summary's item list
     changed from a per-line Hide-icon-only display into a free-text
     editable box, seeded with gathered items, with an explicit
     instructional note that edits here only change this
     generation's prompt input, never the ledger itself.
   - **Scope widened further** (final round, per explicit request):
     `standupCategories` changed from a hardcoded `{DONE, WIN}` to
     **all of "Now" (excluding ONGOING) + all of "Reflect" (excluding
     EODOnly SUMMARY/PRODUCTIVITY/MEETING_HOURS)**, built dynamically
     from `categoryGroupOrder()` (`buildStandupCategories()`) so a
     future category addition to either group is picked up
     automatically. Test added
     (`standup_categories_test.go`) pinning the exact expected set.

7. **Daybook polish** (final round):
   - **Mins box now only shows for completable/"endpoint" categories**
     (DONE/FAIL/WASTED) -- previously always visible regardless of
     category, which risked reading as a time *estimate* for Plan
     items (TODO/GOAL/etc), a different concept never intended.
     Repurposed the existing-but-unused `IsTimeTrackable`/
     `timeTrackableCategories` helper (`categories.go`), narrowed from
     its old broader set (`DONE/ONGOING/TODO/FAIL/WASTED/MEETING/WAITING`)
     down to exactly the three documented "endpoint" categories.
     Wiring was simple: `minsWrapper.Show()`/`Hide()` driven by the
     category picker's `OnChanged`; `stretchRowLayout` already
     correctly skips hidden objects for both layout and MinSize, so
     `input` reclaims the space automatically -- no custom widget work
     needed.
   - **WASTED help text reworded**: now leads with "unfocused,
     pointless work or distraction -- a simple way to keep track of
     time you feel wasn't well spent," with the existing
     endpoint/DONE-sibling framing kept as a secondary note rather
     than the primary framing.

## Design docs produced/updated this session

- `docs/todo-carryforward-design.md` -- new, full design doc (ADR-ish,
  matching `docs/category-taxonomy.md`/`docs/navigator-design.md`'s
  informal-notes footing): rejected-lookback-window rationale,
  chosen copy-forward approach, staleness display, Postpone's
  redefined meaning, SOMEDAY-browser rationale, "when it runs"/no-
  wizard-gate rationale, full implementation-status checklist (kept
  updated as pieces landed this session).

## Not yet visually confirmed by Micah

**Everything below was built/tested via `make build`/`go vet`/
`go test ./...`/`gofmt`/`eca__editor_diagnostics` and targeted unit
tests, but the agent has no runtime GUI access** -- nothing from this
session has been clicked through in the actual running app yet.
Highest-priority things to try next session:

1. **Tag autocomplete** (again) -- this is the *third* attempt at this
   feature across two sessions; the root-cause understanding is much
   more solid this time (any overlay breaks keyboard routing, not
   just PopUpMenu specifically), but needs real confirmation:
   multi-letter typing, Up/Down navigation, Enter-to-accept,
   Escape-to-dismiss, and that it truly never re-steals focus, across
   all 3 call sites (Daybook, SOD, Recurring Meetings).
2. **TODO carry-forward end to end**: does a TODO logged yesterday
   actually appear in today's Planned with the right text (no visible
   raw `(since ...)` suffix, since display strips it)? Does the stale
   badge show up correctly after 4+ days? Does Postpone actually stop
   it from reappearing tomorrow? Does the SOMEDAY browser
   (tray menu + Planned's "Browse SOMEDAY Items..." button) show
   postponed items and correctly promote/discard them?
3. **Window padding** -- spot-check a handful of the ~24 touched files
   (not all 43 call sites individually) to confirm the padding reads
   as intended and nothing regressed layout-wise (e.g. Settings,
   Standup Summary, Reports Library, SOMEDAY browser).
4. **Post-Meeting Capture at meeting start** -- confirm it actually
   pops up alongside Meeting Prep when a recurring meeting's start
   time nudge fires (not just the tray-menu-invoked path, which is
   easy to test manually), and that the multi-line TODO/TIL/GOAL/RISK
   fields behave (multiple lines each become separate ledger entries).
5. **DSU/Standup**: confirm the editable items box is pre-populated
   correctly with the wider Now+Reflect category set, that edits
   there actually change what's summarized, that exclude-tags now
   actually excludes case-insensitively, and that the generated
   summary reads like a real scrum standup (3 headings, "today"
   section using real open TODOs/GOALs, help-needed callout).
6. **Daybook mins box** -- confirm it only shows for DONE/FAIL/WASTED
   and correctly hides/reflows for every other category, including via
   the group-filter dropdown (Now/Plan/Reflect/All/Faves) switching
   the category list out from under it.
7. **WASTED help text** -- quick glance at Help window to confirm the
   new wording reads naturally.

## Carried-over open items (from prior sessions, still open)

Still true, not touched this session (see
`SESSION-SAVE-2026-09-02-navigator-reports-taxonomy.md` and
`SESSION-SAVE-2026-09-02-okr-picker-bugfixes.md` for original
context):

- Full report/summary navigator for *all* saved reports across every
  unit/theme -- Reports Library (built 2026-09-02) partially
  addresses this (kind-filtered browse + search) but not exact
  period-unit/theme cross-referencing.
- OKR "1:1" content -- still totally unscoped.
- `hoverbutton.go`'s tooltip-popup double-click bug -- still only
  patched at Daybook's Planned/Activity/Reflections call sites;
  `standup.go`'s old per-item Hide button used this pattern but was
  removed this session (replaced by the editable text box), so that
  specific instance is now moot; `recurring.go:187` ("Delete") still
  unpatched.
- Trend View deliberately left out of `Config.ReportExcludeTags` --
  still an unconfirmed judgment call.
- Pandoc export action (DOCX/PPTX/PDF) -- still not built.
- Reports menu's standalone "Annual Review..." still not folded into
  "Review... -> Year".
- **New from this session**: possible reconciliation between the new
  SOMEDAY browser and Month Review's existing IDEA/SOMEDAY triage step
  (`monthreview.go`) -- flagged in `docs/todo-carryforward-design.md`'s
  open questions, not reconciled.
- **New from this session**: a passive one-time "Carried forward N
  item(s) from earlier" surfaced note was discussed as a nice-to-have
  but explicitly deferred (not required for correctness) -- still not
  built.

## Verification performed this session

Every change built clean (`make build && go vet ./... && go test ./...`),
`eca__editor_diagnostics` clean, and `gofmt -l` clean on every touched
file (only pre-existing `holidays.go` drift remains, confirmed
untouched by this session, same as noted in prior sessions' saves).
New test files added this session: `carryforward_test.go`,
`somedaybrowser_test.go`, `summarize_test.go`,
`standup_categories_test.go`. No permanent test suite existed before
this repo started accumulating these test files across sessions; still
no CI, per `AGENTS.md` -- manual GUI testing by Micah remains the only
way anything here gets truly confirmed working.

All work is direct-to-`main` commits (personal repo, no branch/PR
involved, per this repo's `AGENTS.md`) -- **not yet committed as of
this save** (worth doing a `git status`/`git diff` review and
committing in smallish logical chunks next session, rather than one
giant commit, given the number of distinct features/fixes bundled
into this session).
