# Session Save — 2026-09-03 — Ledger/Report Directory Reorg

## Context / decision made

Real ledger data lives in `~/proj/mydunnits` (git repo). User just set
`DUNNIT_DIR` in their env to point there (was previously unset/using
default `~/.config/dunnit`, which is empty in the sandboxed dev
session — don't confuse the two).

**Problem identified**: `getLedger()`/`ledgerDirFor()` (`dunnit/ui.go`)
derive a ledger day's directory as `<year>/w<ISOweek>-<month>/`, where
`<month>` comes from **that individual day's own calendar month** —
not the week's start month. Since ISO weeks don't respect month
boundaries, a week straddling a month boundary produces two different
directories for the same ISO week, e.g.:

```
2026/w36-Aug/   (contains Aug 31, the start of ISO week 36)
2026/w36-Sep/   (contains Sep 1-6, the rest of ISO week 36)
```

This happens ~4-5 times/year, every time a week spans two months.

**Also clarified**: Weekly/Monthly/Quarterly/Annual **Review reports**
(`review.go`) do NOT currently use this per-week directory scheme at
all — they're flat files at `DunnitDir()` root (e.g.
`review-week-20260831-status_report.md`, via `periodReportPathRaw`).
Only two things currently use the per-week directory:
`ledger-YYYYMMDD.txt` files themselves, and the daily-summary doc
(`summary-YYYYMMDD.md`, `dailysummary.go`) which lives alongside its
ledger file.

## Agreed new scheme

Month-first hierarchy, week nested inside, with the week's parent
month **fixed by the week's START (Monday), not each day's own
month**:

```
<DunnitDir>/
  2026/
    review-quarter-2026Q3-....md      # UNCHANGED: stays flat at year root
    review-year-2026-....md           # UNCHANGED: stays flat at year root
    Aug/
      review-month-....md             # MOVED here (was flat at DunnitDir root)
      w35/
        ledger-20260824.txt ...
      w36/
        ledger-20260831.txt           # week 36 started in Aug -> whole week lives here
        summary-20260831.md
        w36-review-....md             # MOVED here (was flat "review-week-...") --
                                       # user's explicit preference: nest inside the
                                       # week's own dir rather than as a sibling of it
    Sep/
      w36/                            # NOTE: does NOT exist under this scheme --
                                       # all of week 36 (including Sep 1-6 entries)
                                       # lives under Aug/w36/ per the week-start rule.
      w37/
        ...
```

Key rules:
1. Bare `w<N>` directory name (no month suffix) — disambiguation comes
   from nesting under the correct month dir, not a compound name.
