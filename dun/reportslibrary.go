package dun

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// reportKindDisplayNames maps a ReportFile.Kind to a friendlier label for
// Saved Reports. The labels describe the user's purpose rather than the
// filename prefix used on disk.
var reportKindDisplayNames = map[string]string{
	"review-day":     "Review: Day",
	"review-week":    "Review: Week",
	"review-month":   "Review: Month",
	"review-quarter": "Review: Quarter",
	"review-year":    "Review: Year",
	"standup":        "Quick Standup",
	"status":         "Legacy Status Report",
	"summary":        "Custom Summary",
	"eod":            "End of Day",
}

func reportKindLabel(kind string) string {
	if label, ok := reportKindDisplayNames[kind]; ok {
		return label
	}
	return kind
}

func reportAudienceLabel(r ReportFile) string {
	if r.Audience != "" {
		return r.Audience
	}
	if strings.HasPrefix(r.Kind, "review-") {
		return "Private"
	}
	return ""
}

func reportVariantLabel(r ReportFile) string {
	style := reportStyle(r)
	if audience := reportAudienceLabel(r); audience != "" {
		style += " · " + audience
	}
	return style
}

func reportGroupLabel(group ReportGroup) string {
	if len(group.Reports) == 0 {
		return ""
	}
	representative := group.Reports[0]
	variants := make([]string, 0, len(group.Reports))
	seen := make(map[string]bool)
	for _, report := range group.Reports {
		label := reportVariantLabel(report)
		if !seen[label] {
			seen[label] = true
			variants = append(variants, label)
		}
	}
	suffix := ""
	if len(group.Reports) > 1 {
		suffix = fmt.Sprintf(" · %d variants", len(group.Reports))
	}
	return reportKindLabel(group.Kind) + "  ·  " + reportCoverageLabel(representative) +
		"  ·  " + strings.Join(variants, ", ") + suffix
}

func reportOverlapsRange(report ReportFile, from, to time.Time) bool {
	if from.IsZero() || to.IsZero() {
		return true
	}
	reportFrom, reportTo := report.From, report.To
	if reportFrom.IsZero() || reportTo.IsZero() {
		reportFrom = report.SavedAt
		reportTo = report.SavedAt
	}
	return !reportTo.Before(from) && !reportFrom.After(to)
}

func reportPresetRange(preset string, now time.Time) (from, to time.Time, ok bool) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch preset {
	case "Today":
		return today, today.AddDate(0, 0, 1).Add(-time.Second), true
	case "This week":
		from, to := periodNominalRange(periodWeek, today)
		return from, to, true
	case "This month":
		from, to := periodNominalRange(periodMonth, today)
		return from, to, true
	case "This quarter":
		from, to := periodNominalRange(periodQuarter, today)
		return from, to, true
	case "This fiscal year":
		from, to := periodNominalRange(periodYear, today)
		return from, to, true
	case "Last 30 days":
		return today.AddDate(0, 0, -29), today.AddDate(0, 0, 1).Add(-time.Second), true
	case "Last 90 days":
		return today.AddDate(0, 0, -89), today.AddDate(0, 0, 1).Add(-time.Second), true
	default:
		return time.Time{}, time.Time{}, false
	}
}

func parseReportDate(text string, endOfDay bool) (time.Time, error) {
	value, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(text), time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("use YYYY-MM-DD")
	}
	if endOfDay {
		value = value.AddDate(0, 0, 1).Add(-time.Second)
	}
	return value, nil
}

func filteredReportGroups(all []ReportFile, kind, audience, query string, from, to time.Time) []ReportGroup {
	matchedPaths := map[string]bool{}
	if strings.TrimSpace(query) != "" {
		for _, match := range SearchReports(query) {
			matchedPaths[match.Report.Path] = true
		}
	}
	var filtered []ReportFile
	for _, report := range all {
		if kind != "All" && reportKindLabel(report.Kind) != kind {
			continue
		}
		if audience != "All" && reportAudienceLabel(report) != audience {
			continue
		}
		if !reportOverlapsRange(report, from, to) {
			continue
		}
		if strings.TrimSpace(query) != "" && !matchedPaths[report.Path] {
			continue
		}
		filtered = append(filtered, report)
	}
	return groupReportFiles(filtered)
}

