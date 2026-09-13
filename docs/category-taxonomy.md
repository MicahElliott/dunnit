# Category Taxonomy -- Design Discussion

Status: informal design notes, not a spec. Captures reasoning behind
the current `Categories` grouping (`dunnit/categories.go`) and open
questions for future discussion. Updated 2026-09-12 for the DOING
lifecycle -- read this before
changing group membership or adding new categories.

## The core insight: "endpoints"

Most `plan`-group categories (TODO, DOING, IDEA, GOAL, QUESTION, WAITING,
FIXME, RISK, SOMEDAY, OPTIMIZE) represent an *open item*: something
logged now, tracked, and expected to eventually resolve. They all
share the same lifecycle shape: **logged -> tracked as open ->
resolved**. TODO has an explicit active state: **TODO -> DOING -> DONE**.
The state changes keep one logical task in the active Daybook view; daily
carry-forward history remains readable.

**DONE, FAIL, and WASTED are the resolution states of that lifecycle**
-- the "endpoints" a plan-group item can land on. This is why they
belong conceptually with "End" (day-to-day capture, logged in the
moment) rather than "Hilite" (which has a "retrospective, look-back"
framing) -- resolving an item isn't a retrospective act,
it's just as much a moment-of capture as logging the item was.

This reframing came from noticing that DONE was already in "End" while
FAIL/WASTED were oddly siloed in "Hilite" despite being the *same kind
of thing* as DONE (just the unsuccessful outcomes). They belong with
DONE as terminal endpoints.

`DOING` belongs in Plan because it is an active, user-visible state. Old
`ONGOING` lines are legacy history from the earlier Ditto implementation;
they remain readable but are not treated as current open work.

## What's left in "Hilite" -- and is it still coherent?

After keeping the endpoints together, "Hilite" contains:

- **IMPACT, MILESTONE, CAREER** -- freestanding notable-event markers.
  These don't resolve any specific open item; a CAREER note might have
  no relationship to any TODO ever logged. This is a genuinely
  different concept from an endpoint: it's "something worth flagging
  happened," not "an open item concluded."
- **SUMMARY, PRODUCTIVITY, MEETING_HOURS** (`EODOnly`) -- day-level
  meta-notes, always machine-written by `eod.go`'s Finalize Day flow.
  Arguably a *third*, distinct concept (day-level stats/wrap-up, not
  itself a loggable "event" at all) -- currently left bundled into
  "Hilite" for simplicity rather than split into its own group.

So "Hilite" is arguably still doing double duty (freestanding
notable-events + day-meta), but this was judged less confusing than
the original three-way conflation (endpoints + notable-events +
day-meta) and left as-is for now. A future split into a 4th group
(e.g. "Daily Wrap" for the EODOnly trio) was discussed but not done --
revisit if "Hilite" still feels muddled in practice.

Renaming "Hilite" itself (e.g. to "Major") was considered but not
done -- IMPACT/MILESTONE/CAREER genuinely are reflective in nature
(recognizing in hindsight that something was significant), so the
word wasn't the actual source of confusion; the endpoint categories
sharing the bucket were.

## Current grouping (post-2026-09-12)

- **End** -- terminal endpoints: DONE, FAIL, WASTED
- **Plan** -- open/tracked items: TODO, DOING, IDEA, GOAL, QUESTION, WAITING,
  FIXME, RISK, MEETING, SOMEDAY, OPTIMIZE
- **Hilite** -- freestanding notable-event markers + day-meta: TIL,
  KUDOS, WIN, PSA, OVERCOMING, INNOVATION, LEADERSHIP, IMPACT,
  MILESTONE, CAREER, SUMMARY*, PRODUCTIVITY*, MEETING_HOURS*
  (*EODOnly)

## The Jira/GitHub-Issues parallel (TODO/FIXME/IDEA/GOAL)

Comparing to scrum/issue-tracker taxonomies (Jira issue types, GitHub's
default Issues labels: Bug, Enhancement, ...):

