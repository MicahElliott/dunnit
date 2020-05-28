# Dunnit Time Recorder

Dunnit is the simplest possible activity recorder.  It’s dumbly
simple, works only on a Mac, and relies on
[Alerter](https://github.com/vjeantet/alerter) to do almost all of the
real work (which isn’t much).

Dunnit is set to pop up a notification prompt every hour to ask you
what you worked on. The update you record in the popup is saved to a
daily log. At the end of a good day you’ll end up with a timestamped
list of things you worked on.

![Dunnit Screenshot](dunnit.png)

## Install and Run

1. Install Alerter: copy the [alerter zip
   file](https://github.com/vjeantet/alerter/releases) to somewhere on
   your PATH (`/usr/local/bin/alerter`)
1. Clone this Dunnit repo and `cd` into it
1. Start the dunnit service: `launchctl load dunnit.plist`
1. Answer the popup prompt every hour (or ignore it; you have 2 minutes)
1. At the end of the week (or day), look over what you did in the
   `$DUNNIT_DIR` log

You can **stop the service** (if you ever feel the need) with:
`launchctl unload dunnit.plist`.

If it doesn’t seem to be working, check for dunnit logs in `/tmp`.

## Customization

The following config is reflects the defaults. It will leave you with
a local directory and sequence of files (with timestamped lines) like:
`~/doc/dunnit/20200530.log`

```sh
# The location of all the daily dunnit log files
export DUNNIT_DIR=~/doc/dunnit
# The timestamp format for each entry
export DUNNIT_TIME_FMT='%…'
# The directory format
export DUNNIT_DATE_FMT='%…'
# Use org-mode date-stamp formatting and file extensions
export DUNNIT_USE_ORG=false
```

## Example Dunnit Log for a Day

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

- maybe package up as a homebrew service
