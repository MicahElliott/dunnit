package dun

import (
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// reportKindDisplayNames maps a ReportFile.Kind to a friendlier label
// for the browse dropdown -- falls back to the raw kind string for
// anything not explicitly listed (keeps this forward-compatible with
// new report kinds without needing this map updated in lockstep).
var reportKindDisplayNames = map[string]string{
	"review-day":     "Review: Day",
	"review-week":    "Review: Week",
	"review-month":   "Review: Month",
	"review-quarter": "Review: Quarter",
	"review-year":    "Review: Year",
	"standup":        "Standup",
	"status":         "Status Report",
	"eod":            "End of Day",
}

func reportKindLabel(kind string) string {
	if label, ok := reportKindDisplayNames[kind]; ok {
		return label
	}
	return kind
}

// showReportsLibraryWindow opens the "Reports Library" -- a browse/
// search window over every generated report file on disk (Review/
// Standup/Status/EOD), the reports-corpus counterpart to
// Navigator's ledger browsing (navigator.go). Two modes in one
// window: a Kind filter (browse all reports of one family,
// chronological by file mtime) and a free-text search across every
// report's body (SearchReports, reportsearch.go) -- selecting a
// result opens the full report in showGeneratedReport (report.go),
// reusing the same read-only viewer (with Copy/Save) every other
// report-producing feature already uses, rather than a new one-off
// viewer here.
//
// See docs/navigator-design.md for the design discussion this closes
// out ("Saved-reports library/browser" and "Cross-report search").
func showReportsLibraryWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: Reports Library")

	all := AllReportFiles()
	kindOptions := []string{"All"}
	seen := map[string]bool{}
	var kinds []string
	for _, r := range all {
		if !seen[r.Kind] {
			seen[r.Kind] = true
			kinds = append(kinds, r.Kind)
		}
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		kindOptions = append(kindOptions, reportKindLabel(k))
	}
	kindSelect := widget.NewSelect(kindOptions, nil)
	kindSelect.SetSelected("All")

	queryEntry := widget.NewEntry()
	queryEntry.SetPlaceHolder("Search report contents (blank = browse by kind only)\u2026")

	results := widget.NewMultiLineEntry()
	results.Wrapping = fyne.TextWrapOff
	results.SetMinRowsVisible(18)

	// resultReports tracks which ReportFile each line of `results`
	// corresponds to (in order), so browseReportAtLine can open the
	// right one -- results is a plain text widget (matching this
	// codebase's search.go/trend.go house style of simple
	// MultiLineEntry result lists), not a clickable list widget, so a
	// separate "open by index" action is used instead of per-row
	// click handlers (see openSelectedBtn below).
	var resultReports []ReportFile
	countLabel := widget.NewLabel("")

	refresh := func() {
		resultReports = nil
		query := strings.TrimSpace(queryEntry.Text)
		var sb strings.Builder

		if query != "" {
			matches := SearchReports(query)
			for _, m := range matches {
				if kindSelect.Selected != "All" && reportKindLabel(m.Report.Kind) != kindSelect.Selected {
					continue
				}
				resultReports = append(resultReports, m.Report)
				sb.WriteString(m.Report.Date.Format("2006-01-02") + "  " +
					reportKindLabel(m.Report.Kind) + "  " + m.Excerpt + "\n")
			}
		} else {
			var filtered []ReportFile
			for _, r := range all {
				if kindSelect.Selected != "All" && reportKindLabel(r.Kind) != kindSelect.Selected {
					continue
				}
				filtered = append(filtered, r)
			}
			sort.Slice(filtered, func(i, j int) bool { return filtered[i].Date.After(filtered[j].Date) })
			for _, r := range filtered {
				resultReports = append(resultReports, r)
				theme := ""
				if r.Theme != "" {
					theme = "  (" + themeDisplayNames[r.Theme] + ")"
				}
				sb.WriteString(r.Date.Format("2006-01-02") + "  " +
					reportKindLabel(r.Kind) + theme + "\n")
			}
		}

		if len(resultReports) == 0 {
			countLabel.SetText("No reports found.")
			results.SetText("")
			return
		}
		countLabel.SetText(pluralCount(len(resultReports), "report", "reports"))
		results.SetText(sb.String())
	}
	kindSelect.OnChanged = func(string) { refresh() }
	queryEntry.OnSubmitted = func(string) { refresh() }
	refresh()

	openBtn := widget.NewButton("Open Selected Line…", func() {
		line := results.CursorRow
		if line < 0 || line >= len(resultReports) {
			return
		}
		r := resultReports[line]
		body, err := ReportBody(r)
		if err != nil {
			body = "(error reading report: " + err.Error() + ")"
		}
		showGeneratedReport(a, "Dunnit: "+reportKindLabel(r.Kind), r.Path, body)
	})

	content := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Kind:"), nil, kindSelect),
		container.NewBorder(nil, nil, nil, widget.NewButton("Search", refresh), queryEntry),
		container.NewBorder(nil, nil, nil, openBtn, countLabel),
		results,
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(720, 560))
	w.Show()
}