2. Which month a week's directory lives under is fixed **once**, from
   that week's Monday's month — computed via a single shared helper,
   not re-derived per-day at each call site (today's bug).
3. Weekly Review reports move to inside their own week's directory:
   `Aug/w36/w36-review-<theme>.md` (user's explicit preference over my
   original `Aug/w36-review.md` sibling-file proposal — makes sense
   since the `w36/` dir already exists and holds everything else about
   that week).
4. Monthly Review reports move to their month's directory:
   `Aug/review-month-<token>-<theme>.md`.
5. Quarterly and annual Review reports **stay flat** at `<year>/` root
   — not enough volume (1-4/year) to justify their own nesting.

## Plan (from `eca__task`, 5 items, all still pending as of this save)

1. **Audit all path-construction code** — read `ui.go`
   (`getLedger`/`ledgerDirFor`), `eod.go` (`tomorrowLedgerPath`),
   `dailysummary.go` (`dailySummaryPath`), `streak.go`, `review.go`
   (`reviewReportPath`/`periodReportPathRaw`/
   `listReviewReportsForPeriod`/`listReviewReportsOverlapping`),
   `report.go` (`periodReportPath`), `meetingprep_test.go` +
   `carryforward_test.go` (these two **duplicate** the path formula
   inline instead of calling the shared helper — must fix to call the
   real helper so they can't drift out of sync again).

2. **Implement new directory-scheme helpers + update all call sites**
   — a shared "week-start-month" helper computing `(year, moname,
   week)` from a date's ISO week's *Monday*, not the date's own month;
   used by every current call site instead of each independently
   deriving month-from-date. Update `ledgerDirFor`'s signature/callers.
   Move weekly Review reports into the nested `Aug/w36/w36-review-*.md`
   path (only `periodWeek` changes here); monthly Review reports move
   to `Aug/review-month-*.md`; quarter/year Review paths **unchanged**.
   Fix the two test files to call the shared helper instead of
   duplicating the formula.

3. **Build/test the code changes** — `make vet`, `make build`,
   `go test ./...` all passing, BEFORE touching any real data.

4. **Migrate real `~/proj/mydunnits` data** — `git mv` every existing
   ledger/summary file from old `<year>/w<week>-<month>/` scheme into
   new `<year>/<month>/w<week>/` scheme, per the week-start-month rule
   (e.g. both `2026/w36-Aug/` and `2026/w36-Sep/` merge into
   `2026/Aug/w36/`). Also relocate any existing flat `review-week-*.md`
   into the new nested location, and `review-month-*.md` into the new
   per-month location. Leave `review-quarter-*.md`/`review-year-*.md`
   flat, untouched. Cover **every year present**: 2021, 2025, 2026 (per
   `find` output below). **Do NOT touch `~/proj/mydunnits/tmp/`**
   (looks like scratch data, e.g. `tmp/2026/Aug`, `tmp/2026/q3`) without
   asking first. Verify no files lost (file-count diff) and prefer real
   `git mv` renames over delete+add pairs in `git status`.

5. **Final verification against real migrated data** — run/script
   against `DUNNIT_DIR=~/proj/mydunnits` to confirm
   `getLedger()`/`ledgerDirFor()` resolve to paths that actually exist
   post-migration for a few sample dates (today, and a date in the old
   `w36-Sep` range). Grep the whole `dunnit/` tree once more for any
   remaining hardcoded old-style path assumptions.

## Actual current directory listing (`~/proj/mydunnits`, captured this session)

```
2021/w16-Apr
2021/w17-May
2021/w18-May
2021/w19-May
2021/w20-May
2021/w21-May
2021/w22-Jun
2021/w23-Jun
2021/w24-Jun
2021/w25-Jun
2021/w26-Jul
2025/w17-Apr
2025/w18-Apr
2025/w18-May
2025/w25-Jun
2025/w26-Jun
2025/w32-Aug
2026/w35-Aug
2026/w36-Aug
2026/w36-Sep
tmp/2026/Aug        <- DO NOT TOUCH without asking (looks like scratch)
tmp/2026/q3         <- DO NOT TOUCH without asking (looks like scratch)
```

**Confirmed** (ran `find . -maxdepth 1 -name '*.md'` at repo root):
no flat `review-*.md`/`som-*.md` files exist. Only 2 flat `dsu-*.md`
files (`dsu-20260902.md`, `dsu-20260903.md`) — these are Standup
exports, a separate/unrelated flat convention per `report.go`'s own
comment ("dailysummary.go's per-ledger-directory summary-<date>.md
convention is intentionally left alone... a deliberately different
scheme") — **not** part of this reorg, leave them alone.

**Confirmed** (ran `find . -maxdepth 2 -type d -name 'w*'` across the
whole repo): only **2026** has an actual week/month collision
(`w36-Aug` + `w36-Sep`, both week 36). 2021 and 2025's week
directories are all distinct week numbers, no merges needed there —
those years' `git mv` migration is a straightforward 1:1 rename
(`w<N>-<Mon>` -> `<Mon>/w<N>`), no directory-merging logic needed
except for 2026/w36.

## Also worth double-checking early next session

- Whether `~/proj/mydunnits` being a **separate git repo** from
  `~/proj/dunnit` (the app's own source repo) has any implications for
  how `git mv` should be run (it should — run `git mv` commands with
  `working_directory` set to `~/proj/mydunnits`, not `~/proj/dunnit`).
  **Confirmed via `eca__shell_command` with `working_directory` set**
  that `git status --short` in `~/proj/mydunnits` shows a clean tree
  (only untracked `2026/`, `config.toml`, `dsu-*.md`,
  `last_pulled.json` — nothing already staged/modified). Note: the
  `eca__git` tool itself does NOT accept a working-directory override
  and always runs against the main workspace root (`~/proj/dunnit`) —
  use `eca__shell_command` with an explicit `working_directory` for
  all `git`/`git mv` operations against `~/proj/mydunnits` instead.
- Whether other years (2021, 2025) have any week-spans-month-boundary
  collisions like 2026's w36 — **confirmed no**, see above.
