# Dunnit business plan

This is a working product and business plan. It captures the current
direction for Dunnit's product positioning, data model, paid offerings, and
next validation steps. Pricing and hosted-service details are hypotheses to
test with users.

## Executive thesis

Dunnit is a desktop-first, factual work journal that turns small, timely
check-ins into an owned record of a person's professional life.

The immediate benefit is knowing what happened today. The accumulated benefit
is a current, evidence-backed source for standups, status reports, reviews,
retrospectives, client updates, and resumes.

The central promise is:

> Dunnit keeps an up-to-date record of your work in files you own, so your
> next resume, review, or status report starts from evidence instead of memory.

## Product positioning

Dunnit answers a different question from a task manager.

- A planner asks, “What should I do?”
- Dunnit asks, “What are you doing right now?”

That distinction makes the product factual rather than aspirational. A small
popup appears while the user is already working, accepts a short answer, and
gets out of the way. The user resumes the original task immediately. Dunnit
does not require a later reconstruction of the day or a trip into a separate
productivity world.

The best short description is:

> Your work, captured where it happens. Your professional history, always
> getting better.

Other useful positioning lines:

- Write it once. Reuse it for every professional narrative.
- The factual work log behind your next resume.
- A personal evidence base for the work you actually do.
- The open, text-first memory of your professional life.

## Why desktop first

Desktop is the primary capture surface because that is where most knowledge
work happens. Dunnit can briefly take focus, ask one question, accept a few
words, and return focus to the task already underway. This is a deliberate
design advantage over phone-centered productivity products:

- the prompt arrives in the user's working context;
- capture takes seconds instead of requiring a context switch;
- the user does not need to reach for a distracting phone;
- keyboard-first entry keeps the interruption small;
- the original task remains the center of attention.

The mobile app has a different job. It is a slim companion for moments when
the user is away from the desk: quick capture, reminders, lightweight review,
and synchronization. It should not become an endless feed, a social product,
or a second source of distraction. The distinction itself is part of Dunnit's
identity: desktop for seamless work-context capture, mobile for occasional
capture and recall.

## The accumulated value: Hilites and resumes

Encouraging users to mark meaningful entries with Hilite categories creates a
career record with very little extra work. A Hilite can represent an
accomplishment, shipped feature, solved problem, difficult decision, learned
skill, important result, or meaningful collaboration.

After a few months, the ledger contains recent evidence that would otherwise
be forgotten. Dunnit can turn that evidence into:

- resume bullets and new resume sections;
- promotion and performance-review material;
- quarterly and annual accomplishment summaries;
- client progress reports;
- project retrospectives;
- interview preparation;
- a chronological record of skills and impact.

The resume is an output of the work record, not a separate document that must
be rebuilt from memory. Future resume tooling should preserve links back to
the source dates, projects, tags, and ledger lines so that generated claims
remain grounded in evidence.

Potential Hilite workflows include:

- a weekly Hilite review that asks which entries deserve to be retained;
- a monthly “resume delta” showing what has changed since the last review;
- resume views for different roles, such as engineering, management,
  consulting, or research;
- a proof packet containing selected entries, dates, project references, and
  metrics;
- a job-description mode that finds relevant evidence and identifies gaps;
- a career-capital report showing recurring skills, themes, and outcomes.

## Data ownership and FOSS

Dunnit's trust proposition is part of the product, not merely an engineering
implementation detail.

- Ledger data is plain text and human-readable.
- Users can edit it with a normal text editor.
- Git, GitHub, Dropbox, iCloud, Syncthing, or another file workflow can remain
  the sync mechanism.
- Users can feed the data into any AI, script, database, or reporting tool.
- The application is FOSS and can be built for another platform if needed.
- A hosted service must provide complete export and must not be required to
  use the local application.

The goal is not to promise that data can never be lost. Users still need
backups. The promise is that Dunnit will not make the data inaccessible,
unreadable, or dependent on the continued existence of one company.

## Product layers

### 1. Local Dunnit: free and complete

The local application remains fully functional without an account or paid
service. It includes the core popup, ledger files, daily and periodic views,
local reports, Git or shared-directory workflows, and BYO-AI integrations.

This is the adoption and trust layer. There should be no artificial feature
limits on the basic data model or on the user's ability to leave.

### 2. Dunnit Mobile: a paid companion

