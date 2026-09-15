package dun

import (
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// trendPoint is one day's PRODUCTIVITY (1-5) and/or SENTIMENT
// (Negative/Neutral/Positive, mapped to -1/0/1) reading, parsed from
// that day's ledger (FR-20). Either field may be absent (zero value)
// if that day only logged one of the two.
type trendPoint struct {
	date         time.Time
	productivity int // 0 = not recorded that day, else 1-5
	sentiment    int // -1/0/1, only meaningful if sentimentSet
	sentimentSet bool
}

// sentimentScore maps the SENTIMENT category's free-text values
// (as written by showEODWindow) to a -1..1 numeric score.
func sentimentScore(s string) (int, bool) {
	switch s {
	case "Negative":
		return -1, true
	case "Neutral":
		return 0, true
	case "Positive":
		return 1, true
	}
	return 0, false
}

// gatherTrendPoints scans all ledger entries (via AllLedgerEntries(),
// the shared index) dated within the last days days (inclusive of
// today) and returns one trendPoint per day that has at least one
// PRODUCTIVITY or SENTIMENT entry, oldest first. Reads existing data
// only -- no new capture (FR-20 has no new data requirement).
func gatherTrendPoints(days int) []trendPoint {
	since := time.Now().AddDate(0, 0, -days)
	excludeTags := LoadConfig().ReportExcludeTags
	byDate := map[string]*trendPoint{}
	for _, e := range AllLedgerEntries() {
		if e.Date.Before(since) {
			continue
		}
		if lineHasExcludedTag(e.Text, excludeTags) {
			continue
		}
		key := e.Date.Format("20060102")
		switch e.Category {
		case "PRODUCTIVITY":
			n, err := strconv.Atoi(strings.TrimSpace(e.Text))
			if err != nil {
				continue
			}
			pt := byDate[key]
			if pt == nil {
				pt = &trendPoint{date: e.Date}
				byDate[key] = pt
			}
			pt.productivity = n
		case "SENTIMENT":
			score, ok := sentimentScore(strings.TrimSpace(e.Text))
			if !ok {
				continue
			}
			pt := byDate[key]
			if pt == nil {
				pt = &trendPoint{date: e.Date}
				byDate[key] = pt
			}
			pt.sentiment = score
			pt.sentimentSet = true
		}
	}
	var out []trendPoint
	for _, pt := range byDate {
		out = append(out, *pt)
	}
	// sort oldest-first
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].date.Before(out[j-1].date); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// sentimentGlyph renders a crude plain-text sparkline character for a
// sentiment score.
func sentimentGlyph(score int) string {
	switch {
	case score > 0:
		return "+"
	case score < 0:
		return "-"
	default:
		return "="
	}
}

// formatTrend renders points as a plain-text table, one line per day.
func formatTrend(points []trendPoint) string {
	if len(points) == 0 {
		return "(no PRODUCTIVITY/SENTIMENT entries found in range)"
	}
	var sb strings.Builder
	sb.WriteString("Date        Productivity  Sentiment\n")
	for _, p := range points {
		prod := "-"
		if p.productivity > 0 {
			prod = strings.Repeat("*", p.productivity) + strings.Repeat(" ", 5-p.productivity)
		} else {
			prod = strings.Repeat(" ", 5)
		}
		sent := "-"
		if p.sentimentSet {
			sent = sentimentGlyph(p.sentiment)
		}
		sb.WriteString(p.date.Format("2006-01-02") + "  " + prod + "         " + sent + "\n")
	}
	return sb.String()
}

// showTrendView opens a window showing a plain-text productivity/
// sentiment trend over a selectable range (FR-20).
func showTrendView(a fyne.App) {
	rangeSelect := widget.NewSelect([]string{"7", "14", "30", "90"}, nil)
	rangeSelect.SetSelected("30")

	body := widget.NewMultiLineEntry()
	body.Wrapping = fyne.TextWrapOff
	body.SetMinRowsVisible(20)

	refresh := func() {
		days, err := strconv.Atoi(rangeSelect.Selected)
		if err != nil || days <= 0 {
			days = 30
		}
		body.SetText(formatTrend(gatherTrendPoints(days)))
	}
	rangeSelect.OnChanged = func(string) { refresh() }
	refresh()

	w := a.NewWindow("Dunnit: Productivity/Sentiment Trend")
	w.SetContent(windowPad(container.NewBorder(
		container.NewHBox(widget.NewLabel("Last N days:"), rangeSelect),
		nil, nil, nil,
		body,
	)))
	w.Resize(fyne.NewSize(480, 480))
	w.Show()
}
