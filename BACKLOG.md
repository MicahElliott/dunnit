# Dunnit backlog

This is the short, living implementation queue. Keep completed history and
large product exploration in [FRD-DRAFT-2026-08-28.md](FRD-DRAFT-2026-08-28.md),
which is now a historical requirements document. Add a checkbox here only when
there is a concrete next step that can be implemented and checked off.

## How to use this file

- `Now` is the small set of items worth actively implementing.
- `Next` is refined enough to pick up after `Now`.
- `Later` is recorded but still needs product or technical discovery.
- Change `[ ]` to `[x]` only when the acceptance checks are satisfied.
- Put new questions under the item rather than making a vague feature entry.
- Re-estimate after investigation. Effort is a rough engineering size: S is a
  focused change, M touches several parts of the app, and L is a small project.

The intended loop is: choose one `Now` item, clarify its open decisions, split
it into implementation steps if needed, build it, run the normal checks, then
check it off and move the next item into `Now`.

## Now

### [ ] Fix TODO → DONE conversion leaving the TODO behind

**Kind:** bug · **Effort:** S/M · **Area:** ledger mutation, TODO UI

When a TODO is completed through the app, the source TODO currently remains
intact after the inflection change. Completion should consume or update the
source item according to the app's current ledger model, so the same work is
not still presented as open.

Acceptance checks:

- [ ] Reproduce the current behavior with a real TODO.
- [ ] Completing it results in exactly one completed item in the open-items
  view; the stale TODO is gone or changed to the chosen completed form.
- [ ] Restarting the app does not resurrect the TODO.
- [ ] Add or update a focused regression test for the ledger transformation.
- [ ] `make build` and `go vet ./...` pass.

Open decision: confirm whether the desired ledger behavior is an in-place
replacement or an append-only DONE plus an explicit closed marker. The current
FRD describes append-only behavior, while actual usage now expects the TODO to
stop remaining intact; settle this from the current code and ledger history
before implementing.

### [ ] Make window titles consistent

**Kind:** polish · **Effort:** S · **Area:** Fyne windows

Every app window title starts with `Dunnit: `, followed by its specific title.

Acceptance checks:

- [ ] Find every window constructor and apply the prefix consistently.
- [ ] Check the main window, settings, dialogs, and report/review windows
  manually.
- [ ] No title receives the prefix twice.

### [ ] Make list presentation and editing consistent

**Kind:** UI cleanup · **Effort:** M · **Area:** shared list/item widgets

All list-like screens use the same bullet treatment, row spacing, edit affordance,
and interaction conventions. Avoid one-off list implementations where an
existing shared row/widget can cover the behavior.

Acceptance checks:

- [ ] Inventory the list-like screens and identify the shared presentation
  primitive.
- [ ] Lists use proper, consistent bullets and alignment.
- [ ] Editable lists expose the same edit/save/cancel behavior.
- [ ] Read-only lists remain clearly read-only.
- [ ] Manually check the affected screens at normal and long-content sizes.

Open decision: define the small shared list convention after the inventory;
this item should be split if it turns out to contain independent list bugs.

### [ ] Keep the quit action available

**Kind:** bug · **Effort:** S · **Area:** tray/menu lifecycle

The Quit button currently disappears after some time. It remains available for
the full lifetime of the app and still exits cleanly.

Acceptance checks:

- [ ] Reproduce the disappearance and identify whether the menu is rebuilt,
  replaced, or hidden by a timer.
- [ ] Quit is present after startup, after a nudge, and after opening/closing
  other windows.
- [ ] Quit exits cleanly from each relevant tray/menu path.

## Next

### [ ] Edit or add yesterday's DONEs during today's kickoff

**Kind:** feature · **Effort:** M · **Area:** kickoff flow, ledger editing

Today's kickoff can show yesterday's DONE entries and lets the user correct an
entry or add a missed DONE. The resulting ledger remains parseable and the
operation is clear about which date it changes.

Acceptance checks:

- [ ] Yesterday's DONE entries are visible from the kickoff flow.
- [ ] An existing entry can be edited and saved to yesterday's ledger.
- [ ] A missed DONE can be added to yesterday's ledger with an appropriate
  timestamp/date treatment.
