Usage model:

- each day gets a dated file that can live/grow on multiple machines
- sync across 2 machines
- log-style file
- only a couple times per day
- never concurrent
- only additive entries, but an entry could be edited on the machine in which it originated
- ok to always pull before push
- one-line entries only, timestamped to the minute
- all entries are obviously unique (no similar lines)
- would be nice to avoid pulling in a git package

Not sure where to sync to; could be:

- shared mounted dir (dropbox, icloud, etc)
- or to service (to local file that's push to service (dropbox, etc))
- git
- does it matter?

I wrote to LLM:

> I don't care about order on entries in file. Also, no similar entry would
> show up from the two sources, so no almost-matching lines problems. I'd
> rather not have to read before every write, but maybe that's fine too --
> it's more about push/pull syncing. So I'd guess it doesn't matter with this
> model if it's dropbox or icloud or local-with-manual-push-pull, right?

Its response:

> Correct. If you do not care about the order of lines and the lines are
> entirely unique, the underlying storage layer becomes irrelevant. Whether it
> is Dropbox, iCloud, a Git repository, or a manual file-transfer script, the
> core synchronization rules remain exactly the same.

> Because you want to avoid reading the remote file on every single local log
> write (which slows down performance), you can decouple your app into a
> Local-First Write system with a Periodic Sync Step.

Let me know your thinking on how to simply design this.
