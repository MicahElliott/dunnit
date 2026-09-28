# Session save: reports, metrics, and review workflows

Date: 2026-09-27
Branch: `main`
Commit: `6b57f2a Improve reports, a lot`

## Goal

Implement the agreed redesign of Saved Reports and Metrics, reduce report-menu redundancy, and unify Status/Annual Review behavior with the Review workflows.

## Completed

- Reworked Saved Reports into a read-only browsing library with type, audience, preset/custom date, and text filters.
- Grouped report variants and added explicit Preview and Edit actions.
- Replaced the old Trend View with structured Metrics tables, fixed trailing ranges, custom dates, missing-day visibility, summaries, and exclusion-tag filtering.
- Unified Week Review with private/shareable audience selection and share-safe input filtering.
- Removed redundant Status Report and Annual Review menu entries; Year Review is now the replacement, with a one-time config migration for existing users.
- Kept legacy saved report files readable and grouped with their newer equivalents.
- Updated the related report and navigation documentation and added focused metadata, grouping, and metrics tests.

## Verification

- `go test -count=1 ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

## Known limitation

No manual Fyne GUI click-through was performed in this environment. Desktop notification delivery was attempted, but the local notification service denied the connection.
