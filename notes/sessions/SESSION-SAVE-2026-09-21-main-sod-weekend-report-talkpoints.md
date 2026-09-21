# Session Save — 2026-09-21 — main

## Goal

Fix SOD DOING carry-forward across weekends, remove misleading SOD copy, and
make generated reports handle dates, excluded tags, tags, and people better.

## Completed

- SOD carry-forward now skips weekends and configured off-days when selecting
  its source day, so a Friday DOING survives into Monday even when weekend
  plan entries exist.
- Added Friday-to-Monday DOING regression coverage and updated carry-forward
  documentation.
- Standup summaries now filter excluded tags from both completed items and the
  currently-open TODO/DOING/GOAL context. Monday prompts say “Friday”.
- Added shared structured talking-point sections for tags and people, with
  bold Markdown markers and deterministic grouping, across Standup, Summary,
  Status, Review, Annual Review, and EOD report paths.
- EOD reports now retain a deterministic talking-points section after AI
  generation.
- Updated SOD wording: editing context to TODO/DOING activates it today, and
  removed the Daybook editing instruction from “Carried into”.
- Repaired conflict markers in the active sibling `mydunnits/config.toml`,
  retaining DOING, the full report exclusion list, and both recurring-item
  variants. The file now parses as TOML.

## Files

- `dun/carryforward.go`, `dun/carryforward_test.go`
- `dun/standup.go`, `dun/standup_categories_test.go`
- `dun/reportmentions.go`
- `dun/periodreport.go`, `dun/period.go`, `dun/statusreport.go`
- `dun/dailysummary.go`, `dun/eod.go`
- `dun/summarize_test.go`, `dun/periodreport_test.go`
- `dun/sod.go`
- `docs/GUIDE.md`, `docs/todo-carryforward-design.md`
- sibling repo: `/home/mde/proj/mydunnits/config.toml`

## Verification

- `go test ./... -count=1` — passed
- `make build` — passed
- `make vet` — passed
- `git diff --check` — passed
- sibling config parsed with Python `tomllib`

## Status

Changes are uncommitted and unstaged on `main`. The sibling `mydunnits`
worktree also contains pre-existing ledger changes plus the repaired config;
nothing was staged or committed.

DONE command:

```sh
~/proj/dunnit/dunnit DONE 'Fix weekend SOD carry-forward and report talking points'
```

Suggested commit message:

```text
Fix weekend SOD carry-forward and report talking points
```
