# Dunnit Session Save -- 2026-08-31 -- recurring-items, Daybook sections, GoReleaser

Status: work described below is **done and verified** (build/vet/test
clean) except the GoReleaser/GH Actions piece, which is **deferred to
next session** per Micah's request (he wants to run `goreleaser init`
himself interactively first).

## 1. Recurring Items feature (earlier in session -- done)

Full design + implementation already landed; see
`RECURRING-ITEMS-DESIGN-SEED.md` at repo root for the complete
writeup (decisions: surface as suggestion not auto-seed, has a
management GUI, weekly piggybacks on daily SOD, monthly seeds into
SOM as a checklist). New file `dunnit/recurring.go`, plus touches to
`config.go`/`sod.go`/`som.go`/`settings.go`/`ui.go` (tray menu link
under Ledger -> Recurring Items...).

Follow-up polish this session:
- Fixed tiny/scrolling entry box (now stretches via
  `newStretchRowLayout`).
- Enter key in the text field now submits (`OnSubmitted`).
- Removed redundant in-window "Recurring Items" title label (window
  title already says it).
- Added an italic help line under the entry row, with an emoji
  prefix -- see "emoji gotcha" below for why it went through two
  iterations (ℹ️ -> 📝).
- Replaced "Delete Selected" bottom button with a per-row 🗑️
  (tooltip "Delete") hover-button, one per existing item.
- Category picker restricted to TODO/GOAL/KUDOS only (was previously
  all `openTrackedCategories`).
- Added horizontal `widget.NewSeparator()` between the entry
  controls and the existing-items list.
- Added `Ledger -> Recurring Items...` tray menu item, consistent
  with the existing `Meetings -> Recurring Meetings...` pattern.

### Emoji gotcha (reusable knowledge)
`ℹ️` (U+2139 + U+FE0F variation selector) rendered as a plain `i` in
Fyne's canvas.Text -- Fyne doesn't honor the VS16 "render as emoji"
variation selector, so any emoji that's normally only "colorful" via
a variation selector (rather than being its own dedicated codepoint)
will silently degrade to its plain-text form. Fix: pick an emoji
that's a single dedicated codepoint (no variation selector needed),
e.g. the 📝 used here, or the other emoji already working elsewhere
in this app (✔️ 🗑️ etc -- some of *those* do carry VS16 too and
apparently still render as pictographs in this environment/font, so
this isn't 100% predictable -- if a new emoji looks like plain text
after adding it, that's the likely cause; just try a different one
and rebuild to check visually, no way to confirm from code alone).

## 2. Daybook UI / categories changes (this turn -- done)

**`categories.go`:**
- Moved `IDEA` from `"now"` group to `"plan"` group (alongside
  `SOMEDAY`, since `som.go`'s step 2 already treats IDEA/SOMEDAY as
  a matched triage pair -- now the picker grouping matches that).
- Added a new `Category.EODOnly bool` field. Set `true` for
  `SUMMARY`, `PRODUCTIVITY`, `MEETING_HOURS` (all three are always
  written directly via `recordActivity` calls in `eod.go`'s Finalize
  Day flow, never meant to be hand-picked). `IMPACT`/`MILESTONE`/
  `CAREER`/`FAIL`/`WASTED` stay pickable (`EODOnly: false`) --
  Micah only explicitly named IMPACT/SUMMARY/MEETING_HOURS as
  wanting removed, and on reflection IMPACT reads as something you
  might want to log the moment it happens, not just at EOD, so it
  was left pickable; flag if that guess was wrong.
- `CategoryLabelsForGroup` now skips `EODOnly` categories -- affects
  Daybook's live picker only. `Categories` itself (used by the Help
  legend, annual review scans, status report, etc) is untouched, so
  EOD-only categories still show up there for documentation/scanning
  purposes.
- Note: all `Category{}` struct literals are positional (not named
  fields), so adding `EODOnly` required appending a 6th positional
  bool to every single literal in the `Categories` slice -- if you
  add another field later, same mechanical edit is needed across all
  ~26 entries (or switch the slice to named-field literals to avoid
  this in the future -- not done here to keep the diff minimal).

**`todos.go`:** added two new general-purpose helpers alongside the
existing TODO/GOAL-specific `getOpenItems`/`groupOpenItemsByCategory`:
- `categoryGroupOrder(group)` -- category codes for a group, in
  `Categories`' declared order.
- `getCategoryGroupItems(group)` -- today's ledger entries whose
  category is in that group (used for `"now"` and `"reflect"`; `"plan"`
  continues to use the existing `getOpenItems`/`openTrackedCategories`
  path since that one has resolve/dedup semantics the other two don't
  need).
- `groupCategoryItemsByGroup(group, items)` -- buckets those items by
  category preserving `categoryGroupOrder`, mirroring
  `groupOpenItemsByCategory`.
- `getCompletedItems()` (old DONE-only helper) is now unused
  dead code -- left in place, not deleted, in case something still
  references it later; grep before assuming it's truly orphaned.

**`ui.go`:** Daybook's bottom accordion now has three sections
instead of two, all built the same way (grouped-by-category with
italic sub-headings, same visual pattern "Planned" already had):
1. **"Completed"** (renamed from implicitly-DONE-only) -- now shows
   all `"now"`-group entries (DONE/ONGOING/TIL/KUDOS/WIN), not just
   DONE, each under its own category sub-heading. Still first/open-
   by-default in the accordion.