| Jira/GitHub | Dunnit | Relationship |
|---|---|---|
| Bug | FIXME | Same concept: a genuinely different *type* of item (fixing vs. building), not a maturity stage of TODO. |
| Story/Feature (un-scoped) | IDEA | IDEA is essentially an un-scoped Story/Feature -- an earlier maturity stage of TODO, not a sibling type the way Bug is. |
| Task | TODO | Generic, scoped, ready-to-act-on unit of work -- the "whatever thing that needs to get done" catch-all. Notably, traditional issue trackers don't have a direct TODO-app-style equivalent; Task is the closest analog. |
| Epic | GOAL | Bigger rollup, longer review cadence, not itself a single actionable item. |

Key finding from this comparison: **the real asymmetry is FIXME vs.
TODO** (broken vs. new-thing-to-do), mirroring Jira's core Bug vs.
Feature/Story split -- not FIXME vs. IDEA. IDEA/QUESTION/SOMEDAY are
better understood as maturity stages on the *same* TODO axis, not
sibling "types" the way FIXME is a sibling type to TODO.

**Practical implication, not yet built:** a "Bugs (FIXME) vs. everything
else" split in the Planned/Upcoming view could be a useful triage lens
(bugs often want faster turnaround than a low-priority IDEA) -- flagged
as a possible future feature, not implemented. No urgency; revisit if
a concrete workflow need shows up.

**TODO's genericness / naming baggage:** TODO is unusually generic
compared to every other Plan-group category (which are all specific,
obvious-at-a-glance concepts). A look at real logged TODOs showed
recurring shapes: "Set up...", "Talk to...", "Prep for...", "Explore
if...", "Create a...". This confirms there's a real need for a
generic "thing that needs to get done" bucket -- Dunnit can't cleanly
get rid of it, and no clearly-better replacement word was found. TODO
also carries some baggage (it evokes generic "Todo apps," a category
Dunnit is deliberately not trying to be) but renaming was judged not
worth pursuing without a concrete better alternative in hand. Revisit
only if a good replacement word surfaces.

## "(from X)" promotion-annotation convention

When a Plan-group item resolves into an endpoint (DONE/FAIL/WASTED),
it's useful to annotate the endpoint entry with where it came from,
e.g. a DONE line noting `(from TODO)`. Current status:

- **This is currently an informal, hand-typed convention** (used in
  session narration), not something any code path writes
  automatically. Most Daybook items are logged directly with a
  category chosen up front; there's no formal "promote this open item
  to DONE" UI action yet that would stamp this automatically.
- **Applies only to true promotions of Plan-group items into an
  endpoint** -- explicitly does *not* apply to legacy ONGOING records.
- **Future direction, not yet built:** if this is to become genuinely
  trackable/analyzable (e.g. "% of IDEAs that become TODOs then DONE,"
  FIXME time-to-resolution stats), it needs:
  1. A single code path for "mark this open item as resolved" (e.g.
     from the Upcoming/Planned list) that both writes the endpoint
     line *and* stamps the annotation, rather than users just typing a
     fresh DONE line by hand.
  2. A more structured annotation format than free prose -- e.g. a
     tag-like `#from:TODO` (compatible with existing tag
     infrastructure: autocomplete, `KnownTags()`, etc.) rather than a
     bespoke `(from X)` parse target, if real analysis is wanted later.
- Deliberately **not** turned into an enforced state machine --
  Dunnit's design ethos is lightweight/manual logging, not a workflow
  engine. Exceptions are expected in practice (e.g. a WIN or KUDOS
  could also "close out" something; TIL sometimes documents resolving
  a QUESTION) -- a strict FSM would fight this rather than help it.
  Document the pattern and use "(from X)"/`#from:X` as a lightweight
  instrument; only consider formal funnel-stats tooling once there's
  a few months of real tagged data to see if the concept holds up.

## Open questions (not resolved, revisit later)

1. Should SUMMARY/PRODUCTIVITY/MEETING_HOURS (EODOnly day-meta) split
   into their own 4th group, now that "reflect" is narrower?
2. Is there a good non-generic replacement word for TODO? (Nothing
   found yet; not worth pursuing without one.)
3. Should the Planned/Upcoming view eventually split FIXME out
   visually from other Plan items (Bug-vs-rest triage lens)?
4. Should "(from X)" become a real, code-enforced convention (tag-based,
   written by a "mark resolved" UI action), or stay purely informal?
