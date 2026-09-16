# Session save — 2026-09-16 — main

## Goal

Make EOD and all scheduled auto-popups consistent with Dunnit’s new D icon and notification identity, and repair the EOD workflow and item rendering.

## Completed work

- Unified scheduled OS notifications under the `Dunnit` app identity and added the missing weekly digest notification.
- Kept the embedded `Icon.png` installed before tray and scheduler setup so Fyne can use it for native notifications and packaged app identity.
- Changed EOD to require an explicit `Generate` action before starting the AI summary.
- Added `Stop generating`, cancellation on window close, editable summary text, and a `Skip` action that appears after generation is requested.
- Made `Finalize Day` write the EOD report only when generation was requested and the user chooses to write it.
- Added per-day EOD handling state and report creation guards. A previously handled day now opens a blocking message and leaves any existing report unchanged.
- Applied Daybook-style category and carry-forward rendering to Today’s Items and postponed open items.
- Collapsed repeated or legacy `(since YYYY-MM-DD)` markers into one canonical indicator and preserved stale carry-forward badges.
- Renamed visible `DOING` section labels to `DOINGs`.
- Removed the obsolete automatic EOD draft path while retaining the old config key for TOML compatibility.
- Added focused tests for carry-forward normalization, EOD state, report non-overwrite behavior, and config preservation.

## Files changed

- `dunnit.go`
- `dun/sched.go`
- `dun/config.go`
- `dun/eod.go`
- `dun/dailysummary.go`
- `dun/carryforward.go`
- `dun/itemrow.go`
- `dun/periodreview.go`
- `dun/settings.go`
- `dun/carryforward_test.go`
- `dun/itemrow_test.go`
- `dun/eod_test.go`
- `docs/open-design-questions.md`

## Decisions

- Use a blocking “already handled” EOD message instead of offering an overwrite flow. This protects existing reports and avoids accidental duplicate completion.
- Treat an existing dated EOD report as already handled, including reports created through the report action outside the EOD popup.
- Keep `AutoDraftDailySummary` decodable but inactive so existing config files continue to load without preserving the old implicit-generation behavior.
- Use Fyne’s app-level icon and notification title because the current Fyne notification API does not expose a per-notification icon field.

## Verification

- `go test ./...` passed.
- `make vet` passed.
- `make build` passed.
- `git diff --check` passed.

## Known problems

- The completion desktop notification command could not connect to DBus in this environment (`notify-send: Operation not permitted`).
- UI behavior still needs manual verification by running the app and exercising the scheduler popups and EOD buttons, as documented by the project guidance.

