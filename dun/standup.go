package dun

import (
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// standupCategories are the categories pulled into the deterministic
// standup export (FR-17) -- everything from the "End" and "Hilite"
// groups (day-to-day capture/endpoints, plus freestanding notable-
// event markers -- see docs/category-taxonomy.md), except:
//   - ONGOING: purely Ditto's internal rewrite marker, never a real
//     "what I did" item.
//   - EODOnly categories (SUMMARY/PRODUCTIVITY/MEETING_HOURS): day-
//     level wrap-up meta-notes, not standup-worthy activity of their
//     own.
//
// Built dynamically from Categories (via categoryGroupOrder) rather
// than a hardcoded list, so a future category addition to either
// group is picked up here automatically without a second edit.
// Originally just {"DONE", "WIN"} -- widened 2026-09-03 per feedback
// that a standup's "yesterday" should reflect the fuller picture
// (learnings, setbacks, notable events), not just completions/wins.
var standupCategories = buildStandupCategories()

func buildStandupCategories() map[string]bool {
	out := map[string]bool{}
	for _, group := range []string{"end", "hilite"} {
		for _, code := range categoryGroupOrder(group) {
			out[code] = true
		}
	}
	for _, c := range Categories {
		if c.EODOnly || c.Code == "ONGOING" {
			delete(out, c.Code)
		}
	}
	return out
}

// standupTag is the tag whose recurring meeting entry (if configured)
// drives the "since" boundary for gatherStandupLines -- see
// standupWindowStart.
const standupTag = "#dsu"

// standupWindowStart returns the timestamp standup items should be
// gathered from (exclusive lower bound), given now: if a "#dsu"
// RecurringMeeting is configured, that's its lastOccurrence (the
// meeting time itself) -- so the standup naturally covers "everything
// since the last time this meeting happened," spanning across
// midnight into today, not just "yesterday's ledger". Falls back to
// standupSourceDates' fixed weekday-aware heuristic (start of
// yesterday, or Friday on a Monday) if no #dsu meeting is configured,
// preserving the original behavior for anyone who hasn't set one up.
func standupWindowStart(cfg Config, now time.Time) time.Time {
	for _, m := range cfg.RecurringMeetings {
		if strings.EqualFold(m.Tag, standupTag) {
			return lastOccurrence(m, now)
		}
	}
	dates := standupSourceDates(now)
	return time.Date(dates[0].Year(), dates[0].Month(), dates[0].Day(), 0, 0, 0, 0, now.Location())
}

// standupSourceDates returns the dates whose ledgers should feed the
// standup export, given now. Normally just "yesterday". On Monday,
// also include Saturday and Sunday (in addition to Friday) in case
// weekend work was logged, since those days otherwise have no
// standup of their own to surface them. Returned oldest-first. Used
// as a fallback by standupWindowStart when no #dsu recurring meeting
// is configured, and to enumerate which ledger files to scan (a
// timestamp alone doesn't tell you which files to open).
func standupSourceDates(now time.Time) []time.Time {
	yesterday := now.AddDate(0, 0, -1)
	if now.Weekday() != time.Monday {
		return []time.Time{yesterday}
	}
	friday := now.AddDate(0, 0, -3)
	saturday := now.AddDate(0, 0, -2)
	sunday := yesterday
	return []time.Time{friday, saturday, sunday}
}

// ledgerFileForDate returns the ledger file path for date if it
// exists among allLedgerFiles(), or "" if none.
func ledgerFileForDate(date time.Time) string {
	target := "ledger-" + date.Format("20060102") + ".txt"
	for _, path := range allLedgerFiles() {
		if strings.HasSuffix(path, target) {
			return path
		}
	}
	return ""
}

// parseLedgerLineTime parses a ledger line's "[HH:MM:SS]" timestamp
// prefix combined with the given date into a full time.Time, for
// comparing against standupWindowStart's boundary. Returns ok=false
// if the line doesn't start with a well-formed "[HH:MM:SS]" stamp.
func parseLedgerLineTime(line string, date time.Time) (t time.Time, ok bool) {
	if len(line) < 10 || line[0] != '[' {
		return time.Time{}, false
	}
	end := strings.IndexByte(line, ']')
	if end < 0 {
		return time.Time{}, false
	}
	hms := line[1:end]
	parsed, err := time.ParseInLocation("15:04:05", hms, date.Location())
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, date.Location()), true
}

// gatherStandupLines reads ledger files covering standupSourceDates
// (plus today's, so a window that started yesterday but runs through
// "now" still picks up this morning's entries) and returns the
// standupCategories lines' text (category prefix stripped) that fall
// at or after since, oldest-first, deduplicated (exact-text repeats
// collapsed to one bullet, first occurrence kept).
func gatherStandupLines(cfg Config, now time.Time) []string {
	since := standupWindowStart(cfg, now)
	dates := append(append([]time.Time{}, standupSourceDates(now)...), now)
	excludeTags := LoadConfig().ReportExcludeTags

	seen := make(map[string]bool)
	var out []string
	for _, date := range dates {
		path := ledgerFileForDate(date)
		if path == "" {
			continue
		}
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !standupCategories[cat] {
				continue
			}
			if lineHasExcludedTag(line, excludeTags) {
				continue
			}
			ts, ok := parseLedgerLineTime(line, date)
			if !ok || ts.Before(since) {
				continue
			}
			if seen[text] {
				continue
			}
			seen[text] = true
			out = append(out, text)
		}
	}
	return out
}

