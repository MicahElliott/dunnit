# AGENTS.md

Guidance for AI coding assistants working in this repo.

## Shared rules

At the start of each session, read `~/.config/ai-rules/global.md`,
then inspect the frontmatter in `~/.config/ai-rules/rules/` and read
only rules whose path or other scope matches `~/proj/dunnit` and the
current task. Read the shared `projects/dunnit.md` mapping because this repo
is its Go rewrite, and continue using it if the project is renamed Dunnit.
Ignore rules that refer to unavailable tools, services, or environments.
System and repository instructions take precedence over shared rules.

If the shared directory is unavailable, continue with this file and mention
that the shared rules could not be loaded.

## Project intent

Dunnit is a **KISS** Go/Fyne rewrite of `../dunnit` (a zsh
proof-of-concept, see its `README.md` and `dunnit.zsh` for the
original design and behavior). Keep dependencies and LOC minimal.
Prefer reusing what's already in `go.mod` (e.g.
`github.com/BurntSushi/toml` is already an indirect dep via Fyne —
don't add a different TOML/YAML/JSON library).

The end goal is a system-tray-only app that pops up hourly (during
configured working hours) asking what the user is working on, and
appends the answer to a dated ledger file. It should run well on both
macOS and Linux.

## Layout

- `dunnit.go` — `main()`, currently a rough sketch wiring UI + scheduler.
- `dunnit/ui.go` — the Fyne window/widgets, ledger read/write, editor
  launching.
- `dunnit/sched.go` — `gocron`-based scheduler; **not yet wired up
  properly** to the config or UI (still has hardcoded demo times).
- `dunnit/settings.go` — placeholder Settings window (dummy checkbox
  only so far).
- `dunnit/config.go` — TOML config load/save. Everything dunnit owns
  (ledgers + config.toml) lives under one root dir, `DunnitDir()`
  (`~/.config/dunnit` by default, override with `$DUNNIT_DIR`).
- `dunnit/taskmenu.go` — currently just a stub, not wired to anything.

## Data format

Ledger files: `$DUNNIT_DIR/<year>/<month>/w<week>/ledger-<DOW>-<YYYYMMDD>.txt`,
one line per entry: `[HH:MM:SS] CATEGORY free text #tag`. See sample
real data in the sibling `../mydunnits` repo for ground truth on
format nuances (e.g. `GOAL`, `DONE`, `MEETING`, `TIL`, `WIN` categories
seen in practice). Do not assume `dunnit.zsh`'s exact category set is
final — the Fyne UI currently defines its own (with emoji labels) in
`ui.go`'s `category` widget; keep these in sync if you change one.

## Build/verify

```sh
make build   # go build -o dunnit .
make vet     # go vet ./...
make package # native macOS/Linux package; requires `fyne` CLI (go install fyne.io/tools/cmd/fyne@latest)
```

Always run `make build` (and check `go vet`/editor diagnostics) after
edits — there's no CI here yet. First build of Fyne's cgo/GL deps can
take a couple minutes; subsequent builds are fast.

There is no automated test suite yet. Manual testing is done by the
human running `./dunnit` or `Dunnit.app` and clicking around — when
making UI changes, describe what to click/verify rather than assuming
you can confirm it yourself.

## Working style for this repo

- Do a lot of small checkpoints; ask before big-scope changes (this
  project's owner explicitly wants frequent check-ins here, more than
  usual).
- This is a **personal project** — direct commits to `main` are fine
  (no feature-branch requirement, unlike this user's work repos).
- Don't install new tools/dependencies without asking first.
- Prefer small, targeted diffs — this codebase still has some rough
  demo-code cruft (commented-out blocks, TODOs, unused helpers) left
  over from early Fyne experimentation; clean it up opportunistically
  when touching nearby code, but no need to do a big sweep unprompted.
