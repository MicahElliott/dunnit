# Session Save — 2026-09-21 — main

## Goal

Reorder Daybook's lifecycle items and replace the single logging streak with
positive, rotating achievement callouts.

## Completed

- Daybook's Planned section and category picker now put DOING above TODO.
- Added twelve ledger-derived achievement indicators covering logging,
  throughput, Hilites, WINs, learning, people, tickets, projects, explicit
  time, repeated tags, and sustained tag continuity.
- The normal TODO -> DOING -> DONE flow is intentionally not an achievement
  by itself.
- Start of Day detects the full set, shows at most two randomly selected
  examples, and opens a separate window listing every qualifying achievement.
- Ticket detection accepts numeric markers and prefixed markers such as
  `#SCRUM-12345`.
- Updated `AGENTS.md` with the actual `dun/` package layout and these durable
  data-model/streak rules.

## Files

- `AGENTS.md`
- `dun/categories.go`, `dun/categories_test.go`
- `dun/todos.go`
- `dun/streak.go`, `dun/streak_test.go`
- `dun/sod.go`

## Verification

- `go test ./...` — passed
- `make build` — passed
- `make vet` — passed
- `gofmt` — passed
- `git diff --check` — passed

## Status

Changes are uncommitted and unstaged on `main`. Manual UI verification remains:
open Start of Day with qualifying ledger data, confirm the compact summary shows
at most two examples, and click the achievements button to inspect the full
list.

The required DONE command exited successfully but printed Fyne's existing
locale warning because this shell uses the `C` locale.

DONE command:

```sh
~/proj/dunnit/dunnit DONE 'Add rotating Daybook achievement callouts'
```

Suggested commit message:

```text
Add rotating Daybook achievement callouts
```
