# Session handoff — 2026-09-16 — main — lifecycle inflection and movement

## Goal

Restore lifecycle verb inflection after the TODO/DOING edit work, make the
Planned completion flow editable, and support moving terminal lifecycle entries
back to TODO or DOING.

## Completed

- Added base, present-participle, and simple-past lifecycle inflection:
  `TODO Send` → `DOING Sending` → `DONE Sent`.
- Edit Entry re-inflects text before opening and when its lifecycle category
  changes.
- Lifecycle Edit supports `TODO`, `DOING`, `DONE`, `FAIL`, and `WASTED`.
- Planned's Done checkmark opens Edit Entry with DONE selected, allowing the
  user to choose DONE/FAIL/WASTED and adjust the text before saving.
- Terminal entries can move back to TODO or DOING; lifecycle resolution markers
  are added or removed as needed.
- Start and Ditto now preserve the appropriate lifecycle tense.
- TODO/DOING parser identity and DONE resolution recognize inflected forms.
- Postponing or promoting a DOING item normalizes it back to future-facing
  wording.

## Files changed

- `dun/pastverb.go`, `dun/pastverb_test.go`
- `dun/todos.go`, `dun/todos_test.go`
- `dun/undo.go`, `dun/ui.go`
- `dun/somedaybrowser.go`

## Decisions

- Lifecycle transitions rewrite the existing ledger row in place, preserving
  its timestamp and accumulated minutes.
- DONE, FAIL, and WASTED all use simple past wording.
- The existing Edit modal is the single user-facing movement flow; the former
  separate Planned completion modal was removed.
- SOMEDAY remains the direct postpone flow, while Edit handles lifecycle state
  changes.

## Verification

- `go test ./...` passed.
- `go vet ./...` passed.
- `make build` passed.
- `git diff --check` passed.
- Committed in `7419b44` (`Improve inflections/editing, negative advancement`).

## Known problems

- Inflection remains intentionally lightweight and English-specific; uncommon
  verbs may remain unchanged or need another irregular spelling entry.
- Manual UI verification is still needed for the modal selector and the
  TODO/DOING/DONE movement flow.
- The desktop completion notification was blocked by the environment.

## Next step

Run the built app and manually verify: TODO Send → Start → DOING Sending →
Done → Edit Entry showing DONE/Sent, then change DONE back to TODO or DOING.
