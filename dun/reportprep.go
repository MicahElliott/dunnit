package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// The Daybook owns the temporary preparation banner, while report windows own
// the continuation callback. Keeping the two ends separate lets every report
// workflow share the same short tidy-up handoff without making Daybook know
// anything about reports.
var (
	setDaybookReportPrepNotice   func(subject string, finish func())
	clearDaybookReportPrepNotice func()
)

// showReportPreparation gives the user a deliberate chance to tidy the
// ledger before an AI report reads it. The preparation window remains open
// while Daybook is visible, so closing or hiding Daybook cannot strand the
// report workflow. Generation begins only after the user chooses one of the
// two explicit continuation actions.
func showReportPreparation(a fyne.App, subject string, onReady func()) {
	w := a.NewWindow("Dunnit: Prepare " + subject)

	finished := false
	finish := func() {
		if finished {
			return
		}
		finished = true
		if clearDaybookReportPrepNotice != nil {
			clearDaybookReportPrepNotice()
		}
		if trayWindow != nil {
			hideDaybook(trayWindow)
		}
		w.Close()
		onReady()
	}

	openDaybook := widget.NewButton("Open Daybook", func() {
		if trayWindow == nil {
			return
		}
		if setDaybookReportPrepNotice != nil {
			setDaybookReportPrepNotice(subject, finish)
		}
		ShowDaybook(trayWindow, false)
	})
	continueBtn := widget.NewButton("Continue without tidying", finish)
	cancelBtn := widget.NewButton("Cancel", func() { w.Close() })

	w.SetOnClosed(func() {
		if !finished && clearDaybookReportPrepNotice != nil {
			clearDaybookReportPrepNotice()
		}
	})
	w.SetContent(windowPad(container.NewVBox(
		newWindowHeading("🧹 Quick tidy-up before "+subject),
		newExplanatoryLabel("Add missing work, correct categories, record useful Hilites, and apply any exclusion tags before the report reads the ledger. This takes you to Daybook; return here when you are finished."),
		container.NewHBox(openDaybook, continueBtn, cancelBtn),
	)))
	w.Resize(fyne.NewSize(520, 220))
	w.Show()
}
