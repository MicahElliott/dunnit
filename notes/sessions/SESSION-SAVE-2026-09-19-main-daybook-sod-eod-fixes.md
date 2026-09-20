# Session save — 2026-09-19 — main — Daybook, SOD, and EOD fixes

## Goal

Fix Daybook person-marker alignment, normalize stale indicators, repair the
Start of Day workflow, reduce new ledger timestamps to minute precision, and
make EOD reports retain important ledger details.

## Completed

- Render person markers inline: the display replaces `@` with a colored,
  text-style person icon beside the person name while preserving raw ledger
  text.
- Replace platform emoji age markers with consistently sized, flat colored
  dots for yellow, orange, and red age bands.
- Merge SOD carried and stale TODO sections with deduplication. Stale rows
  retain Delete/Postpone/Done actions and display the red dot instead of a
  long `since` date.
- Add SOD context-row Edit and Delete actions. Historical rows now retain
  their source ledger path and line index, so editing can change WAITING,
  RISK, and other context categories into TODO/DOING and carry the result
  into today's plan.
- Add modest top/bottom heading spacing in SOD and replace recurring-item
  Add buttons with `+` icons that show Add on hover.
- Write new entries as `[HH:MM]`; parsing still accepts legacy
  `[HH:MM:SS]` records. Search and navigator display timestamps at minute
  precision as well.
- Strengthen the EOD prompt with category vocabulary and completeness rules.
  Add deterministic Completed and Learnings sections that preserve every
  DONE and TIL entry, plus a rules-based people/topics sentence.
- Add focused regression coverage for timestamp compatibility, historical
  edits, flat age markers, EOD facts, and preservation of DONE/TIL details.

## Files changed

`dun/carryforward.go`, `dun/dailysummary.go`, `dun/eod.go`,
`dun/eod_test.go`, `dun/itemrow.go`, `dun/itemrow_test.go`,
`dun/ledgerentry.go`, `dun/navigator.go`, `dun/recurring.go`,
`dun/search.go`, `dun/sod.go`, `dun/standup.go`,
`dun/standup_categories_test.go`, `dun/todos.go`, `dun/ui.go`,
`dun/undo.go`, and `dun/undo_test.go`.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --cached --check` passed before saving this note.
- Visual UI verification remains a human step: open SOD with carried/stale
  items and context rows, check inline person markers and flat dots in
  Daybook, inspect recurring `+` buttons, create a minute-only entry, and
  generate an EOD report containing repeated DONE and TIL entries.

The implementation is staged by the user. This session note is intentionally
unstaged.

## Commit blurb

fix: repair Daybook SOD and EOD report details
