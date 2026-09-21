# Dunnit User Guide

## Why Dunnit

Dunnit is a factual, not aspirational, activity tracker. Once an hour,
it asks one question — "what are you working on?" — and appends your
answer to a plain-text daily ledger. No planning, no TODOs, no guilt:
just a running, honest record of what actually happened. Over weeks
and months that ledger becomes a searchable corpus you can turn into
standups, status reports, reviews, and even a resume — increasingly
with AI doing the heavy lifting. It's tiny, keyboard-driven,
cross-platform, and the data is always yours: plain text, optionally
synced with git, readable and editable with nothing more than a text
editor if you ever want to go tool-free.

### Why Dunnit, longer version

**Dunnit answers a different question than most productivity tools.**
TODO apps and planners ask "what should I do?" — a question about the
future, which is aspirational and often wrong. Dunnit asks "what are
you doing right now?" — a question about the present moment, answered
in the five seconds it takes to type a short line and dismiss a popup.
It's not about going back to reconstruct your day later; it's about
capturing *now*, hour by hour, so easily that it barely registers as
effort. The history and the reports are simply what falls out of doing
that consistently — a byproduct, not the point.

The mechanism is deliberately dumb: once an hour (configurable, and
only during your working hours), a small popup asks what you're
working on. You type a few words and hit enter. That's it — no
categorization required, no forms, no mouse. Over a day that becomes a
timeline; over a week, a status report; over a quarter, an impact
review; over a year, a resume's worth of raw material.

Everything about Dunnit is built to make that habit frictionless and
permanent:

- **It's tiny and fast** (a ~20MB desktop app) and **keyboard-first**,
  so answering the prompt never feels like a chore.
- **It's cross-platform** (macOS and Linux, wherever you actually
  work).
- **Your data is yours**: plain text files, one line per entry,
  organized by day/week/month. Sync them with git if you want history
  and backup, or query them via a lightweight local sqlite view if you
  want structure — but the source of truth is always human-readable
  text. If Dunnit disappeared tomorrow, you could keep doing this by
  hand in any editor.
- **It's open source**, so there's nothing hidden about how your data
  is stored or used.
- **It's highly configurable** but ships with sane defaults, closer in
  spirit to org-mode than to a heavyweight project-management tool —
  GUI-simple, without requiring Emacs.

The ledger itself borrows from older, well-tested ideas: the
append-only, timestamped-entry discipline of **accounting ledgers**;
the small, frequent, factual check-ins of **agile standups and
retros** (favoring "what happened" over big upfront planning); and the
daily-capture habit of journaling and bullet-journaling. Dunnit just
automates the prompting and the aggregation.

The long-term payoff is bigger than any single day's log: once you
have months of factual, tagged, timestamped data about your own work,
that corpus becomes something you can point tools at — AI-assisted
status reports, meeting prep and minutes, quarterly/annual reviews,
even resume-building — all generated from ground truth instead of
reconstructed from memory at review time.

### Guiding principles

- Focused on *now* — what you're doing this minute, not what you plan
  or hope to do.
- So simple/intuitive that almost no instruction is needed.
- Seamless, can't-forget integration into your day.
- Builds a corpus/picture of your whole work life that can be
  analyzed, automated, and mined for insights later.
- Mouseless-first, keyboard driven.
- Tiny (20MB desktop app).
- Similar to org-mode, but GUI, more intuitive, no Emacs required.
- Similar in scope to a TODO tracker, but a totally different
  approach.
- Highly configurable.
- Text-based, open data that you own.
- In a pinch, no tooling even needed — the process works by hand in
  any text editor.
- Ledger-based, plain-text storage; optionally synced via git, or
  queried via a 1-table sqlite view.
- Open source, for full transparency.
- Runs on any platform you use.
- Optional AI-powered reporting.
- Assistance/automation for meeting prep and minutes, status reports,
  quarterly/annual reviews, resume building.
- Key ideas borrowed from accounting (ledgers), agile software
  (standups/retros), and journaling practices.

## Methodology

Dunnit's methodology is a nested loop, the same "capture now, reflect a
little later" pattern repeated at increasing scale:

- **Hourly**: the popup captures a one-line entry, tagged with a
  category (`DONE`, `TODO`, `TIL`, `MEETING`, etc.).
- **Daily** (the heart of the process): *Start of Day* reads back your
  still-open items (`TODO`/`DOING`/`GOAL`/`WAITING`/`QUESTION`/`FIXME`/`RISK`)
  so you can decide what's still worth carrying; *End of Day* wraps things
  up — reviewing the log, scoring productivity, and noting meeting hours.

### Daily carry-forward

