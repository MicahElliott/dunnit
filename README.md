# Dunnit Time Recorder

Dunnit is a KISS process for recording your daily goals and activity
and making that easy to find and use for weekly, monthly, and any sort
of reporting — think resumes, daily standups, quarterly reports,
annual reviews, etc. It’s dumbly simple: you just answer the popup
every hour with a short one-line response. It is not a TODO app,
because TODOs are aspirational; Dunnits are factual and enable you to
capture impact.

Dunnit is set to pop up a notification prompt every hour to ask you
what you worked on. The update you record in the popup is saved to a
daily log (the ledger). At the end of the day, you’ll end up with a
timestamped list of things you worked on. At that point, Dunnit helps
you convert your day into an “impact report”.

Analysis is CLI-driven for now. You can roll up your Dunnits at the
end of a week or a month. Generate status reports and aggregate by tag
categories.

![Dunnit Poput](dunnit.png)
![Dunnit Menu](menu.png)

## Caveat and request for feedback

Dunnit is a proof of concept, not quite a real app. The UI is just a
handful of scripts cobbled together into something that kinda works.
The real version is getting started, and will run on at least Linux
and Windows desktops, but the minimal functionality you’re seeing here
is enough for you to get comfortable with the Dunnit workflow. Please
try it out for a couple weeks, collect your thoughts, and send me
anything (good, bad, ideas, whatever) you’re willing to share about
your experience with it. Any feedback is so valuable to me, and I will
buy or make you a very fancy drink of your choosing next time we meet.

## Install and Run

Do most of these steps from a terminal. I would like to see you succeed
with Dunnit, so please reach out to me for help with any of this.

1. Open a terminal: Open up spotlight (Cmd-Space) and type `terminal`

1. Install [Homebrew](https://brew.sh/) if you haven’t already. This
   and the next step will take several minutes.

   ```sh
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

1. Install git

   ```sh
   brew install git
   ```

1. Clone this Dunnit repo to `~/dunnit` (this step is not flexible!)

   ```sh
   cd # to your $HOME
   git clone https://github.com/MicahElliott/dunnit.git
   cd dunnit
   ```

1. Sign up for a [Github](https://github.com/join) (or Gitlab) account.

1. Set up a remote git repo to track your Dunnits: One common approach
   to this is
   [creating a private Github repo](https://docs.github.com/en/github/getting-started-with-github/create-a-repo).
   Follow those instructions, and:

   - name your new repo “mydunnits”
   - select the radio button “Private”
   - check the box “Add a README file”

1. Clone your Dunnit daily activity repo for local tracking.

   ```sh
   cd ~/dunnit
   git clone git@github.com:YOURUSERNAME/mydunnits.git log
   ```

1. Finish the Dunnit setup by installing the remaining dependencies.

   ```sh
   cd ~/dunnit
   ./init.zsh
   ```

1. Run Dunnit.

   ```sh
   cd ~/dunnit
   ./dunnit-menu
   ```

1. Feel free to close the terminal now.

## Usage

See [help.txt](help.txt) for usage (this help is also built into Dunnit).

## Special Sections

- til
- win
- career
- mtg
- idea

## Stop and Remove

You can **stop the service** (if you ever feel the need) with:
`launchctl unload dunnit.plist`.

## Extended Usage

### Hashtagging

It’s a good idea to adopt some tagging conventions for categorizing
your Dunnits. Suggestions: ticket numbers, `#star`, `#rollup`

Search for hashtags across all log files:

```sh
% g '#mywork'
log/2020/w22-May/20200529-Thu.log
[1518] Added snooze feature to #mywork

log/2020/w22-May/20200528-Wed.log
[1739] Added proper LAST message to #mywork

log/2020/w23-Jun/20200601-Mon.log
[1605] Refactored and added some separated commands to #mywork
```

## Customization (coming soon)

The following config reflects the defaults. It will leave you with
a local directory and sequence of files (with timestamped lines) like:
`~/dunnit/log/2020/w23-Jun/20200530-Mon.log`

```sh
# The location of all the daily dunnit log files
export DUNNIT_DIR=~/doc/dunnit

# THESE ARE NOT YET IMPLEMENTED…
# The timestamp format for each entry
export DUNNIT_TIME_FMT='%…'
# The directory format
export DUNNIT_DATE_FMT='%…'
# Use org-mode date-stamp formatting and file extensions
export DUNNIT_USE_ORG=false
```

## Example Dunnit Ledger Log for a Day

```log
[0800] Walked the dog, listened to a podcast; not feeling very productive
[0900] Started the work day with a boring meeting
[1000] Refactored the time traveller #DUMB-42
[1100] Addressed a few linter errors #DUMB-42
[1400] Drew the diagram
[1500] Reviewed John’s PR; tried to make my Emacs windowing faster
[1700] Figured out why the data isn’t showing up in the bucket #DUMB-97
```

## TODO

- add a couple utils to analyze/summarize the day, week, etc
- throw away launchd and just go with daemonize
- flexible scheduler to run on the hour instead of when started
- generate a weekly or monthly status report with the key tagged
- histogram/word cloud of used hashtags
- ditto“ button to indicate working on same thing as last time
- more tagging conventions?
- clickable menubar icon for recording a dunnit at any time
- configurable/invocable day start and end prompts
- better menuing with checkboxes for planned/completed items
- org-todo as basis for structure
- store all todos (not dunnits) in a single file
- support archive feature
