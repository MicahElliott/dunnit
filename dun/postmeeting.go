package dun

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// postMeetingCategories are the quick-entry fields offered in the
// post-meeting capture session (FR-36) -- early shape per Micah:
// TIL/GOAL/RISK, expected to grow/change with real usage. TODO is
// deliberately handled separately (see todoEntry in
// showPostMeetingCapture below) since it's the one category most
// likely to need several entries per meeting (action items), so it
// gets its own larger box at the bottom of the dialog rather than
// sitting inline with the others.
var postMeetingCategories = []string{"TIL", "GOAL", "RISK"}

// showPostMeetingCapture starts a quick multi-category capture
// session (FR-36), tag-scoped to the same meeting tag used for prep
// (e.g. "#boss") so entries logically group with that meeting's
// MEETING-tagged prep notes. Now opens alongside Meeting Prep right
// at meeting *start* (sched.go, 2026-09-03) rather than 15-45 min
// after, so it can be filled in live during the meeting instead of
// relying on after-the-fact recall.
//
// Every field is a multi-line entry (widget.NewMultiLineEntry) --
// one item per line, blank lines skipped -- since a meeting commonly
// produces more than one item for a given category (e.g. several
// action items), not just one. Each non-blank line writes its own
// correctly-categorized ledger line, tagged with the given tag;
// entirely-empty fields write nothing. tag may be "" (e.g. invoked
// from the tray rather than a specific recurring meeting) -- the
// user can still type one in.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showPostMeetingCapture(a fyne.App, tag string) {
	w := a.NewWindow("Dunnit: Post-Meeting Capture")

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("#tag (e.g. #boss) \u2014 entries are grouped under this tag")
	tagEntry.SetText(tag)

	fields := make(map[string]*widget.Entry, len(postMeetingCategories))
	formItems := make([]*widget.FormItem, 0, len(postMeetingCategories)+1)
	formItems = append(formItems, widget.NewFormItem("Tag", tagEntry))
	for _, cat := range postMeetingCategories {
		e := widget.NewMultiLineEntry()
		e.SetPlaceHolder("(skip if nothing to add \u2014 one item per line)")
		e.SetMinRowsVisible(2)
		fields[cat] = e
		formItems = append(formItems, widget.NewFormItem(cat, e))
	}

	// todoEntry is TODO's own bigger box at the bottom of the dialog
	// (not inline with the other fields' form rows) -- action items
	// are the category most likely to need several entries per
	// meeting, so it gets more visible room to grow into.
	todoEntry := widget.NewMultiLineEntry()
	todoEntry.SetPlaceHolder("Any TODOs/action items? One per line, skip if none\u2026")
	todoEntry.SetMinRowsVisible(5)

	writeLines := func(field *widget.Entry, meetingTag, cat string) {
		for _, line := range strings.Split(field.Text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if meetingTag != "" {
				line = meetingTag + " " + line
			}
			recordActivity(line, cat)
		}
	}

	save := func() {
		meetingTag := normalizeTag(tagEntry.Text)
		for _, cat := range postMeetingCategories {
			writeLines(fields[cat], meetingTag, cat)
		}
		writeLines(todoEntry, meetingTag, "TODO")
		w.Close()
	}

	content := container.NewVBox(
		newWindowHeading("📝 Post-Meeting Capture"),
		newExplanatoryLabel("Quick multi-category dump; skip any field. Use one item per line."),
		widget.NewForm(formItems...),
		widget.NewLabelWithStyle("TODO", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		todoEntry,
		container.NewHBox(
			widget.NewButton("Save", save),
			widget.NewButton("Cancel", func() { w.Close() }),
		),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(480, 520))
	w.Show()
}
