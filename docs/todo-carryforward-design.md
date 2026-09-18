# TODO Carry-Forward Design

These are the current design notes for daily planning and carry-forward.

## Daily planning

Start of Day is the one place where daily carry-forward happens. Opening
Daybook, recording the first entry, or using another machine does not copy
anything automatically.

When Start of Day runs, it searches the previous seven calendar days from
newest to oldest. It chooses the newest day whose unresolved daily-plan items
include TODO or DOING entries, then copies only those items into today's
ledger. It does not combine items from multiple days. Each copied item keeps
its original `s/YYYY-MM-DD` annotation so its age remains visible. The UI
presents the item age as a compact colored-ball/day badge: yellow for days
1–3, orange for days 4–7, and red for day 8 onward.

Those copied lines are today's active plan, so they appear in Daybook's
Planned section and remain there until they are completed, postponed, or
discarded. Start of Day may show the same logical item once in stale review
when it has been open for at least seven calendar days; stale review is a
decision prompt, not another task copy. It deduplicates with the same logical
TODO/DOING key used by Daybook and shows the item's original `since` date.

The carry operation is idempotent from the ledger's contents, which means a
second machine that has already received the copied lines does not append
duplicates. Resolutions written from Daybook remain append-only and sync with
the rest of the ledger. A resolution also prevents later synced copies of the
same carried item from resurrecting it; a newly typed TODO without a carry
marker may intentionally reopen the work.

The other open categories are Start of Day context rather than daily-plan
items. SOD shows unresolved context from the last active ledger day, but does
not write those entries into today's ledger or Daybook's Planned section:

- GOAL is longer-horizon planning.
- RISK is something to watch, not a task to repeat in Daybook.
- WAITING and QUESTION are reminders that may need attention, but are not
  automatically placed in today's plan.
- FIXME can be reconsidered later if it proves useful as a daily work item.

Start of Day may show these context items from the last active ledger day,
along with that day's EOD report and reflection fields. They are read-only
there.

## Stale items

The colored age badge is a visual cue in Daybook. At seven days old, an
unresolved TODO or DOING item appears in Start of Day's stale review. The user
can explicitly move stale items to SOMEDAY; age alone never changes an item's
meaning or writes a ledger entry.

Stale review scans the previous 30 calendar days, so a missed Start of Day has
some recovery room without turning the daily surface into an archive. Daily
carry-forward still searches only the previous seven calendar days. Items
outside the daily horizon are not brought into today's plan; use SOMEDAY or
history when they become relevant again.

## Resolution and sync

The existing append-only resolution entries are the shared decision record:

- DONE means the item was completed.
- SOMEDAY means the item was deliberately deferred.
- DISCARDED means the item was deliberately dropped.

The daily carry decision belongs in ledger data. A per-machine automatic
carry-forward trigger would let another machine resurrect items that the
active machine intentionally removed. Git sync remains independent of this
workflow; automatic sync should not be used as a substitute for the Start of
Day decision.
