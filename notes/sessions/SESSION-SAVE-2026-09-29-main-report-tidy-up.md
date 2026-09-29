# Session save: report tidy-up ordering and untagged marker

Date: 2026-09-29
Branch: `main`
Commit: uncommitted

## Goal

Make report preparation happen before report seed/form editing, and make
forgotten tags visible during tidy-up without changing ledger data.

## Completed

- Standup Report now opens the shared Quick tidy-up handoff before building its
  editable seed, so the seed reflects Daybook corrections.
- End-of-Day now uses the same ordering; its editable form is created after
  tidy-up, and its Generate action no longer opens a second preparation window.
- Daybook rows without a real tag show a gray, hover-explained
  `[#UNTAGGED]` display marker. The marker is presentation-only and does not
  enter ledger text, tag history, or exclusion filters.

## Files changed

- `dun/standup.go`
- `dun/eod.go`
- `dun/itemrow.go`

## Verification

- `make build` passed.
- `make vet` passed.
- `go test ./...` passed.
- `git diff --check` passed.

## Known status

- Manual Fyne click-through was not available in this environment. The human
  check should open Quick Standup and End of Day, confirm tidy-up appears
  first, finish or skip it, and confirm the seed/form then opens with current
  ledger data. Confirm an untagged Daybook row shows `[#UNTAGGED]` and that
  editing it with a real tag removes the marker.
- Desktop notification delivery failed because the local notification service
  denied the `notify-send` connection.
- The required DONE command succeeded and printed the existing Fyne `C` locale
  parsing warning.
- Changes remain uncommitted. No commit was made by this session.

DONE command:

`~/proj/dunnit/dunnit DONE 'Fix report tidy-up ordering and mark untagged Daybook rows'`
