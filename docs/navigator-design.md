# Navigator & Ledger Index -- Design Notes

Status: informal design notes, matches docs/category-taxonomy.md's
style. Captures the 2026-09-02 navigator design discussion and the
infrastructure/first-mode work done from it. Read this before adding a
new navigator mode or touching the shared ledger index.

## Background

Prior sessions floated a "navigator" -- a bigger, more general browse/
search/analyze surface over ledger history and generated reports,
explicitly called out as future work in `periodpicker.go`'s own
comments (the period picker is a small, deliberately-scoped precursor,
not the navigator itself).

## Approaches considered (2026-09-02 design session)

Starting list: AI Q&A ("what did I do on X two months ago"),
file-explorer-style report browsing, general full-text search
(already existed, `search.go`), tag nav/search, basic histograms.

Additional approaches surfaced during discussion:
- Timeline/calendar view (GitHub-contribution-graph-style day cells)
- "On this day" / anniversary view
- **Category-first browse** ("show me every FIXME/CAREER/etc I've
  ever logged") -- judged the highest-value cheap win given the
  category-taxonomy work already done, so built first (see below).
- Person-focused browse and rollups (`@Name`) for collaboration history,
  feedback/KUDOS preparation, and person-weighted period summaries.
- Saved-reports library/browser (the literal "bigger period picker")
- Cross-report search (spanning generated .md reports, not just raw
  ledgers -- not yet done, reports aren't in the shared index)
- Tag co-occurrence / relationship view
- Streaks/frequency stats per category/tag
- Diff/compare view between two periods
- Saved/pinned queries
- Export/ad hoc AI summary from any filtered navigator selection

## Key design decisions

1. **Two distinct corpora**: raw ledgers vs. generated reports. This
   round of work only covers raw ledgers (`LedgerEntry`/`LedgerQuery`,
   below). Reports remain a separate, unindexed corpus -- a future
   navigator mode wanting cross-report search or a reports library
   browser will need its own (much smaller) index, reusing
   `listReviewReportsForPeriod`/`listReviewReportsOverlapping`'s
   glob-and-parse approach from `review.go` rather than this one.

2. **Composable query, not one screen per approach.** Every mode
   (category browse today; tag/date-range/AI-ask/etc later) should
   build a `LedgerQuery` and call `FilterLedgerEntries`, rather than
   hand-rolling its own scan+match logic. This is what makes filters
   composable (category + tag + date range together) instead of five
   independent one-off screens.

3. **Shared cached index, replacing 3 independent full-file-walks.**
   Before this work, `search.go`, `tags.go`, and `trend.go` each
   independently walked every `ledger-*.txt` file and re-parsed lines
   themselves. Given corpus scale (one small file per day, thousands
   of files over years of use) this repeated work was wasteful and
   was flagged as the top-priority prerequisite before building any
   new navigator mode on top.

4. **No charting library.** Plain ASCII/text-table stays the house
   style (matches `trend.go`'s existing approach and AGENTS.md's
   minimal-deps guidance). Any future histogram mode should stay text-
   based unless a specific need can't reasonably be done that way.

## What was built

### `LedgerEntry` (ledgerentry.go)

One parsed ledger line: `Date`, `Time`, `Category`, `Text`, `Tags`, `People`
(pre-extracted), `Mins` (parsed from trailing `" ~Nm"`), `Source`
(file path), `Line` (line number). Unifies what `parseLedgerLine`/
`parseLedgerLineTime`/`extractTags` each did separately per-caller
before.

### Shared index (ledgerindex.go)

`AllLedgerEntries()` -- cached (5 min TTL, same philosophy as
`tags.go`'s `tagCache`, but `RWMutex` instead of a plain `Mutex` since
reads now vastly outnumber writes), full parsed view of every ledger
line, oldest-first.

`InvalidateLedgerIndex()` / `InvalidateLedgerCaches()` (invalidates
both the tag cache and this index in one call). Wired into:
- `ui.go`'s `recordActivity` (new entries)
- `undo.go`'s `writeLedgerLines` (the shared rewrite path used by
  undo/edit/category-rewrite -- covers all of those in one place
  rather than needing each call site updated individually)

### Query/filter layer (ledgerquery.go)

`LedgerQuery{Categories, Tags, People, From, To, Text}` + `Matches`/
`FilterLedgerEntries`. All filters AND together; each filter empty/
zero means "unconstrained" on that axis.

### Migrated onto the shared index

- `search.go`'s `searchLedgers` -- now `FilterLedgerEntries(LedgerQuery{Text: query})`.
  Note: result lines are now reconstructed from parsed fields (so
  formatting is consistent) rather than the literal raw line -- not
  byte-identical to the pre-migration output, but equivalent content.
- `tags.go`'s `scanAllTags` and `gatherTagStats` -- both iterate
  `AllLedgerEntries()` instead of re-opening/re-scanning every file.
- `trend.go`'s `gatherTrendPoints` -- same migration.

**Deliberately not migrated**: `todos.go`'s `parseOpenItems` (open-
item resolution-state logic) and `standup.go`'s window-scanning
(time-window cutoff logic) -- both have specialized behavior that may
not cleanly fit a generic filter; revisit individually later rather
than forcing them through in this pass.

### First navigator mode: category-first browse (navigator.go)