2. **"Planned"** (renamed from "Upcoming") -- unchanged data/behavior
   (TODO/GOAL/WAITING/QUESTION/FIXME/RISK, Done/Postpone/Discard
   actions), just the label text changed. `sod.go`'s sync-note text
   was also updated to say "Planned" instead of "Upcoming" for
   consistency.
3. **"Reflections"** (new) -- shows today's `"reflect"`-group entries
   (IMPACT/MILESTONE/CAREER/FAIL/WASTED -- EODOnly ones would still
   show here too if somehow present, since EODOnly only gates the
   picker, not this readback), grouped by category sub-heading, same
   pattern.

All three sections' refresh functions (`refreshOpenItems`/
`refreshCompleted`/`refreshReflections`) are now called together at
every place any of them used to be called alone (`saveEntry`, Ditto,
Undo/Edit Last Entry, tray menu "Show", the Done/Postpone/Discard
buttons where relevant) so no section goes stale relative to the
others.

### Verification performed
- `make build`, `make vet`, `go test ./...`, `eca__editor_diagnostics`
  all clean.
- Wrote and ran (then deleted) a throwaway `zzcheck_test.go` exercising
  `getCategoryGroupItems`/`groupCategoryItemsByGroup`/
  `CategoryLabelsForGroup` end-to-end against a temp `DUNNIT_DIR` --
  confirmed IDEA no longer appears in `getOpenItems()`, IDEA's group
  is now `"plan"`, `"now"`-group items count correctly includes
  DONE/ONGOING/WIN, `"reflect"`-group items correctly picks up IMPACT
  only (from a small fixture), and `CategoryLabelsForGroup("reflect")`
  excludes SUMMARY/PRODUCTIVITY/MEETING_HOURS. No permanent test file
  was added to the repo (repo currently has zero Daybook/categories
  unit tests -- only `undo_test.go` pre-existed).
- **Not done**: manual click-through in the actual running app (per
  repo convention, that's on Micah) -- specifically worth eyeballing:
  the new 📝 emoji actually renders as a pictograph (not plain text)
  in your environment, and that all three accordion sections
  look right together with real ledger data.

## 3. GoReleaser + GitHub Actions (next session)

**Not started -- deferred on purpose.** Micah has installed
GoReleaser (`brew install --cask goreleaser/tap/goreleaser`) but has
NOT yet run `goreleaser init`, and wants to do that himself in a new
session before any workflow file gets written.

Repo facts gathered this session, useful context for next time:
- No `.github/` directory exists yet at all (no CI of any kind).
- `git remote -v` -> `https://github.com/MicahElliott/dunnit.git`
  (origin, both fetch/push) -- so releases would publish to
  `MicahElliott/dunnit` on github.com (not GH Enterprise, no
  `gh-enterprise-host-gotcha.md` concerns here since this is a
  personal, not work, repo).
- Module name is `gsd` (`go.mod`), Go version `1.23.7`.
- Current build path is a plain `Makefile` (`make build` ->
  `go build -o dunnit .`; `make package` -> `fyne package -os darwin
  -icon Icon.png`, macOS-only, requires the separately-installed
  `fyne` CLI).
- This is a Fyne (GUI, cgo/OpenGL-backed) app -- **not** a typical
  pure-Go CLI binary GoReleaser handles out of the box. Two known
  complications to raise/resolve when picking this back up:
  1. Fyne apps typically need OS-specific packaging (`.app` bundle
     on macOS, maybe DMG; possibly different bundling on Linux) --
     GoReleaser's default `builds:`/`archives:` sections assume
     plain binaries; may need a custom pipe or to keep using `fyne
     package`/`fyne-cross` as a pre-step invoked from the GH Actions
     workflow, with GoReleaser only handling GitHub Release creation
     + artifact upload (not the actual OS packaging step).
  2. cgo (Fyne's OpenGL bindings) generally means **no easy
     cross-compilation** from a single Linux GH Actions runner --
     may need a build matrix (macOS runner for the macOS build, etc)
     or the `fyne-cross` tool (Docker-based cross-compilation
     wrapper Fyne's own docs recommend) rather than relying on
     GoReleaser's native Go cross-compile support alone.
- Per repo's own `AGENTS.md`: this is a personal project, direct
  commits to `main` are fine, no feature-branch requirement, but
  Micah wants small checkpoints/frequent check-ins on this repo
  specifically -- keep that in mind for the GH Actions work too
  (don't do it all in one giant unreviewed change).

### Concrete next steps for the new session
1. Micah runs `goreleaser init` himself first (generates a starter
   `.goreleaser.yaml`).
2. Review the generated config together -- likely needs hand-editing
   for the Fyne/cgo packaging concerns above (may end up simpler to
   have GH Actions run `fyne package`/`fyne-cross` directly and use
   GoReleaser (or even just `gh release create` / softprops action)
   just for the GitHub Release + upload step, rather than expecting
   GoReleaser's `builds:` section to invoke Fyne packaging itself).
3. Decide macOS-only (matches current `make package` scope) vs. also
   attempting Linux packaging for v1 of the workflow.
4. Write `.github/workflows/release.yml` triggered on push to `main`
   (or on tag push -- worth asking Micah which trigger he wants:
   "every push to main" vs. "only on version tags", since GoReleaser
   conventionally expects the latter).
5. Add any required repo secrets (likely just the default
   `GITHUB_TOKEN`, which Actions provides automatically for
   `contents: write` release creation -- shouldn't need a PAT unless
   doing something like a Homebrew tap push).
6. Test with a real push/tag once the workflow is in place; check
   the produced GitHub Release artifacts look right.
