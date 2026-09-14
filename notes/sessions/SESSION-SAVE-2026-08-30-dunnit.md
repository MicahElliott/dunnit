# Dunnit Session Save — 2026-08-30 — FRD Implementation + UX Polish

Personal project: `/Users/E463390/proj/dunnit` (git repo, direct commits
to `main` are fine, no feature-branch requirement). Latest commit:
`1a823fa` "Convert Daybook-parented dialogs to standalone windows".

## Where things stand

**All of FRD Phase 0-4 is implemented** (`FRD-DRAFT-2026-08-28.md`),
plus the promoted Phase-5 items (FR-14, FR-26, FR-36). That's FR-01
through FR-23, FR-14, FR-26, FR-36 -- essentially everything except
the remaining genuinely-speculative Phase 5 items (FR-24, FR-25,
FR-27, FR-29 through FR-35).

This session (2026-08-29 evening through 2026-08-30) covered:
1. Implementing FR-14 (SOM wizard), FR-36 (post-meeting capture),
   FR-26 (snooze).
2. A UX/architecture discussion + implementation pass:
   - Tray menu reorganized into submenus (Meetings/Reports/Ledger),
     since Fyne supports `ChildMenu` for real (labeled, collapsible)
     submenus, not just flat separators.
   - **Key decision, worth remembering**: Daybook (the main capture
     window) is assumed to be **normally hidden**, popping up only
     occasionally then closing. This flipped a bunch of earlier
     "where should this button live" calls -- anything that isn't a
     direct reaction to Daybook already being open belongs in the
     tray menu, not Daybook. Daybook's button row got pruned down to
     just Save/Ditto/Snooze/Help.
   - Found and fixed a real architectural mistake: six dialogs
     (Meeting Prep, Post-Meeting Capture, Search, Status Report,
     Annual Review, Summarize) had been built with
     `dialog.NewCustomConfirm`/`NewCustom`, which requires a parent
     `fyne.Window` -- so they'd been parented on Daybook (`w4`) as a
     shortcut, meaning every tray call site also did `w4.Show()`
     first just to satisfy the dialog constructor. This was backwards
     given "Daybook usually isn't open." Converted all six to their
     own standalone `fyne.Window`s (matching the pattern already used
     correctly by Settings/Trend View/Help/Recurring Meetings/Undo-
     Edit). No more tray action depends on Daybook being open.
   - Various polish: renamed "Category Legend" to "Help" (in both tray
     and a new Daybook button), configurable snooze duration
     (`snooze_minutes` config key, Settings field, tray submenu with
     15/30/60 min quick options), tray's "Show" now calls
     `RequestFocus()`, removed dummy "I colored this" green text,
     reordered Daybook layout (prompt now at top), replaced hardcoded
     `getCommonTopics()` placeholder with a real
     `commonAndRecentTags()` that blends frequency+recency from
     ledger history, Daybook window titled "Dunnit: Daybook", Settings
     window also links to Recurring Meetings.

## Immediate next step (where to pick up)

No urgent unfinished work -- last thing done was the standalone-
window refactor, verified building/vetting/testing clean. Reasonable
next steps, in rough priority order:

1. **Manual GUI testing** of everything changed this session --
   especially the standalone-window conversions (Meeting Prep, Post-
   Meeting Capture, Search, Status Report, Annual Review, Summarize)
   and the tray submenu reorg (Meetings/Reports/Ledger, Snooze
   submenu). None of this has been clicked through by a human yet.
2. **FR-18 open design questions** (see `docs/open-design-
   questions.md`) -- still unresolved: what actually differentiates
   the daily summary doc from Summarize's existing "Day" output,
   whether EOD is the right auto-trigger timing, and whether it
   should shell out to an LLM unprompted by default at all. Currently
   gated behind an opt-in config flag (`auto_draft_daily_summary`,
   default false) pending resolution.
3. Remaining Phase 5 speculative FRs, if wanted (FR-24 quick-log
   hotkey, FR-25 voice-to-text, FR-27 DND awareness, FR-29 "on this
   day", FR-30 more AI features, FR-31 pandoc export, FR-32 global OS
   hotkey [docs-only, not real app work], FR-33 notification-center
   popups, FR-34 multi-machine sync, FR-35 product rename [decided:
   no-go, not doing this]). None have been started. See FRD's Phase 5
   section for details/status on each.
4. There's an untracked `SESSION-SAVE-2026-08-28-dunnit.md` in the
   repo root -- superseded by this file and the actual FRD progress;
   safe to delete whenever, or ignore.

## Working agreement for implementation phase (unchanged)

- One commit per FR item (or small logical chunk) -- clean history
  mapped to FRD numbering. Non-FR polish/refactor commits are fine
  too (this session had several).
- Lots of small checkpoints, ask before big-scope changes.
- Always run `go build`, `go vet ./...`, `gofmt -l` (note: `ui.go` has
  pre-existing gofmt issues predating this whole effort, unrelated to
  any of these changes -- don't treat those as new regressions),
  `go test ./...`, and `editor_diagnostics` after edits.
- Don't install new tools/deps without asking first.
- Manual GUI testing is done by Micah clicking around -- describe
  exactly what to click/verify, don't assume you can confirm it
  yourself.

## Key architecture/design notes worth remembering

- **Daybook is normally hidden.** This is the deciding factor for
  where any new button/action should live: if it's not a direct
  reaction to Daybook being open right now, it goes in the tray menu,
  not Daybook.
- **Never parent a dialog on Daybook's window as a shortcut.** Give
  every tray-invoked, occasional workflow its own standalone
  `fyne.Window` instead of `dialog.NewCustomConfirm(..., w4)`. This
  was a real mistake made and then fixed this session -- worth not
  repeating.
- Tray menu structure (current): top-level = Show, Start of Day/End
  of Day/Start of Month, Snooze (with 15/30/60 min submenu),
  submenus = Meetings (Meeting Prep/Post-Meeting Capture/Recurring
  Meetings), Reports (Summarize/Standup Summary/Status Report/Annual
  Review/Trend View), Ledger (Show/Edit Today's Ledger/Undo-Edit Last
  Entry/Search/Daily Summary Doc), then Help/Settings at the bottom.
- Daybook's button row is now just: Save, Ditto, Snooze, Help.
- Config file (`config.toml` under `DunnitDir()`, default
  `~/.config/dunnit`) now has: `day_start`, `day_end`,
  `nudge_interval_minutes`, `lunch_time`, `[[recurring_meeting]]`
  array-of-tables, `weekly_digest_day`/`weekly_digest_time`,
  `auto_draft_daily_summary`, `snooze_minutes`.

## Full FR list location

See `FRD-DRAFT-2026-08-28.md`'s "Summary table" section for all 36
items at a glance with phase and dependency notes (status there is
now stale re: what's done -- cross-reference against git log instead,
since implementation has run well ahead of that doc being updated).