At Start of Day, Dunnit carries unresolved `TODO`s and `DOING`s from the most
recent prior workday with open plan items, looking back up to seven days while
skipping weekends and other configured off-days. Each item is copied into
today's Daybook once and keeps its original “since” date. `WAITING`s, `GOAL`s, `RISK`s, `QUESTION`s, and `FIXME`s are not copied
into today's plan; unresolved items from the last active day remain visible as
Start of Day context.

Items open for seven or more days appear in **Stale TODOs** for review. This
daily review looks back up to 30 calendar days and shows each item only once.
Stale items remain active in Daybook until you complete, postpone, or discard
them. A resolution also prevents later synced copies of the same item from
returning; a newly typed TODO without a carry marker may intentionally reopen
the work. Older items belong in `SOMEDAY` or history until you choose to bring
them back.

- **Weekly / Monthly / Quarterly / Annual**: the same Kickoff (forward-
  looking) and Review (backward-looking) shape repeats at each larger
  scale, with more built up at the larger scales (OKRs at
  quarter/year, an IMPACT/MILESTONE/WIN-driven narrative at year-end).

Two workflows run orthogonally to this calendar rhythm: **Meeting
Prep** (pull recent history on a topic/person before a meeting) and
**Post-Meeting Capture** (quickly log TILs/GOALs/RISKs/TODOs right
after). Standups draw automatically from the ledger since your last
standup.

The throughline: nothing here requires you to plan ahead correctly.
Every level just asks "what's still open?" and "what's happening
now?" — the same questions from the hourly popup, replayed at
week/month/quarter/year scale.

See the tray menu's **Kickoff**/**Review** submenus for the full set
of periods, and the README for setup/config details.

## AI / LLM CLI setup

Dunnit's report features can use a locally installed command-line client.
Dunnit does not ask for, receive, or store API tokens. Set up and authenticate
the CLI itself, then choose the client in **Settings → LLM CLI for Reports**.

The available choices are:

- **Auto (recommended)** — uses the first available client in this order:
  `copilot`, `claude`, `codex`, `gemini`, then `llm`.
- **Copilot** — GitHub Copilot CLI.
- **Claude Code** — Anthropic's Claude Code CLI.
- **Codex** — OpenAI Codex CLI.
- **Gemini** — Google Gemini CLI.
- **llm** — Simon Willison's `llm` CLI and whichever model/provider it has
  configured.

Pin a client when you care which model, account, privacy policy, cost, or
response style is used. Auto selection only falls through when a client is
missing. If a client starts and then reports an authentication, network, or
model error, Dunnit reports that error instead of retrying with another client;
retrying could charge another provider and produce a different report.

Install and authenticate the client outside Dunnit using its own setup flow:

