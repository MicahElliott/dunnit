package dun

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// trendPoint is one day's latest valid PRODUCTIVITY (1-5) and/or SENTIMENT
// reading. Counts retain the fact that more than one reading was logged that
// day instead of silently hiding it.
type trendPoint struct {
	date              time.Time
	productivity      int
	productivitySet   bool
	productivityCount int
	sentiment         int // -1/0/1, only meaningful if sentimentSet
	sentimentSet      bool
	sentimentCount    int
	productivityTime  time.Time
	productivitySrc   string
	productivityLine  int
	sentimentTime     time.Time
	sentimentSrc      string
	sentimentLine     int
}

// sentimentScore maps the SENTIMENT category's free-text values to a -1..1
// numeric score. Case-insensitive parsing keeps hand-edited ledger values
// from disappearing from the metrics view.
func sentimentScore(s string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "negative":
		return -1, true
	case "neutral":
		return 0, true
	case "positive":
		return 1, true
	}
	return 0, false
}

func trendEntryIsLater(entry LedgerEntry, currentTime time.Time, currentSource string, currentLine int) bool {
	if !entry.Time.IsZero() && currentTime.IsZero() {
		return true
	}
	if !entry.Time.IsZero() && !currentTime.IsZero() && entry.Time.After(currentTime) {
		return true
	}
	if entry.Time.Equal(currentTime) {
		if entry.Source == currentSource {
			return entry.Line > currentLine
		}
		return entry.Source > currentSource
	}
	return false
}

func gatherTrendPoints(days int) []trendPoint {
	if days <= 0 {
		return nil
	}
	today := time.Now()
	to := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 0, today.Location())
	from := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()).AddDate(0, 0, -(days - 1))
	return gatherTrendPointsForRange(from, to)
}

// gatherTrendPointsForRange reads the latest valid daily signal in the
// inclusive date range. Exclusion tags use the same configured report policy
// as other report and metrics inputs.
func gatherTrendPointsForRange(from, to time.Time) []trendPoint {
	cfg := LoadConfig()
	byDate := map[string]*trendPoint{}
	for _, entry := range AllLedgerEntries() {
		if entry.Date.Before(from) || entry.Date.After(to) || !eodEntryIncluded(entry, cfg) {
			continue
		}
		key := entry.Date.Format("20060102")
		point := byDate[key]
		if point == nil {
			point = &trendPoint{date: entry.Date}
			byDate[key] = point
		}
		switch entry.Category {
		case "PRODUCTIVITY":
			value, err := strconv.Atoi(strings.TrimSpace(entry.Text))
			if err != nil || value < 1 || value > 5 {
				continue
			}
			point.productivityCount++
			if !point.productivitySet || trendEntryIsLater(entry, point.productivityTime, point.productivitySrc, point.productivityLine) {
				point.productivity = value
				point.productivitySet = true
				point.productivityTime = entry.Time
				point.productivitySrc = entry.Source
				point.productivityLine = entry.Line
			}
		case "SENTIMENT":
			score, ok := sentimentScore(entry.Text)
			if !ok {
				continue
			}
			point.sentimentCount++
			if !point.sentimentSet || trendEntryIsLater(entry, point.sentimentTime, point.sentimentSrc, point.sentimentLine) {
				point.sentiment = score
				point.sentimentSet = true
				point.sentimentTime = entry.Time
				point.sentimentSrc = entry.Source
				point.sentimentLine = entry.Line
			}
		}
	}
	var out []trendPoint
	for _, point := range byDate {
		if point.productivitySet || point.sentimentSet {
			out = append(out, *point)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].date.Before(out[j].date) })
	return out
}

func sentimentLabel(score int) string {
	switch {
	case score > 0:
		return "Positive"
	case score < 0:
		return "Negative"
	default:
		return "Neutral"
	}
}

type trendTableRow struct {
	date, productivity, sentiment string
}