`showNavigatorWindow(a)` -- a `widget.NewSelect` category dropdown
("All" + every `Categories` code, including EODOnly ones -- unlike
`showHelp`'s picker-oriented legend, Navigator is about browsing what's
actually *in* the ledger, and EODOnly categories do appear there even
though they're never hand-picked) plus a scrollable results view,
built directly on `FilterLedgerEntries`. Wired into the tray's
Ledger submenu, right after "Search...".

Also added (not yet surfaced in any UI): `categoryCounts`/
`sortedCategoryCounts` -- small building blocks for a future
"histogram" mode, computing per-category counts over a given entry
slice.

### Tag filter, date-range filter, and Ask-AI (2026-09-02, same session)

Extended the same Navigator window (not a separate mode/screen) with
two more composable filters plus an action, all built on the existing
`LedgerQuery`:

- **Tags**: a freeform entry field (`tagsEntry`), same input
  convention as Settings' Report Exclude Tags field (space/comma
  separated, tolerant of a missing leading `#`) -- parsed by
  `parseNavigatorTagsInput` into `LedgerQuery.Tags`. A full multi-
  select over `KnownTags()` was considered but freeform text matches
  the tag corpus's open-ended nature (same reasoning Settings already
  used for its own tag field) better than a long checkbox list would.
- **Date range**: a fixed-option dropdown (`navigatorDateRangeOptions`:
  All time / Today / This-or-Last Week/Month/Quarter/Year), resolved
  by `navigatorDateRange` into `LedgerQuery.From/To` by reusing
  `periodNominalRange`/`periodOffsetAnchor` from `period.go` directly
  -- no new range-addressing scheme invented. Deliberately a small
  fixed list rather than a full date-picker, matching Navigator's
  other filters' low-effort dropdown style; a free date-range picker
  can be added later if this proves too coarse.
- **Ask AI about these...**: takes the currently-filtered entry set
  (all three filters composed together), renders it back to ledger-
  line-shaped text (`ledgerEntriesToText`), and feeds it plus a
  user-typed free-form question into `summarizeWithLLMCLIPrompt` --
  the same integration point every other AI-report feature already
  funnels through (Standup/Status Report/Annual Review/Kickoff-Review),
  just with a one-shot question instead of a fixed instruction
  template. Answer opens in its own window (`showNavigatorAIAnswerWindow`)
  separate from Navigator's own, so the filter view stays usable and
  multiple questions can be asked without each answer replacing the
  last.

This directly closes out the original "AI: ask questions about what
did I accomplish on topic X" bullet from the initial navigator
brainstorm, composed with the category/tag/date filters rather than
built as its own separate screen.

### Histogram (2026-09-02, same session)

Wired `categoryCounts`/`sortedCategoryCounts` into an actual view:
Navigator's "Histogram..." button renders the currently-filtered
entry set's per-category breakdown as a plain-text ASCII bar chart
(`formatCategoryHistogram`), scaled so the largest bar is a fixed
width (`histogramBarWidth`) and every other bar proportional to it --
matching `trend.go`'s existing house style (no charting library).
Opens in its own window, same rationale as the Ask-AI answer window
(stays open alongside the filtered browse view).

### Reports Library (2026-09-02, same session)

New, separate corpus/window from Navigator (which only covers raw
ledgers) -- **reportindex.go**/**reportsearch.go**/**reportslibrary.go**:

- `ReportFile` (reportindex.go): lightweight per-report-file metadata
  (Path, Kind, Theme, Date=file mtime) -- deliberately much lighter
  than `LedgerEntry`, since reports are large markdown documents, not
  per-line structured data. `AllReportFiles()` walks the entire
  `DunnitDir()` tree, including year-level and ledger-adjacent reports,
  parsing kind/theme out of each filename via
  `parseReportFileName` (reusing the same dash-suffix theme-stripping
  approach `review.go`'s `listReviewReportsForPeriod` already uses,
  generalized across all known report-file kinds). No caching (unlike
  `AllLedgerEntries`) -- report file counts are expected to be orders
  of magnitude smaller than ledger line counts, so a fresh walk per
  call is cheap enough; revisit if that assumption proves wrong.
- `SearchReports(query)` (reportsearch.go): case-insensitive substring
  search across every report body (read via `ReportBody`), returning
  one result per matching file with a short surrounding excerpt
  (`excerptAround`) rather than dumping the whole body -- "which
  reports mention X", not an every-occurrence full-text index.
- `showReportsLibraryWindow(a)` (reportslibrary.go, tray: Reports ->
  Reports Library...): combines a Kind filter (dropdown, chronological
  browse of one report family by file mtime) with the free-text search
  above in one window -- selecting a result line and clicking "Open
  Selected Line..." opens the full report in `showGeneratedReport`
  (report.go), reusing the same read-only viewer (with Copy/Save)
  every other report-producing feature already uses, rather than
  building a new one-off viewer.

**Note on Status Report/Annual Review/Kickoff**: these are NOT
included in `AllReportFiles()` -- confirmed during this work that
Status Report and Annual Review are clipboard-only (no `WriteFile`
call anywhere in `statusreport.go`/`annualreview.go`), and Kickoff
windows don't appear to save to disk either. Only Review
(`review-*`), Standup (`standup-*`), Status (`status-*`), and EOD
(`eod-*`) persist as files today -- Reports Library only browses what
genuinely exists on disk.

This closes out the "Saved-reports library/browser" and "Cross-report
search" bullets from the original navigator brainstorm.

## Open items / next steps (not yet done)

- Free (arbitrary) date-range picker for Navigator, if the fixed
  dropdown options prove too coarse in practice.
- Multi-select category filter (currently single-select) if browsing
  "FIXME + RISK together" turns out to be a common need.
- Saved/pinned queries, once real usage shows which filter
  combinations get reused often.
- Reports Library's Kind dropdown is single-select and its results
  view is a plain click-a-line-number affordance (`CursorRow`-based),
  not a proper clickable list widget -- fine for a first pass, but
  worth a real `widget.List`-based results view if this gets used a
  lot.
- Ask-AI-about-these for Reports Library (parallel to Navigator's),
  if a use case for it shows up (e.g. "what changed between these two
  Quarter reviews").
