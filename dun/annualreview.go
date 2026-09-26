package dun

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// annualReviewCategories keeps the annual report focused on completed work
// and notable callouts while periodReportInput supplies metrics and pending
// plan items alongside these source lines.
var annualReviewCategories = buildStandupCategories()

// showAnnualReviewDialog lets the user pick a year (default: current)
// and generates an editable narrative summary via the shared LLM CLI
// plumbing, scoped to that year's accomplishments and Hilites.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showAnnualReviewDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Annual Review")

	cfg := LoadConfig()
	currentYear := fiscalYearLabel(time.Now(), cfg)
	yearEntry := widget.NewEntry()
	yearEntry.SetText(strconv.Itoa(currentYear))

	generate := func() {
		year, err := strconv.Atoi(yearEntry.Text)
		if err != nil {
			year = currentYear
		}
		w.Close()
		runAnnualReview(a, year)
	}

	content := container.NewVBox(
		widget.NewLabel("Gather accomplishments and Hilites for fiscal year label:"),
		yearEntry,
		container.NewHBox(
			widget.NewButton("Generate", generate),
			widget.NewButton("Cancel", func() { w.Close() }),
		),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(320, 160))
	w.Show()
}

func runAnnualReview(a fyne.App, year int) {
	showReportPreparation(a, "Annual Review", func() {
		runAnnualReviewReady(a, year)
	})
}

func runAnnualReviewReady(a fyne.App, year int) {
	cfg := LoadConfig()
	anchor := fiscalYearAnchorForLabel(year, cfg, time.Local)
	from, to := periodNominalRangeWithConfig(periodYear, anchor, cfg)
	ledgerText := gatherLedgerTextForRange(from, to, annualReviewCategories)
	if ledgerText == "" {
		w := a.NewWindow("Dunnit: Annual Review")
		w.SetContent(windowPad(widget.NewLabel("No accomplishments or Hilites found for that year.")))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunnit: Generating Annual Review\u2026")
	request := newLLMCLIRequest()
	progress.SetOnClosed(request.close)
	progress.SetContent(windowPad(llmCLIProgressContent(
		"Asking configured LLM CLI to summarize, please wait\u2026\n"+
			"The generated report will be editable before you save it.", request)))
	progress.Show()

	go func() {
		title := periodSummaryTitle(periodYear, anchor)
		summary, err := summarizeWithLLMCLIPromptContext(request.ctx,
			periodSummaryPrompt(periodYear, title),
			periodReportInput(periodYear, anchor, ledgerText))
		request.finish()
		fyne.Do(func() {
			progress.Close()
			if request.canceled() {
				return
			}
			if err != nil {
				w := a.NewWindow("Dunnit: Annual Review " + strconv.Itoa(year))
				w.SetContent(windowPad(widget.NewLabel("Error running configured LLM CLI:\n" + err.Error())))
				w.Resize(fyne.NewSize(600, 500))
				w.Show()
			} else {
				showEditableReportWindow(a, "Dunnit: "+title,
					summaryReportPath(periodYear, anchor), normalizePeriodReport(summary, title))
			}
		})
	}()
}
