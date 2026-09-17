# Session handoff — 2026-09-17 — `main`

## Goal

Repair repeated Ditto duration increments and clarify the surrounding
duration-entry behavior.

## Completed

- Ditto now adds the effective configured Nudge interval on every click,
  including when the DOING entry initially has no duration.
- The scheduler and Ditto share the same 60-minute fallback for missing or
  invalid legacy configuration.
- The Daybook minutes field is cleared when categories change and is ignored
  for non-time-trackable categories, preventing a hidden value from leaking
  onto TODO/GOAL entries.
- Settings now rejects zero, negative, or nonnumeric Nudge intervals.
- Added regression coverage for repeated 30-minute Ditto clicks and the
  interval fallback.

## Files changed

- `dun/config.go`
- `dun/config_test.go`
- `dun/sched.go`
- `dun/settings.go`
- `dun/todos_test.go`
- `dun/ui.go`

## Decisions

- Ditto counts only explicit clicks. Snoozed or skipped prompts do not add
  time.
- Carry-forward preserves an item's cumulative duration across days.
- A late Ditto does not reset the scheduler's separate last-activity timestamp;
  that remains a possible future scheduling refinement.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Manual UI verification remains: enter minutes, switch categories, confirm
  the field clears, then use Ditto repeatedly and confirm each click adds the
  configured Nudge interval.

## Status

The six code/test files are staged and uncommitted on `main`; this handoff
note is new and untracked. No commit was made.

DONE command:

`~/proj/dunnit/dunnit DONE 'Fix repeated Ditto duration increments and minutes handling'`

Pasteable commit message:

`fix: make Ditto accumulate configured nudge intervals`
