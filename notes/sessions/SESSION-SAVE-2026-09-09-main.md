# Session handoff — 2026-09-09 — main

## Goal

Establish a lightweight feature backlog and evaluate whether go-git is worth
adding for future ledger synchronization; reduce the Dunnit executable size.

## Completed

- Added the living backlog in `BACKLOG.md`, leaving the long FRD draft as a
  historical requirements artifact.
- Recorded the TODO-to-DONE bug, window-title consistency, shared list UI,
  disappearing Quit action, yesterday's DONE editing, Git bootstrap/sync,
  Settings-based ledger directory configuration, and future Turso/SQLite work.
- Tested `github.com/go-git/go-git/v5` (v5.19.2). It was removed after the
  experiment because it added substantial transitive dependencies and no sync
  implementation yet.
- Updated `Makefile` so `make build` uses `-trimpath -ldflags "-s -w"`.

## Size measurements

On Linux:

- Existing unstripped build: 33.5 MB.
- Existing stripped build: 25.0 MB.
- With go-git linked, unstripped: 37.3 MB.
- With go-git linked, stripped: 27.7 MB.

The current checked-out `dunnit` is 24,975,496 bytes and is stripped.

## Verification

- `go vet ./...` passed during the session.
- `git diff --check` passed.
- The final stripped binary is present and reports as stripped.

## Known state

- Branch: `main`.
- Latest commit: `f0ffac6 Fix SIGSEGV`.
- The worktree has one untracked file, `tufa.log`; it was left untouched.
- The repository is otherwise clean. Existing migration work is already in the
  committed history and was not altered during this session.

## Next step

Start with the TODO-to-DONE regression, then implement Git sync by shelling out
to the system `git` executable; keep go-git out until a concrete requirement
justifies its size and dependency cost.
