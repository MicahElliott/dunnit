# Dunnit Session Save — 2026-08-29 — Phases 0-3 substantially complete

Personal project: `/Users/E463390/proj/dunnit` (git repo, direct commits
to `main` are fine). Latest commit: `70e720d` "feat: (FR-13)
Start-of-Day nudge".

## Where things stand

Massive amount of implementation done since the 2026-08-28 planning
session. **Phase 0, 1, and 2 are fully complete.** Phase 3 has one item
done (FR-11, re-scoped) and one done (FR-13); FR-12 and FR-14 remain.

- **Phase 0** (bugs): FR-01, FR-02 — done.
- **Phase 1** (foundational): FR-03, FR-04, FR-05, FR-06 — done, plus
  several polish rounds (emoji fixes, category reordering/grouping,
  Now/Plan/Reflect filter dropdown, color-coded legend, monospace
  styling where feasible).
- **Phase 2** (TODO/GOAL workflow): FR-07, FR-08, FR-09, FR-10 — done.
- **Phase 3** (meeting/agenda + SOD):
  - **FR-11 (Meeting Prep)** — done, but significantly **re-scoped**
    from the original FRD spec after user feedback. See "Re-scoping
    notes" below — this is the most important thing to read before
    touching `meetingprep.go`.
  - **FR-12 (Agenda view)** — NOT separately implemented. FR-11's
    re-scoped version already covers most of it. FRD has an explicit
    note flagging this overlap; needs a decision on whether FR-12 is
    still needed as its own thing (see FRD's FR-12 section for the
    gap: "last pulled" marker/incremental-only-new-items behavior is
    the main piece still missing).
  - **FR-13 (Start-of-Day nudge)** — done, mirrors FR-09's EOD pattern.
  - **FR-14 (Start-of-Month wizard)** — not started, next candidate in
    Phase 3.

No app code changes are pending/uncommitted — every round this session
ended with `make build && go vet ./...` clean, `go test ./...` clean,
and a commit. Test suite now has real coverage (didn't exist at start
of previous session): `dunnit/*_test.go` covering categories,
todos/resolution logic, undo/edit, EOD carry-forward, tags, meeting
prep pull/filter logic.

## Re-scoping notes (important context for future work)

1. **FR-11/Meeting Prep** ended up quite different from its original
   one-line FRD spec. Actual behavior now: a dialog with a tag field,
   a **category filter dropdown** (`MEETING` default / `Related`
   [MEETING+IDEA+QUESTION+WIN+RISK+GOAL+IMPACT+CAREER] / `All`), a
   **weeks-back dropdown** (1/2/3/4/12, default 2), a "Refresh" button,
   and an **editable-but-non-destructive** history box showing the
   last ~8 matching entries (edits never write back to the ledger).
   Separately, a "Save Note" action still lets you log a fresh
   `MEETING #tag note` line, same as the original capture-only spec.
   The FRD has been updated in-place with dated notes explaining this.
2. **MEETING category moved from "Now" to "Plan"** group in the
   picker — it's inherently future-facing (prepping for an upcoming
   meeting), doesn't belong in day-to-day capture.
3. A few small UI naming/behavior corrections happened along the way
   this session (details in commit messages, not repeated here):
   OPTIMIZE/FAIL emoji tweaks, category reordering with negatives at
   bottom of each group, "Stall" renamed to "Postpone", CAREER moved
   from "plan" to "reflect" group, KUDOS help text broadened, Ditto
   button now logs `ONGOING` instead of repeating the last category,
   added a `mins` field with `@Nm` informal time-tracking syntax,
   fixed a Tab-order bug (root cause: `container.NewBorder` reorders
   objects internally in a way that breaks Fyne's focus traversal --
   documented via `stretchrow.go`'s custom layout, used specifically
   to preserve both correct Tab order AND NewBorder-style stretch
   behavior for the main input field).

## Known unresolved issue

**Tag autocomplete (FR-10) has a reported-but-unconfirmed-fixed bug**:
typing `#` + first char shows a suggestion popup, but focus reportedly
gets stuck in the popup (can't type a second character without hitting
Esc first). Two fix attempts were made (`6a1fcbc`, `f33f568`) —
synchronous `canvas.Focus(input)` right after showing the popup, then
deferring that call via `fyne.Do`. **Neither has been confirmed
working by the user yet** — this needs retesting. If still broken,
next step is likely abandoning `widget.PopUpMenu` (which unconditionally
calls `canvas.Focus(p)` in its `Show()`, seemingly not overridable
cleanly) in favor of a custom non-focus-stealing suggestion widget
(e.g. a plain `container.NewVBox` of clickable labels shown/hidden
manually, positioned under the input, never taking focus itself).

## Immediate next step (where to pick up)

Ask the user whether to:
1. Retest the tag-autocomplete focus bug fix first (quick, isolated).
2. Continue Phase 3 with **FR-14** (Start-of-Month wizard — multi-step:
   prior-month digest, IDEA/SOMEDAY cleanup, IMPACT/MILESTONE prompts,
   new month's GOALs).
3. Decide FR-12's fate (separate feature vs. considered done via
   FR-11's re-scoping).
4. Move to Phase 4 (reporting/review) or Phase 5 (speculative/
   deferred) per the FRD's phase table if Phase 3 feels "good enough. "

## Working agreement (unchanged from prior session, still in effect)

- One commit per FR item, or small logical chunks combined when they
  naturally belong together (explicitly OK'd by user this session —
  no need to force strict 1:1 FR-to-commit mapping).
- Always run `make build && go vet ./...`, `go test ./...`, and check
  `editor_diagnostics` after edits, before every commit.
- Terminal-notifier fires automatically at the end of sizeable work
  (new global ECA rule added this session:
  `~/proj/eca-config/rules/terminal-notifier-on-completion.md`,
  committed and pushed).
- Lots of small checkpoints; ask before big-scope changes (per
  AGENTS.md). This session included several such checkpoints (e.g.
  confirming the FR-09 EOD-hookup approach, confirming Meeting Prep's
  re-scope details, confirming Phase 2 scope before starting).
- Manual GUI testing is done by Micah clicking around — always
  describe exactly what to click/verify after each change, since the
  agent cannot run/see the actual Fyne UI.

## Full FR list location

See `FRD-DRAFT-2026-08-28.md`'s "Summary table" section for all 36
items at a glance with phase and dependency notes. The FR-11/FR-12
sections there now have dated inline notes reflecting this session's
re-scoping — read those before resuming meeting-prep-adjacent work.