// formatStandup renders lines as a simple bullet list, or a
// placeholder message if there's nothing to report.
func formatStandup(lines []string) string {
	if len(lines) == 0 {
		return "(no standup-worthy entries found for the covered period)"
	}
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("\u2022 ")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

// summarizeStandupWithCopilot runs the given (already-filtered,
// hidden items excluded) lines through the same summarizeWithCopilot
// pipeline used elsewhere, framed for a standup specifically.
//
// Prompt reframed 2026-09-03 to explicit classic-scrum structure
// (What did you do yesterday / What will you do today / Risks &
// blockers) rather than a generic "well-organized update" -- per
// feedback that the prior framing didn't read as a real standup.
// "What will you do today" has no dedicated data source of its own
// (gatherStandupLines only ever pulls standupCategories -- "End" and
// "Hilite" group entries, i.e. backward-looking items) -- today's still-open TODOs/GOALs (getOpenItems) are passed
// alongside as explicit "currently open" context so the model has
// real material for that section instead of inventing generic filler.
// Also explicitly instructs the model to surface a "what do you need
// help with" callout and favor concrete results over a busy-sounding
// activity log.
func summarizeStandupWithCopilot(lines []string) (string, error) {
	var openBuf strings.Builder
	for _, item := range getOpenItems() {
		if item.Category != "TODO" && item.Category != "GOAL" {
			continue
		}
		openBuf.WriteString("- " + stripCarryForwardSince(item.Text) + "\n")
	}
	openSection := "(nothing currently open)"
	if openBuf.Len() > 0 {
		openSection = openBuf.String()
	}

	input := "Completed/notable items:\n" + strings.Join(lines, "\n") +
		"\n\nCurrently open TODOs/GOALs (candidates for \"today\"):\n" + openSection

	return summarizeWithCopilotPrompt(
		"Turn this into a classic scrum daily standup update, structured "+
			"under exactly these three headings: "+
			"\"What did I do yesterday\", \"What will I do today\", and "+
			"\"Risks / blockers\". Base \"yesterday\" on the completed/"+
			"notable items given; base \"today\" on the currently-open "+
			"TODOs/GOALs given (pick the most relevant ones, don't just "+
			"dump the whole list verbatim). If there's nothing worth "+
			"flagging as a risk or blocker, say so briefly rather than "+
			"omitting the heading. Explicitly call out anything the "+
			"person might need help with, as its own short line under "+
			"Risks/blockers if applicable. Focus on concrete results "+
			"and outcomes rather than a busy-sounding activity log. Be "+
			"concise -- bullet points, not prose.", input)
}

// showGeneratedStandupSummary displays an AI-generated standup
// summary via the shared showGeneratedReport window (Copy/Save/
// Close), saving to periodReportPath("dsu", today, "20060102").
func showGeneratedStandupSummary(a fyne.App, parent fyne.Window, summary string) {
	showGeneratedReport(a, "Dunnit: Generated Standup Summary",
		periodReportPath("dsu", time.Now(), "20060102"), summary)
}

// showStandupExport builds the deterministic standup summary (FR-17
// -- no LLM/network call, pure local ledger parsing) covering
// everything since the last #dsu meeting (or the weekday-aware
// yesterday fallback), copies it to the clipboard, and shows it in an
// **editable** text area (2026-09-03, replacing the old per-line
// Hide-icon list) seeded with one gathered item per line, plus a
// bottom "Generate Summary" button that runs whatever's currently in
// that box (after the user's own edits, additions, or deletions)
// through summarizeStandupWithCopilot and shows the result via
// showGeneratedStandupSummary.
//
// Editing here only changes what's fed into *this generation's*
// prompt input -- it never writes anything back to the ledger itself
// (an instructional note above the box says so explicitly). This
// replaces the old Hide-only affordance with something strictly more
// capable (hide was really just "remove a line from what gets sent,"
// which free-text editing already covers, plus now supports adding
// a line the deterministic gather step didn't pick up, or fixing/
// clarifying wording before it goes to the LLM).
func showStandupExport(a fyne.App) {
	now := time.Now()
	cfg := LoadConfig()
	lines := gatherStandupLines(cfg, now)
	text := formatStandup(lines)
	a.Clipboard().SetContent(text)

	w := a.NewWindow("Dunnit: Standup Summary")

	itemsEntry := widget.NewMultiLineEntry()
	itemsEntry.SetMinRowsVisible(10)
	if len(lines) == 0 {
		itemsEntry.SetPlaceHolder("(no standup-worthy entries found for the covered period -- type your own below if you like)")
	} else {
		itemsEntry.SetText(strings.Join(lines, "\n"))
	}

	generateBtn := widget.NewButton("Generate Summary", func() {
		var visible []string
		for _, line := range strings.Split(itemsEntry.Text, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				visible = append(visible, line)
			}
		}
		if len(visible) == 0 {
			dialog.ShowInformation("Nothing to Summarize", "The items box is empty.", w)
			return
		}
		progress := dialog.NewCustomWithoutButtons("Generating Summary", widget.NewLabel("Running gh copilot, please wait\u2026"), w)
		progress.Show()
		go func() {
			summary, err := summarizeStandupWithCopilot(visible)
			fyne.Do(func() {
				progress.Hide()
				if err != nil {
					log.Println("Error generating standup summary:", err)
					dialog.ShowError(err, w)
					return
				}
				showGeneratedStandupSummary(a, w, summary)
			})
		}()
	})

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Standup items since %s:", standupWindowStart(cfg, now).Format("Mon 15:04"))),
			widget.NewLabelWithStyle(
				"Edit freely before generating -- add, remove, or reword lines "+
					"(one item per line). This only changes what's sent to the "+
					"summary prompt; it never edits the ledger itself.",
				fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		),
		generateBtn,
		nil, nil,
		itemsEntry,
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(480, 440))
	w.Show()
}
