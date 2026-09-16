# Session Save — 2026-09-16 — Main — Inline Entry Links

## Goal

Add clickable Jira, Teams, and general web links to Dunnit entries while
keeping the ledger plain text and removing distracting link underlines.

## Completed work

- Added shared parsing for labeled Markdown links and bare `http`/`https`
  URLs.
- Preserved the original entry text and URL in the ledger.
- Added derived labels for common Jira/Atlassian, GitHub, Teams, Slack,
  Google, Notion, Linear, Asana, Trello, Zoom, Figma, Dropbox, SharePoint,
  Office, and YouTube URLs; unknown bare URLs display as `[link]`.
- Made entry links small, blue, clickable, and tooltip-backed with the full
  URL.
- Removed the underline from frecent tag links.
- Applied the shared entry renderer across Daybook, Start/End of Day,
  kickoff views, SOMEDAY, Search, and Navigator.
- Documented the syntax in `docs/GUIDE.md`.

## Files changed

- `dun/links.go`
- `dun/links_test.go`
- `dun/itemrow.go`
- `dun/taglink.go`
- `dun/ui.go`
- `dun/eod.go`
- `dun/sod.go`
- `dun/search.go`
- `dun/navigator.go`
- `dun/somedaybrowser.go`
- `dun/monthkickoff.go`
- `dun/periodkickoff.go`
- `docs/GUIDE.md`

## Decisions

- Markdown remains the canonical labeled-link syntax:
  `[Jira #74750](https://example.atlassian.net/browse/ABC-74750)`.
- Bare URLs are accepted as a convenience and receive a service-aware label
  when the host and path make one obvious.
- Only HTTP(S) destinations are accepted by the entry renderer.
- Link labels replace raw URLs visually; the full destination remains in the
  source ledger and appears in the hover tooltip.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

## Known problems

- GUI click-through and visual spacing still need human verification in a
  running Fyne window; no automated desktop interaction was available.
- The DONE command printed Fyne's existing `C`-locale parsing warning but
  exited successfully and wrote the entry.

## Repository state

- Branch: `main`
- Work is uncommitted. Do not stage or commit automatically.