The mobile app can follow the Anki model: the desktop product is open and
free, while a polished official iOS app helps fund ongoing development. Anki
explicitly describes AnkiMobile purchases as support for development while
keeping the computer version and synchronization ecosystem available
separately. See the [Anki donation FAQ](https://faqs.ankiweb.net/how-can-i-donate.html)
and [official product site](https://apps.ankiweb.net/).

The Dunnit mobile app should focus on:

- one-tap or voice-assisted entry;
- a “record a Hilite” action;
- local reminders;
- viewing and correcting recent entries;
- offline use;
- synchronization through the user's chosen folder or Dunnit Cloud.

A one-time $10–20 purchase is a reasonable starting hypothesis. Mobile sales
can fund desktop maintenance without requiring a subscription from every user.

### 3. Dunnit Cloud: optional hosted convenience

Dunnit Cloud can provide application-specific services without becoming a
general Dropbox replacement:

- multi-device synchronization;
- encrypted backups;
- account recovery;
- mobile connectivity;
- scheduled reports;
- email and webhook delivery;
- hosted analytics;
- managed AI reports;
- optional notifications.

The likely technical shape is a small Go service that stores ledger mutations,
device cursors, and report artifacts. Clients write locally first, queue
operations, push them when online, and pull changes since their last cursor.
Each operation needs a stable client-generated ID so retries cannot duplicate
entries. Raw text export remains available at all times.

The service does not need generic file sharing, media storage, collaborative
documents, or arbitrary folder synchronization. It needs to understand
Dunnit's small text-oriented data model well.

Cloud privacy can offer two modes:

- Standard mode, where the service can process enabled data for reports and
  AI.
- Private mode, where stored ledgers are end-to-end encrypted and only the
  selected context for an explicitly requested report is sent for processing.

Hosted AI should be opt-in, transparent about retention and provider use, and
subject to fair-use limits. BYO-AI must remain available permanently.

## Revenue model

The likely mix is:

| Offering | Working price hypothesis | What the customer pays for |
| --- | ---: | --- |
| Local Dunnit | Free | Complete FOSS application and owned data |
| Founding Supporter | $20–50 once | Supporting development, templates, early builds, and personal support |
| Dunnit Mobile | $10–20 once | Polished official mobile capture and review |
| Dunnit Cloud | $20–30/year | Sync, backup, mobile connectivity, and notifications |
| Dunnit Cloud + AI | $40–60/year | Hosted reports, analytics, and managed AI within fair use |
| Setup and migration | One-time service fee | Data migration, configuration, custom reports, and workflow design |
| Team support | Annual contract | Private deployment, onboarding, support, and custom integrations |

The $20 lifetime idea fits the local application and supporter relationship.
It does not fit unlimited hosting, notifications, or AI because those create
continuing costs. A prepaid multi-year cloud plan can offer a less subscription-
like alternative without promising permanent infrastructure.

## Target users

Dunnit is especially relevant to people who need to remember and explain their
work:

- software engineers and technical workers preparing promotion packets;
- managers preparing reviews and status updates;
- consultants and freelancers documenting client work;
- researchers maintaining project and contribution histories;
- open-source maintainers tracking contributions;
- people who want an external memory for high-context or attention-fragmented
  work;
- privacy-conscious users who want AI assistance without surrendering their
  entire work history to a productivity vendor.

The strongest initial wedge is probably individual knowledge workers who feel
the pain of reconstructing their work for resumes, reviews, or client reports.
Teams and organizations can follow once the personal workflow is proven.

## Paid reports and workflows

The main paid value is not access to an AI model. It is the reliable workflow
around the model:

- selecting the right period and entries;
- filtering by project, person, tag, or category;
- redacting private material;
- grounding claims in source lines;
- producing repeatable formats;
- saving the result in the user's own data tree.

Useful workflows include resume updates, promotion packets, weekly client
reports, meeting preparation, post-meeting capture, quarterly reviews, annual
reviews, and re-entry summaries after time away.

The same workflows can be available locally with BYO-AI, delivered through
Dunnit Cloud, or packaged as open automation recipes for GitHub Actions and
local cron jobs. This preserves user choice while making the hosted option
meaningfully more convenient.

## Business guardrails

- Keep the local app fully usable without payment.
- Keep the ledger format documented and exportable.
- Preserve Git and shared-directory sync.
- Do not make mobile the primary attention surface.
- Do not require Dunnit Cloud for AI; BYO-AI remains a first-class path.
- Avoid unlimited hosted AI until actual usage costs are known.
- Treat user work histories as sensitive, even when they are plain text.
- Build the hosted service around Dunnit semantics rather than recreating
  Dropbox.
- Make the cloud protocol and export story strong enough that users can leave.

## Near-term validation plan

1. Make Hilite capture and Hilite review prominent in the desktop workflow.
2. Prototype a resume-delta report grounded in ledger entries.
3. Create a mobile concept focused only on capture, reminders, and recent-item
   review.
4. Publish a landing page that tests interest in Mobile, Cloud Sync, and
   Hosted AI separately.
5. Offer a small paid cloud alpha before building a broad service.
6. Measure hosted AI cost per active user before setting a permanent price.

The current strategic direction is therefore:

> Free local FOSS at the center; a paid mobile app that funds development; and
> an optional hosted layer that earns recurring revenue through sync, backup,
> notifications, analytics, specialized reports, and managed AI.

That combination lets Dunnit remain durable and user-controlled while giving
people a compelling reason to pay for convenience.
