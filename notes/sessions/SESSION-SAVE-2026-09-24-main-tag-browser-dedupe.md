# Session save — 2026-09-24 — main — tag browser deduplication

## Goal

Deduplicate the new tag browser and resolve the related tag-count discrepancy,
using the `#snap` ledger history in `mydunnits` as a representative case.

## Completed

- Tag history now shows the newest row for each logical TODO/DOING lineage.
- Deduplication removes all accumulated carry-forward markers, duration
  markers, and lifecycle resolution suffixes before comparing task text.
- Lifecycle leading verbs are normalized across base, present-participle, and
  past forms, while the rest of the task text remains part of identity.
- Same-day duplicate TODO/DOING rows collapse; independent unmarked terminal
  entries remain separate completed uses.
- Frecent tag totals and recent counts use the same shared deduplication path
  as the browser.
- The related `openItemKey` path now uses the same lifecycle metadata
  normalization.

## Files changed

`dun/carryforward.go`, `dun/tags.go`, `dun/tags_test.go`, `dun/todos.go`, and
`dun/todos_test.go`.

## Decisions

- A duration or carry-forward marker is metadata, not task identity.
- Multiple DONE records without lineage metadata remain countable because they
  can represent separate completed uses.
- Distinct normalized task text remains distinct even when it carries the
  same tag and source date.

## Verification

- `go test ./dun -count=1` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.

## Status

The five code and test files are staged on `main`; no commit was created.
Manual UI verification remains: open the tag browser against the `#snap`
history and confirm that each logical item appears once, with its newest time
and lifecycle category.
