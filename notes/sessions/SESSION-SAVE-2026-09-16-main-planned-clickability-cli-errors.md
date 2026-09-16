# Session save — 2026-09-16 — Planned clickability and CLI errors

## Goal

Fix the recurring first-click failure on tiny Planned-section action buttons
and make the `dunnit` CLI return a non-zero status when a ledger write fails.
Diagnose the Codex configuration needed to write the configured ledger repo.

## Completed

- Replaced Planned row action controls (Discard, Postpone, Done, Start, and
  Edit) with plain Fyne icon buttons. These controls no longer create the
  full-canvas hover tooltip overlay that could consume the first click while
  the button was unfocused.
- Changed `RecordActivity` and its internal ledger writer to return wrapped
  errors from directory creation, file open, write, and close operations.
- Changed the CLI to print the write error and return status `1` when the
  ledger cannot be written.
- Added CLI tests for a successful write and a blocked ledger root.
- Confirmed the additional Codex setting needed for the sibling ledger repo:
  add `/home/mde/proj/mydunnits` to `[sandbox_workspace_write]`
  `writable_roots`, preserving the existing Go cache path. Restart Codex
  after changing the user config.

## Files changed

- `dun/ui.go`
- `dunnit.go`
- `dunnit_test.go`

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Built binary returned `0` for a writable CLI write and `1` for a blocked
  ledger root.
- Manual desktop click-through remains: open Planned, leave focus in another
  control, and click Done once to verify the completion dialog opens.

## Repository state

- Branch: `main`.
- Source changes and this session note are uncommitted.
- The generated `dunnit` binary remains ignored by Git.

## Known problems

- The active Codex session still has the older restricted filesystem profile;
  `/home/mde/proj/mydunnits` was not writable until a new session uses the
  updated configuration.
- The required completion DONE entry was attempted and returned status `1` as
  intended because `/home/mde/.config/dunnit` is read-only in the active
  environment. The desktop notification was also blocked by DBus permissions.
