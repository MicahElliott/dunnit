# Session save — 2026-09-18 — main — people and duration markers

## Goal

Make people a first-class ledger trackable using `@Name`, move duration
metadata from `@N[mhd]` to `~N[mhd]`, and provide a one-time migration for
existing ledgers.

## Completed

- Switched people parsing, filtering, Daybook highlighting, person indicators,
  and the Frequent people insertion row from `&Name` to `@Name`.
- Switched duration parsing, display metadata, lifecycle accumulation, UI
  writes, CLI examples, and tests from `@N[mhd]` to `~N[mhd]`.
- Deliberately kept no dual syntax path; old `@10m` is no longer parsed as a
  duration.
- Added `scripts/migrate-people-and-duration-markers.zsh`, which rewrites
  `&Person` to `@Person` and `@N[mhd]` to `~N[mhd]` only in ledger files.
- Updated README, GUIDE, Navigator design notes, and the backlog item for
  person-aware feedback/collaboration rollups.

## Files changed

Documentation: `README.md`, `docs/GUIDE.md`, `docs/navigator-design.md`, and
`BACKLOG.md`.

Implementation and tests: `dun/people.go`, `dun/people_test.go`,
`dun/ledgerentry.go`, `dun/ledgerquery.go`, `dun/itemrow.go`, `dun/ui.go`,
`dunnit.go`, `dun/carryforward_test.go`, `dun/itemrow_test.go`,
`dun/pastverb_test.go`, and `dun/todos_test.go`.

Migration: `scripts/migrate-people-and-duration-markers.zsh`.

The user had already staged part of the prior people-trackable work. The
working tree now contains both staged and unstaged portions; none were staged,
unstaged, reset, or committed by this session.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `zsh -n scripts/migrate-people-and-duration-markers.zsh` passed.
- The migration transformation was checked against person markers, durations,
  `AT&T`, email addresses, and ordinary ampersands.

## Known problems / next step

The migration script has not been run against the user's ledger directory.
Run it once before using the new build:

```sh
./scripts/migrate-people-and-duration-markers.zsh /path/to/mydunnits
```

Manual UI verification remains: enter a `DONE` with minutes and confirm the
ledger uses `~20m`; enter a line with `@Person` and confirm the Daybook shows
the `👤` indicator and Frequent people entry.

## Commit blurb

feat: switch people and duration trackable markers
