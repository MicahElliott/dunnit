# Dunnit DOING Category — Implementation Handoff

## Objective

Replace the internal `ONGOING` mechanism with a visible, first-class `DOING` category. A planned item should move through one active lifecycle:

`TODO → DOING → DONE`

The item remains one logical task as its state changes. The active UI should not show separate TODO, DOING, and DONE copies of the same task. Historical ledger records, including old `ONGOING` records, remain untouched and readable.

The user explicitly wants:

- `DOING` under the visible **Planned** category/group.
- No `WIP` category.
- A fifth icon button on every planned TODO row: a Start/play button.
- Start changes the selected TODO into DOING.
- Completing a DOING changes it into DONE.
- Ditto extends the current task and aggregates time.
- Time totals accumulate on the single logical item.
- TODO and DOING continue to advance only to DONE in the mainline workflow.

## Current repository facts

The application is `/home/mde/proj/dunzo`; source is primarily in `dun/`. The current branch is `main`. Baseline `go test ./...` passes. The current UI has four Planned-row actions: Discard, Postpone, Done, and Edit. Ditto currently rewrites a DONE line to `ONGOING` and appends another DONE line. `ONGOING` is currently an internal EOD-only category.

Relevant areas:

- `dun/categories.go`: category registry, groups, picker visibility, and time-trackable categories.
- `dun/ui.go`: Planned rows, Endings, Start/Done actions, and Ditto.
- `dun/carryforward.go`: historical open-item scanning and daily carry-forward.
- `dun/todos.go`: open-item parsing and resolution behavior.
- `dun/ledgerentry.go` and `dun/itemrow.go`: ledger parsing and trailing metadata display.
- `dun/eod.go`, `dun/periodreview.go`, and `dun/standup.go`: open-item review/reporting.
- `docs/category-taxonomy.md` and `docs/GUIDE.md`: category and user workflow documentation.

## Implementation decisions

### Category model

1. Replace the current `ONGOING` category definition with `DOING`.
2. Use an appropriate doing/in-progress emoji and help text; keep it in group `plan` and make it selectable/visible as a normal Planned category.
3. Remove `EODOnly` treatment from the new DOING category. Historical `ONGOING` must still parse as a recognized legacy category where needed, but it must not appear as a current picker choice or be presented as the active DOING state.
4. Keep DONE as the terminal success state. Keep FAIL and WASTED as existing endpoint categories for their existing workflows.
5. Do not add WIP.
6. Update category comments, group labels, help text, and taxonomy documentation so Planned explicitly describes TODO/DOING as the open lifecycle and Endings describes DONE/FAIL/WASTED as endpoints.

### Ledger/state behavior

1. Mainline TODO/DOING/DONE transitions operate on the current ledger row in place when the row has a valid `LineIndex`. Preserve its timestamp and task text while changing its category.
2. Completing a TODO or DOING must preserve the task’s accumulated minutes and mark the resulting DONE as a lifecycle completion so the open logical item is resolved.
3. A direct manually entered DONE remains an independent DONE unless it was produced through a Planned-row transition or Ditto. Do not change unrelated historical semantics accidentally.
4. Existing carry-forward creates physical historical copies by day. Preserve that history, but change active-item collection/grouping so carried copies and state-transition copies of the same logical task collapse to one current item. The newest active state wins.
5. Use the existing task text/carry-forward identity conventions wherever possible. If a small transition marker is required to distinguish lifecycle DONE records from independent DONE records, make it structured and parser-owned, and preserve it when displaying task text. Do not rewrite old ledger files.
6. Old `ONGOING` entries are legacy history only. They remain readable in raw/history views and must not be converted or treated as new DOING entries during migration.
7. Postpone and Discard remain available on Planned rows. Preserve their existing SOMEDAY/DISCARDED resolution behavior unless the in-place state implementation requires a narrowly scoped adjustment. The required singular lifecycle is specifically TODO/DOING/DONE.

### Time aggregation and Ditto

