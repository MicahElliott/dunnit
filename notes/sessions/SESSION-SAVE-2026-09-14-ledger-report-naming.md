# Session Save — 2026-09-14 — Ledger and Report Naming

## Context

The working project directory was renamed from `dunzo` to `dunnit`. The
Go build cache is independent of that filesystem directory name. The stale
`DUNZO_DIR` configuration check was removed; `DUNNIT_DIR` remains the only
environment override.

## Decisions

- Ledger files use only the canonical weekday-first format:
  `ledger-Mon-20260914.txt`.
- Legacy ledger filenames are ignored by the application. A zsh migration
  snippet was supplied for the Git-tracked `mydunnits` repository; it uses
  `git mv`, handles both date-only and older date-first/DOW names, and refuses
  destination collisions.
- Reports use descriptor-first names with covered-period and generation-date
  tokens, such as `status-w38-20260914.md` and
  `review-week-20260907-20260914-status_report.md`.
- EOD recaps are standalone `eod-Mon-20260914.md` reports. The EOD `SUMMARY`
  entry is no longer written to the ledger.
- Ledger write paths flatten embedded newlines, including ordinary entries,
  undo rewrites, and tomorrow's carry-forward writes. Report Markdown keeps
  its line breaks.
- Automatic EOD drafting requires at least three `DONE` entries. Explicit
  EOD entry and manual report generation remain available below that count.
- Old `dsu-*` and `summary-*` report filenames are no longer recognized.

## Kickoff and Review clarification

The existing design treats Kickoff as a forward-looking interaction: it reads
back current goals/open items/recurring or OKR context, solicits planning input,
and writes resulting items such as `GOAL` to the ledger. Review is the
backward-looking workflow that generates standalone Markdown reports.

Month's former combined SOM flow is already split into Month Kickoff and Month
Review. The naming changes did not remove Day, Week, Month, Quarter, or Year
Kickoff/Review workflows. Kickoffs do not currently have saved report files;
their output is ledger planning data.

## Implementation

- Canonical ledger parsing and path generation were updated in `paths.go` and
  `summarize.go`.
- Report naming, indexing, Review discovery, Status Report, Standup, and EOD
  paths were updated to the new descriptors.
- Reports Library now recognizes current `review-*`, `standup-*`, `status-*`,
  and `eod-*` families only.
- Documentation and tests were updated, including newline, filename, report
  parsing, and EOD threshold coverage.

## Validation

Passed:

- `GOCACHE=/tmp/dunnit-go-cache go test -count=1 ./...`
- `GOCACHE=/tmp/dunnit-go-cache go vet ./...`
- `git diff --check`
- migration snippet syntax check and a temporary Git worktree rename test
- ordinary `go build` to `/tmp/dunnit-build-test`

The exact `make build` command was attempted with a writable cache, but its
`-trimpath` build stalled while rebuilding Fyne/OpenGL CGo dependencies in the
execution environment. No changed-code compile error was reported.

Suggested commit subject:

```text
feat: standardize ledger and report file naming
```
