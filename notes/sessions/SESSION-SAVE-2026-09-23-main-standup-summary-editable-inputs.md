# Session save — 2026-09-23 — editable Standup Summary inputs

## Goal

Make the Standup Summary pre-generation window accurately describe and expose
every input that reaches the generated report.

## Completed

- Renamed the source window to `Standup Summary pre-generation seed`.
- Changed midnight wording from `Tue midnight` to `start of Tue`.
- Verified the fallback boundary is the source date at exactly `00:00`.
- Split the editable source text into two sections:
  - completed/notable items;
  - open TODO/DOING/GOAL items for today.
- Preserved open-item categories with `[TODO]`, `[DOING]`, and `[GOAL]`
  markers, and parsed both sections back into separate report inputs.
- Removed the hidden-input behavior: deleting or editing an open item now
  changes what the report receives.
- Allowed generation when the completed/notable section is empty but open plan
  items remain.

## Files changed

- `dun/standup.go`
- `dun/standup_categories_test.go`

## Verification

- `go test ./... -count=1`
- `make build`
- `make vet`
- `gofmt`
- `git diff --check`

All passed.

Manual Fyne click-through remains: open Standup Summary, verify both editable
sections, delete or reword an open item, and confirm the generated report uses
the edited source.

## Repository state

- Branch: `main`
- Source changes are uncommitted, as required by the session workflow.
- Existing untracked file `notes/sessions/SESSION-SAVE-2026-09-23-main-eod-model-sod.md`
  was preserved.
