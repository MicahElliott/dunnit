# Open Design Questions

Running log of things flagged mid-implementation that need more
thought before considered fully resolved. Not blocking current work
unless noted.

## FR-18 (persistent daily summary doc) vs. existing Summarize "Day"

Raised 2026-08-29. FR-18's auto-draft was gated behind a new
`auto_draft_daily_summary` config flag (default `false`) pending
resolution of these questions; manual drafting via the "Daily Summary
Doc..." tray item is unaffected and still available on demand.

Open questions:

1. **What actually differentiates FR-18's doc from Summarize→Day
   output?** Right now the initial drafted content is functionally
   identical (same prompt, same pipeline, same ledger scope) --  the
   only novel behavior is persistence (durable `.md` file) and the
   never-overwrite-once-created guarantee. Is that alone enough value
   to justify a second on-disk artifact next to the ledger, or should
   the doc's content/purpose diverge from Summarize in some way (e.g.
   a different, more reflective prompt; freeform template instead of
   LLM-drafted; something else)?
2. **Is EOD the right auto-trigger timing at all?** EOD could happen
   mid-afternoon for some workflows, and since the draft is locked in
   once created, an early draft risks capturing an incomplete/stale
   snapshot of the day. Alternatives: trigger at actual midnight/next-
   day rollover, trigger on-demand only (no auto-trigger), or make the
   draft explicitly re-openable/re-draftable (relaxing the never-
   overwrite guarantee somehow, e.g. append additional LLM passes
   rather than skip entirely if content already exists).
3. **Should the auto-draft even shell out to an LLM by default?**
   Considered separately from timing -- shelling out to a configured LLM CLI
   unprompted on every EOD (once enabled) has real latency/cost
   implications, similar to the concern that led FR-19's weekly
   digest to default off. Current fix (this same flag) addresses this
   for now, but worth revisiting once there's real usage data on
   whether the auto-draft is used/valued at all.

No decision made yet on any of the above -- flagging for future
thought when picked up again, not blocking other FR work.
