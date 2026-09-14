# Dunnit Session Save — 2026-08-28 — Planning → Implementation Handoff

Personal project: `/Users/E463390/proj/dunnit` (git repo, direct commits
to `main` are fine). Latest commit: `6aefbc0` "Add PRD and FRD
brainstorm/planning docs".

## Where things stand

All planning/brainstorming is done and committed. Two docs at repo
root are the source of truth for what to build next:

- **`PRD-DRAFT-2026-08-28.md`** — the "why/what" brainstorm doc, fully
  resolved (category set, naming, MEETING semantics, SOD/SOM, two-file
  daily design, mini-calendar concept). Read this for *rationale* if a
  design question comes up mid-implementation.
- **`FRD-DRAFT-2026-08-28.md`** — the "how to build it" doc: every PRD
  item converted to a numbered `FR-01`...`FR-36`, each with acceptance
  criteria, phased into a proposed order (Phase 0 = bugs, 1 =
  foundational, 2 = TODO/GOAL workflow, 3 = meeting/agenda/SOD/SOM
  features, 4 = reporting/review, 5 = speculative/deferred). This is
  the actionable one — **start here**.

No app code has changed yet this round — this was a pure docs session.

## Immediate next step (where to pick up)

Load **Phase 0** from the FRD into `eca__task` as real tasks and start
implementing:

- **FR-01**: Suppress the periodic nudge if an entry was already
  logged within the current interval window.
- **FR-02**: Fix Cmd+W/Ctrl+W not hiding the window when focus is in a
  text-entry field (the common case).

Then **Phase 1** (foundational, no hard dependencies): FR-03 (add new
categories: `SOMEDAY`, `WAITING`, `FIXME`, `OPTIMIZE`, `QUESTION`,
`KUDOS`, `RISK`, revive `TODO`), FR-04 (nudge interval →
"every N minutes" config, replacing `hourly_minute`), FR-05 (drop
`MTG`, keep only `MEETING`), FR-06 (in-app Category Legend/tooltips).

## Working agreement for implementation phase

- **One commit per FR item** (or small logical chunk of one) going
  forward — gives clean history mapped to FRD numbering. (Docs-only
  work like this session was fine as one combined commit.)
- Still: lots of small checkpoints, ask before big-scope changes
  (per AGENTS.md).
- Always run `make build` / `go vet` / `editor_diagnostics` after
  edits — no CI here yet.
- Don't install new tools/deps without asking first.
- Manual GUI testing is done by Micah clicking around — describe
  exactly what to click/verify, don't assume you can confirm it
  yourself.

## Key resolved decisions worth remembering (condensed from PRD)

- `TODO` = small tight near-term item (encouraged). `GOAL` = bigger
  overarching aim, reviewed on longer cadence (SOM), not daily.
- `MEETING` = scratch agenda-builder for an *upcoming* meeting
  (tag-scoped, e.g. `#boss`), NOT attendance logging.
- Two daily files, permanently separate, both always hand-editable:
  the ledger (one-liners) and a markdown summary doc (free-form,
  LLM-drafted then hand-edited, never auto-regenerated over edits).
- "Daybook" = name of the main capture-window UI only. Product stays
  named "dunnit" — rename explicitly declined (FR-35: no-go).
- Streaks/consistency nudges: approved (FR-28), positive-only framing.
- More AI features: explicitly deferred (FR-30), no spec yet.
- Post-meeting capture (FR-36): early shape = quick multi-category
  session (`TIL`/`TODO`/`GOAL`/`RISK` + possibly more), tag-scoped to
  the meeting, expected to evolve with real usage.
- All popups/prompts are called **"nudges"** consistently now.
- No calendar integration (real `.ics`/`EventKit`) — but a tiny
  dunnit-native recurring-meeting mini-calendar (FR-15/16) is in scope,
  purely user-entered (tag, day-of-week, time).

## Full FR list location

See `FRD-DRAFT-2026-08-28.md`'s "Summary table" section for all 36
items at a glance with phase and dependency notes.
