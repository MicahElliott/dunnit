# Session save — 2026-09-21 — main — SOD/SOW carry-forward and UI fixes

## Goal

Repair the recent SOD/report UI problems, make larger-period Kickoffs useful,
and verify whether DOING items carry into today's Daybook.

## Completed

- Reduced and vertically centered category and person icons in shared item
  rows; recurring TODO suggestions now render the category pushpin/icon.
- Restored SOD's Delete, Postpone, and Done actions on every daily-plan row,
  including freshly carried TODO/DOING items.
- Added actionable open-item rows, clearer headings, top Dismiss actions,
  Trend View and Reports Library references, and prior-review references to
  generic Week/Quarter/Year Kickoffs and Month Kickoff goals.
- Added extra spacing above level-2 headings in generated report viewers and
  live previews without changing saved Markdown.
- Confirmed the carry-forward algorithm already preserves DOING categories.
  Added end-to-end TODO/DOING regression coverage.
- Fixed the related UI lifecycle hole: closing SOD through the window close
  control now refreshes an already-open Daybook, so newly carried rows appear
  immediately even when Done is not tapped.

## Files changed

`dun/carryforward_test.go`, `dun/eod.go`, `dun/itemrow.go`,
`dun/monthkickoff.go`, `dun/monthreview.go`, `dun/periodkickoff.go`,
`dun/recurring.go`, `dun/report.go`, `dun/sod.go`, and `dun/tightrow.go`.

## Verification

- `go test ./... -count=1` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Manual Fyne UI verification remains: open SOD with prior-day TODO and
  DOING entries, close it with both Done and the window close control, and
  confirm the open Daybook shows both carried rows and their actions. Check
  recurring TODO pushpins and generated-report section spacing visually.

## Status

Changes are uncommitted and unstaged on `main`. No unrelated worktree files
were found.

DONE command:

`~/proj/dunnit/dunnit DONE 'Fix SOD DOING carry-forward refresh and kickoff report UI'`

Pasteable commit message:

`fix: repair SOD carry-forward and period kickoff UI`

