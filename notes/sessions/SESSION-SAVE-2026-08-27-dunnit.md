# Dunnit Session Save — 2026-08-27

Personal project: `/Users/E463390/proj/dunnit` (git repo, direct commits
to `main` are fine — this is a personal project, not a work repo).
Sister zsh project at `../dunnit` (reference only, bit-rotted, not
runnable). Private data repo at `../mydunnits`.

## Where things stand

Fully working KISS Go/Fyne (v2.8.0) menu-bar app. Builds clean via
`make build`; `make package` produces `Dunnit.app` with a real icon for
macOS (use `open Dunnit.app`, not `./dunnit`, to get proper Cmd-Tab
identity/icon). All work this session is committed to `main` (latest:
`48661d6`). README.md and AGENTS.md are up to date and describe the
architecture — read those first in a new session, they're more
authoritative than this file for anything not called out below as
"in progress"/"known issue".

## Key design decisions made this session

- **Single `DUNNIT_DIR` env var** controls everything: ledgers at
  `$DUNNIT_DIR/<year>/w<week>-<month>/ledger-<YYYYMMDD>.txt`, config at
  `$DUNNIT_DIR/config.toml`. Default `~/.config/dunnit`. (Went through
  a few messy iterations — `DUNNIT_DIR`/`DUNNIT_LEDGER_DIR`/
  `DUNNIT_CONFIG_DIR` confusion — before collapsing to just one var.
  Micah's actual working setup: `DUNNIT_DIR=~/proj/mydunnits`.)
- Config schema (`dunnit/config.go`): `day_start`, `day_end`,
  `hourly_minute`, `lunch_time` — ported from old dunnit's
  `config-example.zsh`. No `ledger_dir`/`dunnits_dir` key anymore.
- Settings menu (tray → "Settings...") is a real editable form now,
  not a placeholder.
- Scheduler (`dunnit/sched.go`) reads config and fires: hourly popup
  (via `gocron.CronJob` at `:hourly_minute`), lunchtime reminder, and
  End-of-Day window — all gated to Mon-Fri and (for hourly/lunch)
  `day_start`..`day_end`. Had a real bug here: gocron callbacks run on
  their own goroutine, not Fyne's main thread — any UI calls
  (`w.Show()` etc.) from inside a scheduled task **must** be wrapped in
  `fyne.Do(func() { ... })` or you get threading-safety warnings /
  crashes. Already fixed and pattern established — follow it for any
  new scheduled UI-touching code.
- Bare `Escape` key can **never** work as a global in-window shortcut
  in Fyne — confirmed by reading Fyne's `triggersShortcut()` source;
  it only builds a dispatchable `Shortcut` when a modifier
  (Ctrl/Cmd/Alt/Shift) is held. Switched hide-window shortcut to
  Cmd+W/Ctrl+W (`fyne.KeyModifierShortcutDefault`) instead — this is
  the last word on that, don't waste time re-litigating bare Escape.
- No true OS-level global hotkey to summon the app from anywhere is in
  scope — Fyne can't do it, and a real solution needs OS-level
  accessibility permissions / a new dependency. Recommended path (not
  implemented, no code involved): OS-native hotkey → `open Dunnit.app`
  (macOS Automator/Shortcuts quick action, or Linux WM keybinding).
- Summarize feature (`dunnit/summarize.go`): tray menu → "Summarize..."
  → pick Day/Month/Quarter → gathers matching `ledger-*.txt` files
  under `$DUNNIT_DIR` → shells out to `gh copilot -p "..." --silent
  --allow-all-tools` (no new Go dep) → result auto-copied to clipboard
  and shown in a selectable `MultiLineEntry` window. `gollm` was
  considered but explicitly deferred in favor of the simpler `gh
  copilot` shell-out per Micah's choice.
- End-of-Day window (`dunnit/eod.go`): recreates `dunnit-eod`'s
  *spirit* as ONE window (not the original's long chain of separate
  alerter popups) — Summary, Productivity 1-5, Sentiment, Tomorrow's
  Goals. Summary/productivity/sentiment write to today's ledger;
  goals write as `GOAL` lines into **tomorrow's** ledger file. Wired
  to tray menu ("End of Day...") and auto-fires at `day_end` via
  scheduler.
- Explicitly deferred/skipped from the original dunnit for KISS: per-
  tag impact-statement prompts, weekly objectives, TODO/BLOCKER
  carryover, pandoc HTML report generation. Revisit only if missed
  after living with the simplified version.

## Known issues / not yet verified

- **End-of-Day and Summarize features are untested by the user as of
  this save** — I (the assistant) can't click through the GUI myself;
  all verification has been `go build`/`go vet`/`editor_diagnostics`
  plus manual logic review. Next session should start by asking
  whether these were tried and how they went.
- `dunnit/taskmenu.go` is still a dead stub, unreferenced.
- `updateTime()` in `ui.go` is dead code (harmless, low priority to
  remove).
- Category-select trimming (`selectedCat = res[1]`) is a naive
  whitespace-split of the emoji-prefixed label; works but is fragile
  string-parsing worth revisiting if labels change.

## Working style reminders for this repo (also in AGENTS.md)

- Lots of small checkpoints; ask before big-scope changes.
- Direct commits to `main` are fine here (personal project exception
  to the general "never commit to master" rule).
- Don't install new tools/deps without asking first (already installed
  this session: `fyne` CLI packaging tool via `go install`).
- Manual GUI testing is done by Micah clicking around after `make
  build`/`make run`/`make package` — always describe exactly what to
  click/verify rather than assuming it's confirmed.

## Natural next steps (pick up here)

1. Get user feedback on Summarize + End-of-Day (untested).
2. Consider whether `gh copilot` summarization prompt/quality needs
   tuning based on real output.
3. Maybe polish: remove dead code (`taskmenu.go`, `updateTime`),
   revisit category-label parsing.
4. Optionally implement OS-native global-hotkey launch (Automator/WM
   keybinding docs in README, not app code) if still wanted.
5. Anything from the "skipped from original" list above, if missed.
