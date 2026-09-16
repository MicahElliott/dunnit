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
presents that stored date as a compact `🌱MM/DD` badge.

The carry operation is idempotent from the ledger's contents, which means a
second machine that has already received the copied lines does not append
duplicates. Resolutions written from Daybook remain append-only and sync with
the rest of the ledger.

The other open categories are Start of Day context rather than daily-plan
items:

- GOAL is longer-horizon planning.
- RISK is something to watch, not a task to repeat in Daybook.
- WAITING and QUESTION are reminders that may need attention, but are not
  automatically placed in today's plan.
- FIXME can be reconsidered later if it proves useful as a daily work item.

Start of Day may show these context items from the last active ledger day,
along with that day's EOD report and reflection fields. They are read-only
there.

## Stale items

The four-day display threshold remains a visual warning in Daybook. At seven
days old, an unresolved TODO or DOING item appears in Start of Day's stale
review. The user can explicitly move stale items to SOMEDAY; age alone never
changes an item's meaning or writes a ledger entry.

Stale review looks beyond the seven-day carry window so a missed Start of Day
does not make old items disappear without an explanation. Items outside the
carry window are not automatically brought into today's plan until the user
chooses how to handle them.

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
