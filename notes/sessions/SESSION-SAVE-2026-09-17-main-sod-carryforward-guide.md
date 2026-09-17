# Session handoff — 2026-09-17 — `main`

## Goal

Clarify what carries into a new day, repair duplicate/stale SOD items and
scrolling, improve the EOD report handoff, and document the user-facing rules.

## Completed

- Carry-forward remains limited to unresolved TODO/DOING items from the
  newest qualifying prior day within seven calendar days.
- Resolution records now suppress later synced copies of the same carried
  item, preventing old DONE/SOMEDAY work from resurfacing. A newly typed
  unmarked TODO can intentionally reopen it.
- SOD stale review deduplicates logical items, shows each original `since`
  date, and is bounded to the previous 30 calendar days. Items remain active
  in Daybook until completed, postponed, or discarded.
- WAITING, GOAL, RISK, QUESTION, and FIXME remain SOD context and are not
  copied into today's plan.
- SOD now has one outer vertical scroll; fitted plan/context sections no
  longer compete with nested scrollbars.
- Inline EOD report content was replaced by the full-report button. The
  standalone report has a date-correct larger title and an italic Stats line
  directly below it.
- Added the user-facing carry-over explanation to `docs/GUIDE.md` and kept
  `docs/todo-carryforward-design.md` aligned with the implementation.

## Files changed

- `docs/GUIDE.md`
- `docs/todo-carryforward-design.md`
- `dun/carryforward.go`, `dun/carryforward_test.go`
- `dun/eod.go`, `dun/eod_test.go`
- `dun/report.go`
- `dun/sod.go`
- `dun/todos.go`, `dun/todos_test.go`

## Verification

- `go test ./... -count=1` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Manual Fyne UI click-through remains for the user: verify SOD with a
  populated carry section, stale items, WAITING context, and the full EOD
  report button.

## Status

Changes are staged on `main` but not committed. No unrelated worktree files
were found.

DONE command:

`~/proj/dunnit/dunnit DONE 'Clarify SOD carry-forward rules and report UI'`

Pasteable commit message:

`fix: clarify SOD carry-forward and EOD report behavior`