- [Copilot programmatic use and setup](https://docs.github.com/en/copilot/how-tos/copilot-cli/automate-copilot-cli/run-cli-programmatically)
- [Claude Code CLI reference and setup](https://code.claude.com/docs/en/cli-usage)
- [Codex CLI](https://github.com/openai/codex)
- [Gemini CLI authentication](https://geminicli.com/docs/get-started/authentication/)
- [`llm` setup](https://llm.datasette.io/en/stable/setup.html)

Reports send the selected period's ledger text to the selected client. Treat
that text according to the client's account, local history, provider logging,
retention, quota, and billing policies. Dunnit asks the `llm` client not to
copy prompts and responses into its local log by passing `--no-log`; this does
not control retention by the remote model provider or by the other clients.

The report commands are one-shot and run without waiting for keyboard input.
Dunnit disables or restricts tools, MCP servers, extensions, project
instructions, and file access where each client supports those controls. The
ledger is still user-controlled text, so entries that look like instructions
can affect a model's response. Review generated reports before sharing them,
especially private/shareable status reports and annual reviews.

Each client may use a different default model and may have a different context
window. Long annual or multi-month reports can therefore be slower, cost more,
or exceed a provider's limit. Dunnit keeps its existing hierarchical review
flows, but a provider error may require a shorter period or a different pinned
client. Scheduled reports are also provider calls: enable them only when their
latency, quota, and cost are acceptable to you.

If Dunnit says no supported client is available, install one of the clients
above or pin a client that is already on your `PATH`. If it identifies a client
but reports an authentication error, run that client's login or setup command
in a terminal and try the report again. Dunnit does not perform login probes,
because those can open a browser or otherwise make network requests.

## Best Practices / How-To

- Tag entries with `#hashtags` for project names, ticket numbers, and topics
  so Navigator/Search/reports can filter by them later.
- Mark people with `@Name`, for example `@Brandon` or `@Surbhi`. Daybook
  adds a `👤` cue to entries containing people and keeps a short
  **Frequent people** row for quick insertion. People are indexed separately
  from `#tags`, which leaves room for feedback, KUDOS, and collaboration
  rollups by person in future reports.
- Time spent is written with a compact `~` marker, such as `~20m` or `~2h`.
- Don't over-categorize in the moment — `DONE` and `TODO` cover most
  entries; reach for the more specific categories (`TIL`, `KUDOS`,
  `WIN`, etc.) only when they clearly apply. Use the in-app **Help**
  window (below) if you forget what a category means.
- Let unresolved items carry forward rather than trying to
  close everything out each day — that's what Start/End of Day and the
  staleness badges are for.
- Use `SOMEDAY` freely as a pressure release valve for anything you're
  not ready to commit to; revisit it via the SOMEDAY browser.
- Prefer the keyboard over the mouse throughout — most entry fields
  and pickers are built for it.
- Add links inline with Markdown, for example
  `[Jira #74750](https://example.atlassian.net/browse/ABC-74750)`. Bare
  `https://...` URLs are also clickable in presentations; Dunnit keeps the
  full URL in the ledger and may show a compact service label such as
  `Jira #74750`, `GitHub PR #12`, `Teams`, or `Google Doc`.

For install, configuration, and data-storage details, see the
top-level [`README`](../README.md) rather than this guide.

## Category Legend

_Kept in sync with the in-app **Help** window (`showHelp` in
`dunnit/ui.go`), which is generated directly from
`dunnit/categories.go`'s `Categories` list — that file is the
authoritative source; update this section whenever categories change._

**End** — day-to-day capture, the terminal states a Plan item resolves into

- ✔️ `DONE` — Something you completed. The most common endpoint a "Plan" item (TODO/IDEA/GOAL/etc.) resolves into.
- ❌ `FAIL` — Something that didn't go as hoped — an endpoint a "Plan" item can resolve into, same as DONE, just the unsuccessful outcome.
- 🗑️ `WASTED` — Unfocused, pointless work or distraction. Opt-in: hidden from the live picker unless enabled in Settings.

**Plan** — future-facing, open items tracked toward a resolution to DONE in "End"

- 📌 `TODO` — A small, tight, near-term item — actively encouraged. Roughly Jira's "Task": scoped and ready to act on (vs. IDEA, which is the same thing before it's scoped).
- ▶️ `DOING` — A TODO currently in progress. It remains the same logical item until it reaches DONE.
- 💡 `IDEA` — A new idea worth capturing, not yet scoped/ready to act on — an earlier maturity stage of TODO.
- 🎯 `GOAL` — A bigger overarching aim, reviewed on a longer cadence (not daily). Roughly Jira's "Epic".
- ❓ `QUESTION` — An open question to follow up on.
- ⏳ `WAITING` — Blocked on someone/something else; not actionable right now.
- 🔧 `FIXME` — Something broken that needs fixing — roughly Jira's "Bug".
- ⚠️ `RISK` — A risk worth flagging/tracking.
- 📅 `MEETING` — Scratch agenda-builder notes for an upcoming meeting (tag-scoped).
- 🕰️ `SOMEDAY` — Something you might want to do eventually, not now (also where stalled TODOs/GOALs land).
- 🏎️ `OPTIMIZE` — Something working but worth improving/speeding up.

**Hilite** — freestanding notable-moment callouts, not tied to resolving any specific Plan item

- 🌱 `TIL` — Today I Learned — something new you picked up.
- 🙌 `KUDOS` — Recognition given to someone else, or received from someone else.
- 🏆 `WIN` — A distinct, successful moment or completed task.
- 📢 `PSA` — An announcement or heads-up the team should know about.
- 💪 `OVERCOMING` — A time you turned a failure, roadblock, or crisis into a recovery.
- ✨ `INNOVATION` — You created a new process, tool, or idea from scratch.
- 👑 `LEADERSHIP` — A moment you mentored someone, led an initiative, or influenced a decision without direct authority.
- 💥 `IMPACT` — The measurable result or value your action caused, not just what you did.
- 🏁 `MILESTONE` — A significant checkpoint or phase transition in a longer journey, bigger in scope than a single WIN.
- 💼 `CAREER` — A big, resume/CV-worthy accomplishment.

_(A few additional categories — legacy `ONGOING`, `SUMMARY`, `PRODUCTIVITY`,
`MEETING_HOURS` — are written automatically by internal flows like
Ditto and End of Day, and aren't meant to be hand-picked; they're
omitted here as they are from the in-app Help window.)_
(`showHelp` in `dunnit/ui.go`)._
