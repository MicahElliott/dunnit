# Session save: local file links and Settings UI

Date: 2026-09-21
Branch: `main`

## Goal

Add readable Markdown references to local files, with `dunnit:` and alias
forms, and make the path configuration discoverable in Settings.

## Completed

- Added local link resolution for relative paths, `dunnit:`, `file:`, and
  explicit aliases such as `cc3:docs/foo.txt`.
- Added ordered `file_search_path` roots; each root's final directory name is
  inferred as an alias.
- Added the Settings field **Project Folders for Links**, accepting a
  space-separated list such as `~/work/cc3 ~/work/kp`.
- Local links in ledger rows and report previews open with `$EDITOR`, falling
  back to the platform file opener.
- Added root confinement for `dunnit:` and aliases, documentation, config
  round-trip coverage, resolver tests, and report-preview link coverage.

## Files changed

`dun/links.go`, `dun/taglink.go`, `dun/itemrow.go`, `dun/report.go`,
`dun/settings.go`, `dun/config.go`, related tests, `README.md`, and
`docs/GUIDE.md`.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Manual UI verification remains: open Settings, set Project Folders for
  Links, save, and click an alias link.

## Commit summary

`Add configurable local file links`

Work remains uncommitted on `main`; the user will stage and commit manually.