- [ ] Cancel leaves the historical ledger unchanged.
- [ ] The next kickoff reflects the saved result after an app restart.
- [ ] Add focused tests for parsing and the selected write/update behavior.

Open decisions: whether editing preserves the original timestamp; whether a
late-added DONE uses its actual entry time or a user-selected time; and whether
this belongs in the existing SOD/kickoff screen or a small dedicated editor.

### [ ] Add minimal git sync for the ledger directory

**Kind:** feature · **Effort:** M/L · **Area:** config, process execution, UX

Add an opt-in sync workflow for the configured ledger directory. Start with the
system `git` executable so the first version stays small and uses the user's
existing Git configuration and credentials.

This is one feature with staged delivery:

- [ ] Detect whether the configured directory is a Git repository and report
  actionable errors.
- [ ] Provide first-use bootstrap: choose/confirm the directory, run `git init`,
  create the expected initial structure, and make the first commit when there
  is content to commit.
- [ ] Add a Sync action that fetches/pulls, rebases local work as appropriate,
  commits local changes with a predictable message, and pushes.
- [ ] Surface command progress and failures in the app; do not silently lose
  local ledger changes.
- [ ] Define behavior for uncommitted unrelated files, conflicts, no remote,
  and an empty repository before wiring the UI.
- [ ] Manually verify the workflow with a disposable local bare remote.

Initial technical choice: shell out to `git`; revisit `go-git` only if the
process-based implementation cannot provide the required error handling or
cross-platform behavior. Keep the sync engine separate from the button so it
can be tested without Fyne.

### [ ] Make the mydunnits directory configurable in Settings

**Kind:** configuration · **Effort:** S/M · **Area:** settings, paths

Allow the user to configure the ledger directory from Settings, while retaining
the environment variable as a useful override or migration path.

Acceptance checks:

- [ ] Settings displays the effective directory and lets the user choose/save a
  directory.
- [ ] The choice persists in config and takes effect without ambiguous mixing
  of old and new paths.
- [ ] Existing `$DUNNIT_DIR` users do not unexpectedly lose access to their
  ledgers.
- [ ] Invalid or inaccessible directories produce a useful error.

Open decision: establish precedence between the Settings value and
`$DUNNIT_DIR`; document it before implementation.

## Later

### [ ] Add person-aware feedback and collaboration rollups

**Kind:** feature · **Effort:** M · **Area:** reports, navigator, people trackable

Use the `@Name` references already captured in ledger entries to show who was
worked with most during a selected period, gather person-scoped DONE/KUDOS/WIN
evidence for feedback and end-of-year reviews, and compare sentiment or
productivity signals across person-heavy days without changing the raw ledger
format.

Acceptance checks:

- [ ] Navigator can filter a period by one or more people.
- [ ] A report can show counts, categories, and linked evidence for a person.
- [ ] Person-heavy days are visible as an aggregate without treating mention
  count alone as positive or negative sentiment.
- [ ] Manual review confirms that the wording makes clear these are mentions,
  not an objective measure of relationship quality or performance.

### [ ] Investigate Turso sync and a SQLite-backed data model

**Kind:** discovery · **Effort:** L · **Area:** storage, sync architecture

Explore whether Turso is a useful future sync/storage layer and what changes a
SQLite model would require. This is intentionally a spike, not a commitment to
replace the current text-ledger format.

Deliverables:

- [ ] Document the current ledger entities, append/edit semantics, and sync
  invariants.
- [ ] Compare text files plus Git, local SQLite plus replication, and Turso
  against offline use, inspectability, portability, cost, and conflict handling.
- [ ] Build a small proof of concept only if the comparison identifies a clear
  advantage.
- [ ] Record a go/no-go decision and migration implications.

Dependency: learn from the minimal Git sync behavior first; avoid designing two
sync systems at once.

## Parked from the old FRD

The remaining FRD items are still useful product history and source material,
but are deliberately not active queue entries. Promote one here when it becomes
something we intend to build, with acceptance checks and an effort estimate.
