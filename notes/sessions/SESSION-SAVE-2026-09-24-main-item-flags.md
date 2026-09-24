# Session save — 2026-09-24 — main item flags

## Goal

Add fixed item flags for TODO/DOING entries, with compact Daybook display,
important-item ordering, editing controls, and documentation.

## Completed

- Added standalone `!!`, `??`, `@@`, and `++` flags in `dun/flags.go`.
- Parsed flags into `LedgerEntry.Flags` and added flag filtering to
  `LedgerQuery`.
- Preserved raw flag tokens in ledger text while ignoring them for lifecycle
  identity, resolution matching, and carry-forward deduplication.
- Preserved and canonicalized flags during TODO/DOING lifecycle transitions.
- Rendered flag icons after the Daybook primary tag with hover help.
- Sorted `!!` items before ordinary items in open-item displays.
- Added flag toggles to the Edit Entry dialog for adding and removing flags.
- Documented the syntax and meanings in `README.md`.
- Added table-driven parser, lifecycle, and sorting tests.

## Decisions

- `@@` means follow up with someone; an `@Person` reference is optional.
- Flags are fixed and non-customizable for now.
- Flags are standalone whitespace-delimited tokens and may appear anywhere in
  an entry. Lifecycle rewrites store them in stable order at the start of the
  entry text.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

GUI interaction still needs the human check: open Dunnit, edit a TODO, toggle
each flag, save, and confirm the icon placement, important ordering, and
removal behavior.

## Commit message

feat: add fixed item flags for planned work
