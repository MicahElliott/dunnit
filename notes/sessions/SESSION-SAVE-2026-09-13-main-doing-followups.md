# Session handoff — 2026-09-13 — main — DOING follow-ups

## Goal

Finish the DOING lifecycle plan and address the follow-up UI, completion, and
sync questions raised after implementation.

## Completed

- DOING is a visible Planned state with TODO → DOING → DONE lifecycle support,
  carry-forward collapsing, duration aggregation, and legacy ONGOING history
  compatibility.
- The top Ditto control now targets only the latest active DOING item and
  displays it as the current last item.
- Settings refreshes the existing Daybook category picker immediately after
  favorites change; restart is no longer needed.
- Tooltip overlays forward clicks to both hover buttons and the category
  selector, removing the extra click previously needed after a hover.
- Planned completion opens an editor where the user can edit text and choose
  DONE, FAIL, or WASTED. Lifecycle rows retain their timestamp and resolve via
  a parser-owned source marker.
- Sync commit messages include the oldest and newest added ledger entry times.
  `git add -A` continues to include untracked ledger and report files.

## Key files

- `dun/ui.go`, `dun/hoverbutton.go`, `dun/hovercategory.go`
- `dun/undo.go`, `dun/todos.go`, `dun/gitsync.go`
- `dun/*_test.go`, `docs/GUIDE.md`, `docs/category-taxonomy.md`

## Verification

- `GOCACHE=/tmp/dunzo-gocache go test ./...` passed.
- `GOCACHE=/tmp/dunzo-gocache go vet ./...` passed.
- `gofmt -d` and `git diff --check` passed.
- Build passed to `/tmp/dunzo-dunnit`.
- Manual Fyne click-through was unavailable in the headless environment.

## Known issue and next step

Sync still uses separate Push and Pull actions: Push stages, commits, then
pushes; Pull stages, commits, then pulls with rebase. Pull/rebase-before-push
was not changed because the automatic safety review rejected broad automatic
remote operations against an unverified repository. On resume, manually test
the Daybook flows and authorize that sync-sequence change only if it is wanted.

The source tree was clean at session end; this handoff is the only new file.
