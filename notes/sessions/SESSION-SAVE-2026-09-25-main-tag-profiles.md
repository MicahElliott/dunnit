# Session save — 2026-09-25 — main tag profiles

## Goal

Add optional, editable profiles for ledger tags without introducing a `TAG`
ledger category or moving descriptive metadata into the append-only activity
history.

## Completed

- Added `$DUNNIT_DIR/tags.toml` profile storage using the existing TOML
  dependency.
- Added profile fields for title, summary, Markdown description, URL, kind,
  status, aliases, and parent.
- Normalized profile names and aliases to bare tag names while continuing to
  display and use `#foo` in the ledger UI.
- Extended the existing tag-history window with profile details above the
  recent 30-day activity list.
- Added a Define/Edit Tag Definition dialog. Clicking a used tag can create a
  new profile; later clicks edit it.
- Added TOML round-trip, alias lookup, normalization, and preservation tests.
- Documented the file format and current-state/activity separation in the
  README and project guidance.

## Files

- `dun/tagdefs.go` — profile model, TOML persistence, lookup, and editor UI.
- `dun/tagdefs_test.go` — profile persistence and normalization tests.
- `dun/tags.go` — profile section in the tag history window.
- `dun/config.go`, `AGENTS.md`, `README.md` — data-root documentation.

## Decisions

- Store the definition key without `#`; treat `#` as ledger syntax.
- Resolve definitions and aliases case-insensitively, while preserving ledger
  spelling.
- Keep current profile metadata in `tags.toml`; derive activity from existing
  ledger categories and do not add `TAG` entries.
- Profiles are created only through a used tag's Define action for now; there
  is no standalone profile manager.

## Verification

- `go test ./...`
- `make build`
- `make vet`
- `git diff --check`

All checks passed. Changes were already staged by the user. This session note
is intentionally left unstaged for the user to include or omit as desired.

