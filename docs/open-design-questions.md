# Open Design Questions

Running log of things flagged mid-implementation that need more
thought before considered fully resolved. Not blocking current work
unless noted.

## FR-18 (persistent daily summary doc) vs. existing Summarize "Day"

The EOD window now keeps report generation explicit: the user taps
`Generate`, can stop the request, edits the draft if desired, and chooses
`Finalize Day` or `Skip`. Finalize writes the report only after Generate was
chosen, and an existing report is never overwritten. The old
`auto_draft_daily_summary` setting remains accepted when decoding older
config files but has no effect.

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
2. **Should the EOD prompt offer a re-run action?** The current behavior
   blocks after completion so a scheduled popup cannot create duplicate
   metadata or replace an existing report. A future explicit overwrite
   action could be added if re-running EOD becomes necessary.

The remaining question is whether the report should eventually diverge more
from Summarize→Day; that is independent of the EOD interaction and storage
behavior.
