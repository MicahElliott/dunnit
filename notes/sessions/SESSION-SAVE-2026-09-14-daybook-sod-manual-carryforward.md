# Session handoff — 2026-09-14 — Daybook Start of Day carry-forward

## Goal

Replace per-machine automatic daily carry-forward with an explicit Start of
Day planning step, while keeping synced ledger decisions consistent across
machines.

## Completed

- Start of Day now searches the prior seven calendar days newest-first and
  carries TODO/DOING entries from the newest qualifying ledger day only.
- Daybook construction, first entry recording, and ordinary ledger writes no
  longer trigger carry-forward.
- RISK, GOAL, WAITING, and QUESTION remain Start of Day context from the last
  active ledger day instead of being copied into today's plan.
- Start of Day shows the streak, last active-day reflection, EOD report when
  available, context reminders, recurring suggestions, and a quick TODO
  entry.
- TODO/DOING items at least seven days old appear in an explicit stale review;
  the user can move them to SOMEDAY. Age alone never mutates the ledger.
- The Daybook Start of Day notice remains visible until the SOD window's Done
  action records completion, and refreshes dynamically when the window closes.
- Month-boundary scheduling also opens daily planning so carry-forward is not
  skipped on the first weekday of a month.
- Updated carry-forward tests, config-safety tests, and the design note.

## Key files

- `dun/carryforward.go` — source-day lookup, daily-plan filtering, stale review
- `dun/sod.go` — planning UI, last-active context, report/reflection display
- `dun/ui.go` — removed automatic triggers and retained dynamic Daybook notice
- `dun/sched.go` — daily SOD on month-boundary mornings
- `dun/eod.go`, `dun/periodreview.go` — updated postpone semantics/comments
- `dun/config.go`, `dun/*_test.go`, `docs/todo-carryforward-design.md`

## Decisions

- Carry only TODO and DOING for now. FIXME can be reconsidered later.
- Use the ledger as the carry/resolution record; the config marker only
  controls whether the Daybook reminder has been completed.
- Do not add automatic Git sync in this change.

## Verification

- `GOCACHE=/home/mde/tmp/codex-go-cache/dunzo go test ./...` passed.
- `GOCACHE=/home/mde/tmp/codex-go-cache/dunzo go vet ./...` passed.
- `GOCACHE=/home/mde/tmp/codex-go-cache/dunzo make build` passed.
- `git diff --check` passed.
- Manual Fyne click-through remains unavailable in the headless environment.

## Known problems / next step

The implementation is committed on `main` as `8109bb8` (`Improve SOD
carry-forward process`). The worktree is clean apart from this new, untracked
handoff note. On resume, manually run Dunnit
with a prior-day TODO: open Daybook before SOD to verify the reminder and no
carry occurs, open SOD to verify the carry/context sections, click Done to
verify the reminder disappears, and exercise stale-item SOMEDAY cleanup.
