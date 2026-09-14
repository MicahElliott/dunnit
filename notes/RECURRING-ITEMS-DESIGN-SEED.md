# Dunnit: Recurring TODOs/GOALs -- Design Seed

Status: **implemented (v1)**. Decisions below were made and acted on
in the follow-up session; see "Implementation notes" for what
actually landed.

## Problem

Daybook has one-off TODO/GOAL capture (typed in as needed) and
recurring *meetings* (FR-15's mini-calendar, `[[recurring_meeting]]`
in config.toml), but no concept of a recurring *task* -- something
the user wants reminded of on a repeating cadence (daily/weekly/
monthly) without having to re-type it from scratch each time, and
without it silently getting forgotten if they don't think to log it
that day.

## Decisions made (resolving the four open questions below)

1. **Surface as suggestion**, not auto-seed. Each due item shows up
   with an explicit "Add" button; nothing is written to the ledger
   until the user taps it.
2. **Management GUI** -- a `showRecurringItemsDialog` (add/edit/
   delete), reachable from Settings, not hand-edit-`config.toml`-only.
3. **Weekly piggybacks on the existing daily SOD nudge** -- no new
   "Start of Week" concept was built; SOD's cadence check just also
   asks "is today the configured day-of-week for this weekly item."
4. **SOM-seeding done now, not deferred further** -- SOM's wizard
   gained a 5th step surfacing monthly recurring items as suggested
   Adds, functioning as the "month checklist" originally asked for.

## Implementation notes

- New file `dunnit/recurring.go`:
  - `RecurringItem` struct (`Category`, `Text`, `Cadence` one of
    daily/weekly/monthly, `DOW`, `DayOfMonth`) -- deliberately
    simpler than `RecurringMeeting` (no time-of-day).
  - `isDueToday`, `clampDayOfMonth` (31 clamps to e.g. 28/30 in
    shorter months), `alreadyLoggedToday` (dedup against today's open
    items by exact category+text match), `dueRecurringItems(cfg, now,
    cadence)` (cadence="" matches any).
  - `recurringItemsSuggestionBox` -- shared "Add" row builder used by
    both SOD and SOM.
  - `showRecurringItemsDialog` -- the management GUI, modeled on
    `showMiniCalendarDialog`.
- `config.go`: added `Config.RecurringItems []RecurringItem` (`toml:
  "recurring_item"` array-of-tables).
- `sod.go`: shows due daily+weekly items as suggestions above the
  entry row; adding one refreshes both the open-items list and the
  recurring suggestion box.
- `som.go`: new "5. Monthly Recurring Items" step showing due monthly
  items as suggestions.
- `settings.go`: added a "Recurring Items..." button opening the
  management dialog, alongside the existing "Recurring Meetings...".

Verified via `make build` (clean) and `make vet` (clean); no
automated tests in this repo yet, so manual click-through in the app
is still needed to confirm the SOD/SOM suggestion flow and the
management dialog's add/edit/delete behavior end-to-end.

## Not yet done / possible follow-ups

- No manual click-through testing performed yet (per repo convention,
  that's on the human).
- Editing an existing recurring item isn't supported by the GUI --
  only add/delete. Edit-in-place could be added if it turns out to be
  needed (currently: delete + re-add).
- No dedup/interaction consideration for a recurring item whose text
  happens to collide with a manually-logged item that isn't an exact
  string match (e.g. slightly reworded) -- `alreadyLoggedToday` only
  catches exact matches, by design (kept simple per "just a couple,
  don't overbuild" framing).
