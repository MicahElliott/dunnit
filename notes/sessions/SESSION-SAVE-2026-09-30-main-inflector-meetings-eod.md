# Dunnit Session Save — 2026-09-30 — Inflector, meetings, terminology, and EOD report handoff

## Goal

Investigate the `Framing` → `Framming` edit bug, clarify the EOD generation
behavior, strengthen the week and meeting guidance in GUIDE, improve recurring
work terminology and field widths, and use abbreviated weekdays in the UI.

## Completed

- Fixed the lifecycle inflector's missing `frame` silent-e exception. Editing
  an active `Framing ...` entry now preserves `Framing ...` instead of creating
  `Framming ...`.
- Added regression coverage for `frame` participle/base conversion and short
  weekday parsing while retaining compatibility with full weekday config values.
- Expanded Meeting Prep labels and instructions. The lookback selector is now
  identified as weeks, the history box is clearly scratch-only, Save Note is
  documented as a new tagged `MEETING` entry in today's ledger, and Record
  attended writes `MEETING #tag attended`.
- Updated MEETING semantics to cover prep, attendance, live notes, and outcomes.
- Added a first-class week section to `docs/GUIDE.md`, including weekly
  Kickoff/Review, weekly ledger organization, daily carry-forward, and the
  meanings of Postpone, Done, and Discard.
- Defined “entry” as a ledger line, “item” as open planned work, and “dunnit”
  as the product name. Renamed the recurring planned-work UI to Recurring
  Plans and widened the recurring meeting and recurring plan time fields.
- Changed user-facing weekday labels to `Sun`/`Mon`/…/`Sat`; full weekday
  names remain accepted in existing settings.
- Fixed the current EOD flow so generated text has a visible status, Finalize
  Day cannot race an in-progress generation, and a successfully saved EOD
  report opens immediately after finalization.

## Files changed

`docs/GUIDE.md`, `dun/categories.go`, `dun/eod.go`, `dun/meetingprep.go`,
`dun/minicalendar.go`, `dun/monthkickoff.go`, `dun/pastverb.go`,
`dun/pastverb_test.go`, `dun/periodkickoff.go`, `dun/recurring.go`,
`dun/sched.go`, `dun/sched_test.go`, `dun/settings.go`, `dun/sod.go`,
`dun/standup.go`, `dun/standup_categories_test.go`, and `dun/ui.go`.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Fyne interactions were not manually exercised in this environment. The most
  useful checks are EOD Generate → Finalize Day → report window, Meeting Prep's
  Record attended action, and the abbreviated weekday/time fields in Recurring
  Meetings and Settings.

## Worktree and handoff

- Branch: `main`.
- All code and documentation changes remain uncommitted for manual review and
  staging.
- Desktop completion notification was attempted but the environment denied
  access to the notification service.
