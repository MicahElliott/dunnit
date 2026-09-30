# Session save — 2026-09-29 — main — tag form, CLI, and history fixes

## Goal

Fix Tag Definition form scrolling and field ergonomics, add a simple CLI path
for agent-assisted tag profile population, and correct duplicate or iconless
recent activity rows.

## Completed

- Added a shared single-line Fyne Entry configuration that avoids the nested
  input scroller intercepting wheel events in surrounding scrollable forms.
- Applied that helper to all ordinary single-line entries across Dunnit.
- Replaced Tag Definition Kind and Status text fields with canonical selectors;
  current stored values remain visible when editing a profile.
- Added `dunnit tag TAG` with incremental profile flags for title, summary,
  description, URL, kind, status, parent, and repeatable aliases.
- Documented the connector-free workflow for harnesses that already have their
  own Jira/GitHub/tracking-system access.
- Required valid ledger timestamps when parsing rows. This prevents Markdown
  continuation lines in SUMMARY entries from becoming fake recent activity
  rows.
- Folded unmarked resolutions into existing carried-forward tag lineages and
  verified the live `#74847` history now shows one logical current row.
- Kept the existing generic bullet fallback for category codes outside the
  current registry; no historical category vocabulary was added.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Manual GUI verification remains for a human: open Tag Definition, wheel over
  each single-line field while it is unfocused, and confirm the outer form
  scrolls; confirm the multiline Description editor still scrolls its own
  content.

## Status

Work remains uncommitted on `main`; the corrected implementation and this note
are staged, but nothing has been committed.
The CLI emits Fyne's existing non-blocking `Error parsing user locale C`
warning in this environment before completing successfully.

Pasteable commit message:

`Fix tag form scrolling, profile CLI, and recent activity history`
