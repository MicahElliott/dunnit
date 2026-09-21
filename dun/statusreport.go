package dun

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// shareUnsafeCategories are excluded from the "shareable" status
// report variant -- personal-only categories not appropriate for a
// boss/colleague audience.
var shareUnsafeCategories = map[string]bool{
	"SENTIMENT": true, "PRODUCTIVITY": true, "WASTED": true, "FAIL": true,
}

func statusReportPath(anchor, generated time.Time) string {
	return weeklyReportPathForKind("status", anchor, generated)
}

const privateStatusPrompt = "Summarize the following ledger entries into a " +
	"status report covering the selected week. Be thorough and candid " +
	"— this is a private report for the author’s own use, so include " +
	"struggles/blockers/personal reflections as well as accomplishments." +
	reportMentionPromptGuidanceText

const shareableStatusPrompt = "Summarize the following ledger entries into a " +
	"status report covering the selected week, suitable to share with a " +
	"manager or colleagues. Focus on accomplishments, progress, and " +
	"upcoming plans; keep a professional, concise tone." +
	reportMentionPromptGuidanceText

// showStatusReportDialog lets the user pick a week and an
// audience (Private/Shareable), then generates the report via the
// shared Summarize/configured LLM CLI plumbing (FR-23).
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showStatusReportDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Status Report")

	today := time.Now()
	var weekAnchors []time.Time
	var weekOptions []string
	for offset := 0; offset >= -periodPickerBackOffsets; offset-- {
		anchor := periodOffsetAnchor(periodWeek, today, offset)
		_, week := anchor.ISOWeek()
		weekAnchors = append(weekAnchors, anchor)
		weekOptions = append(weekOptions, periodLabel(LoadConfig(), periodWeek, anchor)+" (W"+strconv.Itoa(week)+")")
	}
	weekSelect := widget.NewSelect(weekOptions, nil)
	weekSelect.SetSelected(weekOptions[0])

	audienceSelect := widget.NewSelect([]string{"Private", "Shareable"}, nil)
	audienceSelect.SetSelected("Private")

	generate := func() {
		selected := 0
		for i, option := range weekOptions {
			if option == weekSelect.Selected {
				selected = i
				break
			}
		}
		w.Close()
		runStatusReport(a, weekAnchors[selected], audienceSelect.Selected)
	}

	content := container.NewVBox(
		widget.NewLabel("Week:"),
		weekSelect,
		widget.NewLabel("Audience:"),
		audienceSelect,
		container.NewHBox(
			widget.NewButton("Generate", generate),
			widget.NewButton("Cancel", func() { w.Close() }),
		),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(360, 320))
	w.Show()
}

func runStatusReport(a fyne.App, anchor time.Time, audience string) {
	from, to := periodNominalRange(periodWeek, anchor)
	if to.After(time.Now()) {
		to = time.Now()
	}

	var categories map[string]bool // nil = all categories (Private)
	prompt := privateStatusPrompt
	if audience == "Shareable" {
		prompt = shareableStatusPrompt
		// Shareable still wants "all categories except the
		// share-unsafe ones" -- gatherLedgerTextForRange treats a nil/
		// empty categories set as "match everything", so build the
		// full allowed set explicitly by excluding shareUnsafeCategories.
		categories = map[string]bool{}
		for _, c := range Categories {
			if !shareUnsafeCategories[c.Code] {
				categories[c.Code] = true
			}
		}
	}

	ledgerText := gatherLedgerTextForRange(from, to, categories)
	if mentions := reportMentionContextForRange(from, to, categories); mentions != "" {
		ledgerText += "\n\n" + mentions
	}
	if ledgerText == "" {
		w := a.NewWindow("Dunnit: Status Report")
		w.SetContent(windowPad(widget.NewLabel("No matching ledger entries found for that range.")))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunnit: Generating Status Report\u2026")
	request := newLLMCLIRequest()
	progress.SetOnClosed(request.close)
	progress.SetContent(windowPad(llmCLIProgressContent(
		"Asking configured LLM CLI to summarize, please wait\u2026\n"+
			"The generated report will be copied to your clipboard automatically.", request)))
	progress.Show()

	go func() {
		summary, err := summarizeWithLLMCLIPromptContext(request.ctx, prompt, ledgerText)
		request.finish()
		fyne.Do(func() {
			progress.Close()
			if request.canceled() {
				return
			}
			if err != nil {
				w := a.NewWindow("Dunnit: " + audience + " Status Report")
				w.SetContent(windowPad(widget.NewLabel("Error running configured LLM CLI:\n" + err.Error())))
				w.Resize(fyne.NewSize(600, 500))
				w.Show()
			} else {
				showGeneratedReport(a, "Dunnit: "+audience+" Status Report",
					statusReportPath(anchor, time.Now()), summary)
			}
		})
	}()
}
