# Session save — 2026-09-23 — report titles and Go toolchain

## Goal

Give every generated report a meaningful Markdown title for email and other
paste destinations, then update the module's Go requirement to the latest
stable release.

## Completed

- Added shared report-title normalization that guarantees a canonical H1.
- Added dated titles to generated and deterministic Standup output.
- Added audience, ISO week, and date-range titles to Status reports.
- Made EOD saved and copied output use its dated title and report details.
- Updated `go.mod` and the README requirement to Go `1.27.1`.
- Preserved the existing period-review title behavior through the shared
  normalizer.

## Files changed

- `dun/report.go`
- `dun/periodreport.go`
- `dun/standup.go`
- `dun/statusreport.go`
- `dun/eod.go`
- `dun/dailysummary.go`
- `dun/report_test.go`
- `dun/standup_categories_test.go`
- `go.mod`
- `README.md`

## Verification

- Before the Go requirement bump: `make build`, `make vet`, and `go test ./...`
  passed.
- After changing the module to Go `1.27.1`, `GOTOOLCHAIN=local` correctly
  stopped build, vet, and tests because the installed toolchain is Go `1.25.2`.
- `git diff --check` passed.
- Manual Fyne click-through remains: generate Standup and Status reports,
  copy both Markdown and rich text, and verify the dated/ranged H1; generate
  and save an EOD report and verify its dated H1.

## Repository state

- Branch: `main`
- User staged the source changes; no commit or push was performed.
- This session note is newly created and remains unstaged.
