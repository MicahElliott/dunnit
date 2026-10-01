# Session save — 2026-10-01 — main — multiline wheel and tag exclusions

## Goal

Fix mousewheel scrolling over multiline input fields, including the Tag
Definition Description field, and allow a tag profile to mark itself excluded
from reports without using Settings.

## Completed

- Added shared no-nested-scroll handling for multiline fields used inside
  scrollable forms, preserving their visible row heights with an outer field
  wrapper.
- Updated Tag Definition Description, EOD, Month Kickoff, and Month Review
  multiline form fields.
- Added `TagDefinition.Exclude`, the Tag Definition editor checkbox, profile
  metadata display, and exclusion inheritance for aliases.
- Applied profile exclusions throughout report, summary, EOD, streak, Daybook,
  standup, review, and navigator filtering paths.
- Added `dunnit tag --exclude` and `--exclude=false` support.
- Updated README and GUIDE documentation and added regression tests.

## Files changed

The implementation and tests are staged on `main`. The main files are
`dun/scroll.go`, `dun/tagdefs.go`, `dun/eod.go`, `dun/monthkickoff.go`,
`dun/monthreview.go`, `dun/summarize.go`, `dun/ui.go`, `dunnit.go`, and their
related tests and documentation.

## Decisions

Standalone multiline editors retain their own internal scrolling. Fields
inside an outer form use wrapping-off/no-inner-scroll behavior so the parent
form receives the mousewheel; explicit newlines remain supported.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

## Known limitations

The GUI interaction was not manually exercised in this environment. Verify
the Tag Definition Description wheel behavior and the Exclude checkbox in the
desktop app.
