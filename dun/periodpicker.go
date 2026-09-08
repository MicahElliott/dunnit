package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// periodPickerBackOffsets is how many periods back the picker's
// short list offers, beyond "this period (so far)" and "last period"
// -- e.g. offsets -2..-5 for Month gives 4 more months before that.
// Kept small (docs/kickoff-review-design.md's "report navigator" is
// explicitly deferred as a separate, bigger follow-up task -- this
// picker is just enough to unblock "I opened Review and it showed me
// the wrong month" without building a full report browser).
const periodPickerBackOffsets = 4

// showPeriodPicker shows a small preamble dialog letting the user
// choose which occurrence of period to open a Review for: "This
// <unit> (so far)" (offset 0, only meaningfully different from "last"
// once the period is actually in progress), "Last <unit>" (offset -1,
// the prior default/only behavior), and a short back-list (offsets -2
// through -(1+periodPickerBackOffsets)). onChosen is called with the
// resulting anchor time.Time once the user picks one; the dialog then
// closes itself. cfg is used for periodLabel's ExtendWorkWeekTo7Days
// lookup.
//
// This is deliberately a minimal "which of the last few periods"
// picker, not the fuller cross-unit/cross-theme report browser floated
// alongside this fix -- that's tracked as a separate future task.
func showPeriodPicker(a fyne.App, cfg Config, period summaryPeriod, onChosen func(anchor time.Time)) {
	now := time.Now()
	w := a.NewWindow("Dunnit: Which " + string(period) + "?")

	list := container.NewVBox()
	addOption := func(offset int) {
		anchor := periodOffsetAnchor(period, now, offset)
		label := periodLabel(cfg, period, anchor) + periodProgressSuffix(period, anchor)
		if paths, themes := listReviewReportsForPeriod(period, anchor); len(paths) > 0 {
			label += "  [" + reportExistsSummary(themes) + "]"
		}
		list.Add(widget.NewButton(label, func() {
			w.Close()
			onChosen(anchor)
		}))
	}
	addOption(0)
	for offset := -1; offset >= -(1 + periodPickerBackOffsets); offset-- {
		addOption(offset)
	}

	w.SetContent(windowPad(container.NewVBox(
		widget.NewLabel("Which "+string(period)+" would you like to work with?"),
		list,
	)))
	w.Resize(fyne.NewSize(360, 320))
	w.Show()
}

// reportExistsSummary renders a short "already have: Theme1, Theme2"
// style fragment for the period picker's per-option label, given the
// themes of already-saved reports for that period (as returned by
// listReviewReportsForPeriod) -- "" themes (pre-existing reports saved
// before theme was added to the filename) show as "untitled".
func reportExistsSummary(themes []string) string {
	out := "saved: "
	for i, th := range themes {
		if i > 0 {
			out += ", "
		}
		if display, ok := themeDisplayNames[th]; ok {
			out += display
		} else {
			out += "untitled"
		}
	}
	return out
}