func savedReportTitle(report ReportFile) string {
	return "Dunnit: " + reportKindLabel(report.Kind) + " — " + reportCoverageLabel(report)
}

// showSavedReportPreview is deliberately read-only. Editing is a separate
// explicit action so browsing a saved report cannot look like an editor with
// a mysterious Save button.
func showSavedReportPreview(a fyne.App, report ReportFile) {
	body, err := ReportBody(report)
	if err != nil {
		w := a.NewWindow("Dunnit: Saved Report")
		dialog.ShowError(err, w)
		return
	}
	heading, bodyText := reportHeadingAndBody(body)
	bodyRichText := newReportRichText(bodyText)
	bodyRichText.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(bodyRichText)
	scroll.SetMinSize(fyne.NewSize(0, 300))

	w := a.NewWindow(savedReportTitle(report))
	var headingWidget fyne.CanvasObject
	if heading != "" {
		headingWidget = widget.NewLabelWithStyle(heading, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	copyButtons := reportCopyButtons(a, body)
	editBtn := widget.NewButton("Edit", func() {
		w.Close()
		showEditableReportWindow(a, savedReportTitle(report), report.Path, body)
	})
	closeBtn := widget.NewButton("Close", func() { w.Close() })
	w.SetContent(windowPad(container.NewBorder(
		headingWidget,
		container.NewHBox(copyButtons.Objects[0], copyButtons.Objects[1], editBtn, closeBtn),
		nil, nil, scroll,
	)))
	w.Resize(fyne.NewSize(680, 560))
	w.Show()
}

// showReportsLibraryWindow opens Saved Reports: a period-aware, grouped
// browser over Markdown reports. Multiple styles/audiences remain available
// as variants of one logical covered period.
func showReportsLibraryWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: Saved Reports")
	all := AllReportFiles()

	kindOptions := []string{"All"}
	seenKinds := map[string]bool{}
	for _, report := range all {
		label := reportKindLabel(report.Kind)
		if !seenKinds[label] {
			seenKinds[label] = true
			kindOptions = append(kindOptions, label)
		}
	}
	sort.Strings(kindOptions[1:])
	kindSelect := widget.NewSelect(kindOptions, nil)
	kindSelect.SetSelected("All")

	audienceSelect := widget.NewSelect([]string{"All", "Private", "Shareable"}, nil)
	audienceSelect.SetSelected("All")
	periodSelect := widget.NewSelect([]string{
		"All time", "Today", "This week", "This month", "This quarter", "This fiscal year", "Last 30 days", "Last 90 days", "Custom dates",
	}, nil)
	periodSelect.SetSelected("All time")
	fromEntry := widget.NewEntry()
	fromEntry.SetPlaceHolder("From YYYY-MM-DD")
	toEntry := widget.NewEntry()
	toEntry.SetPlaceHolder("To YYYY-MM-DD")
	fromEntry.Hide()
	toEntry.Hide()
	queryEntry := widget.NewEntry()
	queryEntry.SetPlaceHolder("Search saved report contents…")
	filterStatus := newExplanatoryLabel("")

	var groups []ReportGroup
	selectedGroup := -1
	selectedVariant := 0
	variantSelect := widget.NewSelect(nil, nil)
	variantSelect.Disable()
	previewBtn := widget.NewButton("Preview", nil)
	previewBtn.Disable()
	editBtn := widget.NewButton("Edit", nil)
	editBtn.Disable()

	selectedReport := func() (ReportFile, bool) {
		if selectedGroup < 0 || selectedGroup >= len(groups) {
			return ReportFile{}, false
		}
		variants := groups[selectedGroup].Reports
		if selectedVariant < 0 || selectedVariant >= len(variants) {
			return ReportFile{}, false
		}
		return variants[selectedVariant], true
	}
	updateActions := func() {
		if _, ok := selectedReport(); ok {
			previewBtn.Enable()
			editBtn.Enable()
		} else {
			previewBtn.Disable()
			editBtn.Disable()
		}
	}
	variantSelect.OnChanged = func(selected string) {
		if selectedGroup < 0 || selectedGroup >= len(groups) {
			return
		}
		for i, report := range groups[selectedGroup].Reports {
			if reportVariantLabel(report) == selected {
				selectedVariant = i
				break
			}
		}
		updateActions()
	}

	reportList := widget.NewList(
		func() int { return len(groups) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapOff
			label.Truncation = fyne.TextTruncateEllipsis
			return label
		},
		func(id widget.ListItemID, object fyne.CanvasObject) {
			object.(*widget.Label).SetText(reportGroupLabel(groups[id]))
		},
	)
	reportList.OnSelected = func(id widget.ListItemID) {
		selectedGroup = id
		selectedVariant = 0
		if id < 0 || id >= len(groups) {
			variantSelect.SetOptions(nil)
			variantSelect.Disable()
			updateActions()
			return
		}
		options := make([]string, len(groups[id].Reports))
		for i, report := range groups[id].Reports {
			options[i] = reportVariantLabel(report)
		}
		variantSelect.SetOptions(options)
		variantSelect.SetSelected(options[0])
		variantSelect.Enable()
		updateActions()
	}

	refresh := func() {
		var from, to time.Time
		var err error
		if periodSelect.Selected == "Custom dates" {
			from, err = parseReportDate(fromEntry.Text, false)
			if err == nil {
				to, err = parseReportDate(toEntry.Text, true)
			}
			if err == nil && from.After(to) {
				err = fmt.Errorf("the start date must be before the end date")
			}
		} else if periodSelect.Selected != "All time" {
			from, to, _ = reportPresetRange(periodSelect.Selected, time.Now())
		}
		if err != nil {
			filterStatus.SetText(err.Error())
			groups = nil
		} else {
			groups = filteredReportGroups(all, kindSelect.Selected, audienceSelect.Selected, queryEntry.Text, from, to)
		}
		selectedGroup = -1
		selectedVariant = 0
		variantSelect.SetOptions(nil)
		variantSelect.Disable()
		updateActions()
		reportList.Refresh()
		if err == nil {
			filterStatus.SetText(fmt.Sprintf("%d logical report periods (%d files)", len(groups), countGroupedReports(groups)))
		}
	}

	periodSelect.OnChanged = func(selected string) {
		if selected == "Custom dates" {
			fromEntry.Show()
			toEntry.Show()
		} else {
			fromEntry.Hide()
			toEntry.Hide()
		}
		refresh()
	}
	kindSelect.OnChanged = func(string) { refresh() }
	audienceSelect.OnChanged = func(string) { refresh() }
	queryEntry.OnChanged = func(string) { refresh() }
	fromEntry.OnChanged = func(string) {
		if periodSelect.Selected == "Custom dates" {
			refresh()
		}
	}
	toEntry.OnChanged = func(string) {
		if periodSelect.Selected == "Custom dates" {
			refresh()
		}
	}
	previewBtn.OnTapped = func() {
		if report, ok := selectedReport(); ok {
			showSavedReportPreview(a, report)
		}
	}
	editBtn.OnTapped = func() {
		if report, ok := selectedReport(); ok {
			body, err := ReportBody(report)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			showEditableReportWindow(a, savedReportTitle(report), report.Path, body)
		}
	}

	refresh()
	filterRow := container.NewVBox(
		container.NewGridWithColumns(3,
			container.NewVBox(widget.NewLabel("Type"), kindSelect),
			container.NewVBox(widget.NewLabel("Audience"), audienceSelect),
			container.NewVBox(widget.NewLabel("Period"), periodSelect),
		),
		container.NewGridWithColumns(2, fromEntry, toEntry),
		queryEntry,
		filterStatus,
	)
	variantRow := container.NewBorder(widget.NewLabel("Version:"), nil, nil, nil, variantSelect)
	actionRow := container.NewHBox(previewBtn, editBtn, widget.NewButton("Close", func() { w.Close() }))

	listAndVariant := container.NewBorder(variantRow, nil, nil, nil, reportList)
	w.SetContent(windowPad(container.NewBorder(
		filterRow,
		actionRow,
		nil,
		nil,
		listAndVariant,
	)))
	w.Resize(fyne.NewSize(820, 620))
	w.Show()
}

func countGroupedReports(groups []ReportGroup) int {
	total := 0
	for _, group := range groups {
		total += len(group.Reports)
	}
	return total
}
