package dun

import (
	"errors"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// shareUnsafeCategories are excluded from the "shareable" status
// report variant -- personal-only categories not appropriate for a
// boss/colleague audience.
var shareUnsafeCategories = map[string]bool{
	"SENTIMENT": true, "PRODUCTIVITY": true, "WASTED": true, "FAIL": true,
}

const privateStatusPrompt = "Summarize the following ledger entries into a " +
	"status report covering the given date range. Be thorough and candid " +
	"-- this is a private report for the author's own use, so include " +
	"struggles/blockers/personal reflections as well as accomplishments."

const shareableStatusPrompt = "Summarize the following ledger entries into a " +
	"status report covering the given date range, suitable to share with a " +
	"manager or colleagues. Focus on accomplishments, progress, and " +
	"upcoming plans; keep a professional, concise tone."

// showStatusReportDialog lets the user pick a date range and an
// audience (Private/Shareable), then generates the report via the
// shared Summarize/gh copilot plumbing (FR-23).
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showStatusReportDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Status Report")

	today := time.Now()
	fromEntry := widget.NewEntry()
	fromEntry.SetText(today.AddDate(0, 0, -7).Format("2006-01-02"))
	fromEntry.SetPlaceHolder("YYYY-MM-DD")

	toEntry := widget.NewEntry()
	toEntry.SetText(today.Format("2006-01-02"))
	toEntry.SetPlaceHolder("YYYY-MM-DD")

	audienceSelect := widget.NewSelect([]string{"Private", "Shareable"}, nil)
	audienceSelect.SetSelected("Private")

	generate := func() {
		from, err1 := time.ParseInLocation("2006-01-02", strings.TrimSpace(fromEntry.Text), time.Local)
		to, err2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(toEntry.Text), time.Local)
		if err1 != nil || err2 != nil {
			dialog.ShowError(errors.New("From/To must be valid YYYY-MM-DD dates"), w)
			return
		}
		w.Close()
		runStatusReport(a, from, to, audienceSelect.Selected)
	}

	content := container.NewVBox(
		widget.NewLabel("From (YYYY-MM-DD):"),
		fromEntry,
		widget.NewLabel("To (YYYY-MM-DD):"),
		toEntry,
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

func runStatusReport(a fyne.App, from, to time.Time, audience string) {
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())

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
	if ledgerText == "" {
		w := a.NewWindow("Dunnit: Status Report")
		w.SetContent(windowPad(widget.NewLabel("No matching ledger entries found for that range.")))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunnit: Generating Status Report\u2026")
	progress.SetContent(windowPad(widget.NewLabel(
		"Asking gh copilot to summarize, please wait\u2026\n" +
			"The generated report will be copied to your clipboard automatically.")))
	progress.Show()

	go func() {
		summary, err := summarizeWithCopilotPrompt(prompt, ledgerText)
		fyne.Do(func() {
			progress.Close()
			w := a.NewWindow("Dunnit: " + audience + " Status Report")
			if err != nil {
				w.SetContent(windowPad(widget.NewLabel("Error running gh copilot:\n" + err.Error())))
			} else {
				a.Clipboard().SetContent(summary)
				body := widget.NewMultiLineEntry()
				body.SetText(summary)
				body.Wrapping = fyne.TextWrapWord
				w.SetContent(windowPad(container.NewVScroll(body)))
			}
			w.Resize(fyne.NewSize(600, 500))
			w.Show()
		})
	}()
}
