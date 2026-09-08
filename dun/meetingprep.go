package dun

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// normalizeTag ensures s starts with "#" (adding it if the user typed
// the tag name without it), after trimming surrounding whitespace.
func normalizeTag(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "#") {
		return s
	}
	return "#" + s
}

// taggedEntry is one ledger line matched by pullTaggedEntries, along
// with the date it came from (parsed from its source file's name) so
// results across multiple files can be sorted chronologically.
type taggedEntry struct {
	date time.Time
	line string
}

// relatedCategories are the categories considered "Related" to a
// meeting/tag in Meeting Prep's history filter -- broader than just
// MEETING itself, but not everything.
var relatedCategories = map[string]bool{
	"MEETING": true, "IDEA": true, "QUESTION": true, "WIN": true,
	"RISK": true, "GOAL": true, "IMPACT": true, "CAREER": true,
}

// categoryFilterSet returns the set of category codes matching a
// Meeting Prep filter choice: "MEETING" (just that one), "Related"
// (relatedCategories, which includes MEETING), or "All" (nil,
// meaning no filtering -- pullTaggedEntries treats a nil/empty set as
// "match any category").
func categoryFilterSet(choice string) map[string]bool {
	switch choice {
	case "MEETING":
		return map[string]bool{"MEETING": true}
	case "Related":
		return relatedCategories
	default: // "All"
		return nil
	}
}

// pullTaggedEntries scans all ledger files dated on/after since,
// returning every line containing the given tag (as a whole #tag
// token, matched via extractTags) in chronological order (oldest
// first). If categories is non-nil, only lines whose category is in
// that set are included; nil/empty means match any category. Used by
// Meeting Prep's history pull -- lets the user see what's accumulated
// under a tag like "#boss" over recent weeks without altering the
// source files.
func pullTaggedEntries(tag string, since time.Time, categories map[string]bool) []taggedEntry {
	var out []taggedEntry
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil || date.Before(since) {
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			cat, _, ok := parseLedgerLine(line)
			if ok && len(categories) > 0 && !categories[cat] {
				continue
			}
			for _, t := range extractTags(line) {
				if strings.EqualFold(t, tag) {
					out = append(out, taggedEntry{date: *date, line: line})
					break
				}
			}
		}
		f.Close()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].date.Before(out[j].date) })
	return out
}

// lastN returns the last n elements of entries (or all of them if
// there are fewer than n).
func lastN(entries []taggedEntry, n int) []taggedEntry {
	if len(entries) <= n {
		return entries
	}
	return entries[len(entries)-n:]
}

// showMeetingPrepDialog shows recent history for a tag (e.g. "#boss")
// -- the last ~8 ledger entries containing that tag, within a
// user-chosen lookback window -- alongside a fresh note field that
// still logs a new MEETING entry under the tag on Save (FR-11). The
// history box is editable for the user's own scratch/annotation use,
// but editing it never writes back to the original ledger files
// (append-only design preserved) -- only the note field's Save action
// writes anything. FR-12 folded in here rather than as a separate
// screen: the MEETING category filter already covers "agenda view by
// tag", and "Only new since last pull" (backed by lastpulled.go)
// covers the "last pulled" marker behavior.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showMeetingPrepDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Meeting Prep")

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #jeff, #boss)")

	weeksSelect := widget.NewSelect([]string{"1", "2", "3", "4", "12"}, nil)
	weeksSelect.SetSelected("2")

	catFilterSelect := widget.NewSelect([]string{"MEETING", "Related", "All"}, nil)
	catFilterSelect.SetSelected("MEETING")

	// onlyNewCheck (FR-12): when checked, entries are additionally
	// filtered to those newer than the tag's last-pulled marker (see
	// lastpulled.go), so a repeat pull only shows what's accumulated
	// since last time. Unchecked (default) shows the full lookback
	// window regardless of prior pulls -- "all time" within the
	// window, per FR-12's "unless the user asks for all time".
	onlyNewCheck := widget.NewCheck("Only new since last pull", nil)

	history := widget.NewMultiLineEntry()
	history.SetPlaceHolder("Enter a tag above and click Refresh to pull recent entries\u2026")
	history.SetMinRowsVisible(8)

	refreshHistory := func() {
		tag := normalizeTag(tagEntry.Text)
		if tag == "" {
			history.SetText("")
			return
		}
		weeks, err := strconv.Atoi(weeksSelect.Selected)
		if err != nil || weeks <= 0 {
			weeks = 2
		}
		since := time.Now().AddDate(0, 0, -7*weeks)
		if onlyNewCheck.Checked {
			if lastPulled, ok := loadLastPulled()[tag]; ok && lastPulled.After(since) {
				since = lastPulled
			}
		}
		categories := categoryFilterSet(catFilterSelect.Selected)
		entries := lastN(pullTaggedEntries(tag, since, categories), 8)
		markPulled(tag)
		if len(entries) == 0 {
			history.SetText("(no entries found for " + tag + " in the last " + weeksSelect.Selected + " week(s))")
			return
		}
		var lines []string
		for _, e := range entries {
			lines = append(lines, e.line)
		}
		history.SetText(strings.Join(lines, "\n"))
	}
	refreshBtn := widget.NewButton("Refresh", refreshHistory)
	tagEntry.OnSubmitted = func(string) { refreshHistory() }
	weeksSelect.OnChanged = func(string) { refreshHistory() }
	catFilterSelect.OnChanged = func(string) { refreshHistory() }
	onlyNewCheck.OnChanged = func(bool) { refreshHistory() }

	noteEntry := widget.NewMultiLineEntry()
	noteEntry.SetPlaceHolder("New agenda note for this meeting\u2026")
	noteEntry.SetMinRowsVisible(3)

	saveNote := func() {
		tag := normalizeTag(tagEntry.Text)
		note := strings.TrimSpace(noteEntry.Text)
		if tag == "" || note == "" {
			return
		}
		recordActivity(tag+" "+note, "MEETING")
		noteEntry.SetText("")
	}

	content := container.NewVBox(
		widget.NewLabel("Meeting Prep"),
		container.NewBorder(nil, nil, nil, container.NewHBox(catFilterSelect, weeksSelect, refreshBtn), tagEntry),
		onlyNewCheck,
		widget.NewLabel("Recent entries for this tag (editable scratch view \u2014 does not alter the ledger):"),
		history,
		widget.NewLabel("Add a new note:"),
		noteEntry,
		container.NewHBox(
			widget.NewButton("Save Note", saveNote),
			widget.NewButton("Close", func() { w.Close() }),
		),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(480, 520))
	w.Show()
}
