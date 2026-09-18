# Session handoff — 2026-09-18 — main — age badges

## Goal

Replace the Daybook carry-forward sprout and separate stale warning with a
single age cue. The final visible form is a colored ball plus age, such as
`🟡2d`, `🟠6d`, or `🔴10d`.

## Completed

- Shared item-row metadata now uses yellow for days 0–3, orange for days 4–7,
  and red for day 8 onward.
- The compact badge shows the age (`🟡Nd`) rather than the absolute `MM/DD`
  date. Its hover text says `Open for N days`.
- The stored `s/YYYY-MM-DD` marker remains unchanged and continues to supply
  the age calculation.
- The shared renderer applies the cue across Daybook and the other views that
  use `itemTextLabel`.
- Carry-forward design documentation and boundary tests were updated.

## Look-back decision

Start of Day stale review scans 30 calendar days, so it covers items older
than seven days and can reach the red 8+ day band. Automatic daily carry-
forward searches only the previous seven calendar days; that is a separate
window and remains unchanged.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

The prior feature work is staged by the user. The age-only follow-up changes
and this session note are currently unstaged; no commit or staging was done.
Manual UI verification remains: open Daybook with carried items aged 1–3,
4–7, and 8+ days and confirm the three badges visually.