1. Make trailing duration parsing robust for lifecycle metadata. Current `parseEntryMins` only recognizes a final `@Nm`; update the parser/formatter so a duration remains discoverable when transition metadata follows it, while keeping existing `@Nm` lines compatible.
2. Add small helpers to parse, replace, and increment the task’s cumulative `@Nm` value. Preserve non-duration text and metadata.
3. The Start action changes TODO to DOING and carries any existing duration unchanged.
4. The Planned Done action changes TODO or DOING to DONE and carries the cumulative duration unchanged.
5. Ditto acts as an extender for the latest current DONE/DOING lifecycle item:
   - If the latest relevant item is DONE, rewrite that row to DOING and preserve its existing duration.
   - If it is already DOING, keep one DOING row and add the newly entered minutes to its cumulative duration.
   - Do not append another DOING or ONGOING row.
   - When the user completes it, change that same logical item to DONE with the aggregate total.
6. Only explicitly entered, valid non-negative integer minutes are added. If no new minutes are supplied, retain the existing total. If existing duration metadata is malformed or absent, preserve the text and treat the prior numeric total as zero.
7. Clear the minutes input after Start, Done, or Ditto actions as appropriate and refresh all affected UI sections.

### Planned UI

1. Add a fifth icon-only button to every Planned TODO row, using the Fyne play/start icon (`theme.IconNameMediaPlay` or the repository’s chosen equivalent).
2. Place Start with the existing row actions and keep the established left-to-right action order unless visual inspection shows a clear usability issue. The button must have accessible tooltip/help text if the current icon-only convention supports it.
3. Start must target the captured row/item, rewrite its category to DOING, refresh Planned/Endings/accordion state, and show a short confirmation toast.
4. DOING rows must be visible in Planned. They should have the same row actions needed for the lifecycle, including Done and Edit; Start should only be available for TODO rows so it cannot restart an already active item.
5. Ensure Planned grouping includes DOING and that the default “TODOs only” view does not hide active DOING work. The UI should make the number of in-flight tasks visible enough to reveal when too many things are active.
6. Endings should show only terminal categories in its current category listing. A live DOING item belongs in Planned, not Endings.
7. Include DOING in EOD/period-review open-item handling and current-open standup context wherever TODO is treated as an active item. It must remain excluded from terminal-only summaries.

### Carry-forward and open-item parsing

1. Extend the open tracked category set to include DOING.
2. Teach resolution/collapse logic that TODO and DOING are states of one lifecycle item and that a lifecycle DONE resolves it.
3. Preserve existing behavior for GOAL, QUESTION, WAITING, FIXME, RISK, SOMEDAY, and other categories.
4. Ensure repeated carry-forward runs and transitions across multiple days do not produce duplicate visible Planned items.
5. Keep old ONGOING records out of the new active-state set while allowing them to remain parseable in history.

## Tests and acceptance checks

Add meaningful unit tests around behavior rather than implementation details. Cover:

- DOING exists in the category registry, is in Planned, is picker-visible, and has the expected time/picker behavior.
- TODO → DOING rewrites one row, preserves timestamp/text/minutes, and is idempotent against repeated Start attempts.
- DOING → DONE rewrites/resolves one lifecycle item and preserves cumulative minutes.
- Start appears on TODO rows and is absent or disabled for DOING rows.
- Ditto from DONE creates/continues exactly one DOING logical item.
- Repeated Ditto increments `@Nm` rather than adding rows.
- Duration parsing works with and without transition metadata and preserves malformed/missing durations safely.
- Carry-forward plus Start/Done across day boundaries collapses to one active logical item.
- Independent manually entered DONE behavior remains compatible.
- Historical ONGOING lines remain untouched and do not become active DOING items.
- EOD, period review, standup, Planned, and Endings classify DOING correctly.

Run:

```sh
gofmt -w dun/*.go
go test ./...
go vet ./...
make build
```

Manually inspect the Daybook UI for a TODO row, click Start, verify it moves to Planned as DOING, click Done, verify one DONE item and its total minutes, then use Ditto repeatedly with minutes to verify aggregation and row singularity.

## Assumptions for the implementation session

- “Singular” means one current logical item in the active UI and one current state in the ledger for the current day; prior daily carry-forward/history lines remain immutable.
- Mainline transitions use in-place rewriting and preserve timestamp/text; side exits (Postpone/Discard) keep their existing resolution records.
- Time aggregation uses explicit `@Nm` values only; there is no inferred elapsed-time calculation in this change.
- No historical migration is required for ONGOING.
- The plan is intentionally scoped to the Go/Fyne application and its category/workflow documentation; no changes are needed in the separate personal data repository.