func trendTableRows(points []trendPoint, from, to time.Time) []trendTableRow {
	byDate := make(map[string]trendPoint, len(points))
	for _, point := range points {
		byDate[point.date.Format("2006-01-02")] = point
	}
	var rows []trendTableRow
	for date := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()); !date.After(to); date = date.AddDate(0, 0, 1) {
		point, ok := byDate[date.Format("2006-01-02")]
		row := trendTableRow{date: date.Format("Mon Jan 2, 2006")}
		if !ok {
			row.productivity = "—"
			row.sentiment = "—"
		} else {
			if point.productivitySet {
				row.productivity = fmt.Sprintf("%d/5", point.productivity)
				if point.productivityCount > 1 {
					row.productivity += fmt.Sprintf(" (%d readings)", point.productivityCount)
				}
			} else {
				row.productivity = "—"
			}
			if point.sentimentSet {
				row.sentiment = sentimentLabel(point.sentiment)
				if point.sentimentCount > 1 {
					row.sentiment += fmt.Sprintf(" (%d readings)", point.sentimentCount)
				}
			} else {
				row.sentiment = "—"
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func trendSummary(points []trendPoint, from, to time.Time) string {
	dayCount := 0
	for date := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()); !date.After(to); date = date.AddDate(0, 0, 1) {
		dayCount++
	}
	productivityTotal := 0
	productivityCount := 0
	sentimentCounts := map[string]int{"Positive": 0, "Neutral": 0, "Negative": 0}
	daysWithData := make(map[string]bool)
	for _, point := range points {
		daysWithData[point.date.Format("2006-01-02")] = true
		if point.productivitySet {
			productivityTotal += point.productivity
			productivityCount++
		}
		if point.sentimentSet {
			sentimentCounts[sentimentLabel(point.sentiment)]++
		}
	}
	parts := []string{fmt.Sprintf("%d of %d days logged", len(daysWithData), dayCount)}
	if productivityCount > 0 {
		parts = append(parts, fmt.Sprintf("average productivity %.1f/5", float64(productivityTotal)/float64(productivityCount)))
	}
	if total := sentimentCounts["Positive"] + sentimentCounts["Neutral"] + sentimentCounts["Negative"]; total > 0 {
		parts = append(parts, fmt.Sprintf("sentiment: %d positive, %d neutral, %d negative", sentimentCounts["Positive"], sentimentCounts["Neutral"], sentimentCounts["Negative"]))
	}
	if len(parts) == 1 {
		return parts[0] + " · no valid productivity or sentiment readings"
	}
	return strings.Join(parts, " · ")
}

// formatTrend provides a copyable plain-text representation of the same
// data shown in the Metrics table. Tabs keep columns readable in a
// monospace destination without making the on-screen UI depend on spacing.
func formatTrend(points []trendPoint) string {
	if len(points) == 0 {
		return "(no PRODUCTIVITY/SENTIMENT entries found in range)"
	}
	var sb strings.Builder
	sb.WriteString("Date\tProductivity\tSentiment\n")
	for _, point := range points {
		productivity := "—"
		if point.productivitySet {
			productivity = fmt.Sprintf("%d/5", point.productivity)
		}
		sentiment := "—"
		if point.sentimentSet {
			sentiment = sentimentLabel(point.sentiment)
		}
		fmt.Fprintf(&sb, "%s\t%s\t%s\n", point.date.Format("2006-01-02"), productivity, sentiment)
	}
	return sb.String()
}

func formatTrendRange(points []trendPoint, from, to time.Time) string {
	rows := trendTableRows(points, from, to)
	if len(rows) == 0 {
		return "(no PRODUCTIVITY/SENTIMENT entries found in range)"
	}
	var sb strings.Builder
	sb.WriteString("Date\tProductivity\tSentiment\n")
	for _, row := range rows {
		fmt.Fprintf(&sb, "%s\t%s\t%s\n", row.date, row.productivity, row.sentiment)
	}
	return sb.String()
}

func showTrendView(a fyne.App) {
	rangeSelect := widget.NewSelect([]string{"7 days", "14 days", "30 days", "90 days", "Custom dates"}, nil)
	rangeSelect.SetSelected("30 days")
	fromEntry := newSingleLineEntry()
	fromEntry.SetPlaceHolder("From YYYY-MM-DD")
	toEntry := newSingleLineEntry()
	toEntry.SetPlaceHolder("To YYYY-MM-DD")
	fromEntry.Hide()
	toEntry.Hide()
	status := newExplanatoryLabel("")
	summary := newExplanatoryLabel("")
	copyBtn := widget.NewButton("Copy data", nil)

	var from, to time.Time
	var points []trendPoint
	var rows []trendTableRow
	table := widget.NewTable(
		func() (int, int) { return len(rows) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, object fyne.CanvasObject) {
			label := object.(*widget.Label)
			label.Alignment = fyne.TextAlignLeading
			label.Wrapping = fyne.TextWrapOff
			if id.Row == 0 {
				label.TextStyle = fyne.TextStyle{Bold: true}
			} else {
				label.TextStyle = fyne.TextStyle{}
			}
			values := []string{"Date", "Productivity", "Sentiment"}
			if id.Row > 0 && id.Row-1 < len(rows) {
				row := rows[id.Row-1]
				values = []string{row.date, row.productivity, row.sentiment}
			}
			label.SetText(values[id.Col])
		},
	)
	table.SetColumnWidth(0, 120)
	table.SetColumnWidth(1, 190)
	table.SetColumnWidth(2, 190)
	table.SetRowHeight(0, 28)

	refresh := func() {
		var err error
		if rangeSelect.Selected == "Custom dates" {
			from, err = parseReportDate(fromEntry.Text, false)
			if err == nil {
				to, err = parseReportDate(toEntry.Text, true)
			}
			if err == nil && from.After(to) {
				err = fmt.Errorf("the start date must be before the end date")
			}
		} else {
			days := 30
			if _, scanErr := fmt.Sscanf(rangeSelect.Selected, "%d", &days); scanErr != nil {
				days = 30
			}
			now := time.Now()
			to = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
			from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
		}
		if err != nil {
			status.SetText(err.Error())
			rows = nil
			points = nil
			summary.SetText("")
			table.Refresh()
			return
		}
		status.SetText("")
		points = gatherTrendPointsForRange(from, to)
		rows = trendTableRows(points, from, to)
		summary.SetText(trendSummary(points, from, to))
		table.Refresh()
	}

	rangeSelect.OnChanged = func(selected string) {
		if selected == "Custom dates" {
			fromEntry.Show()
			toEntry.Show()
		} else {
			fromEntry.Hide()
			toEntry.Hide()
		}
		refresh()
	}
	fromEntry.OnChanged = func(string) {
		if rangeSelect.Selected == "Custom dates" {
			refresh()
		}
	}
	toEntry.OnChanged = func(string) {
		if rangeSelect.Selected == "Custom dates" {
			refresh()
		}
	}
	copyBtn.OnTapped = func() { a.Clipboard().SetContent(formatTrendRange(points, from, to)) }
	refresh()

	controls := container.NewVBox(
		container.NewHBox(widget.NewLabel("Range:"), rangeSelect, copyBtn),
		container.NewGridWithColumns(2, fromEntry, toEntry),
		status,
		summary,
		newExplanatoryLabel("A dash means no valid reading was logged. Multiple readings show the latest value and their count. Excluded tags are omitted."),
	)
	w := a.NewWindow("Dunnit: Metrics")
	w.SetContent(windowPad(container.NewBorder(controls, nil, nil, nil, table)))
	w.Resize(fyne.NewSize(620, 560))
	w.Show()
}
