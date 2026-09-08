package dun

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// summaryPeriod identifies how far back to gather ledger entries for a
// summary.
type summaryPeriod string

const (
	periodDay     summaryPeriod = "Day"
	periodWeek    summaryPeriod = "Week"
	periodMonth   summaryPeriod = "Month"
	periodQuarter summaryPeriod = "Quarter"
	periodYear    summaryPeriod = "Year"
)

// ledgerFilesFor returns the paths of ledger files under DunnitDir()
// whose date falls within the given period, relative to now.
func ledgerFilesFor(period summaryPeriod, now time.Time) []string {
	var cutoff time.Time
	switch period {
	case periodDay:
		cutoff = now.AddDate(0, 0, -1)
	case periodWeek:
		cutoff = now.AddDate(0, 0, -7)
	case periodMonth:
		cutoff = now.AddDate(0, -1, 0)
	case periodQuarter:
		cutoff = now.AddDate(0, -3, 0)
	case periodYear:
		cutoff = now.AddDate(-1, 0, 0)
	default:
		cutoff = now.AddDate(0, 0, -1)
	}

	var files []string
	for _, path := range allLedgerFiles() {
		datePart := ledgerFileDate(path)
		if datePart == nil {
			continue
		}
		if !datePart.Before(cutoff) && !datePart.After(now) {
			files = append(files, path)
		}
	}
	return files
}

// allLedgerFiles returns the paths of every ledger-*.txt file under
// DunnitDir(), regardless of date, in filesystem walk order.
func allLedgerFiles() []string {
	var files []string
	root := DunnitDir()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasPrefix(name, "ledger-") || !strings.HasSuffix(name, ".txt") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

// ledgerFileDate parses the YYYYMMDD date out of a ledger file's
// name (e.g. ".../ledger-20260828.txt"), returning nil if it doesn't
// match the expected naming pattern.
func ledgerFileDate(path string) *time.Time {
	name := filepath.Base(path)
	datePart := strings.TrimSuffix(strings.TrimPrefix(name, "ledger-"), ".txt")
	t, err := time.ParseInLocation("20060102", datePart, time.Local)
	if err != nil {
		return nil
	}
	return &t
}

// gatherLedgerText concatenates the content of all ledger files for the
// given period into one string (file path headers included, for
// context).
func gatherLedgerText(period summaryPeriod) string {
	files := ledgerFilesFor(period, time.Now())
	return concatLedgerFiles(files)
}

// gatherLedgerTextForDate is like gatherLedgerText, but scoped to a
// single specific date's ledger file rather than a rolling period.
// Used by FR-18's daily summary doc, which always covers exactly one
// day. Returns "" if that day's ledger doesn't exist / has no
// content.
func gatherLedgerTextForDate(date time.Time) string {
	path := ledgerFileForDate(date)
	if path == "" {
		return ""
	}
	return concatLedgerFiles([]string{path})
}

// gatherLedgerTextForRange concatenates ledger content for files
// dated within [from, to] inclusive -- used by FR-22 (year range) and
// FR-23 (arbitrary date range), sharing the same plumbing as the
// rolling-period/single-date variants above.
func gatherLedgerTextForRange(from, to time.Time, categories map[string]bool) string {
	var files []string
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil || date.Before(from) || date.After(to) {
			continue
		}
		files = append(files, path)
	}
	if len(categories) == 0 {
		return concatLedgerFiles(files)
	}
	return concatLedgerFilesFiltered(files, categories)
}

// concatLedgerFiles concatenates the content of the given ledger
// files into one string (file path headers included, for context),
// skipping any line matching one of Config.ReportExcludeTags (see
// lineHasExcludedTag) -- applied here, the lowest-level shared
// concatenation helper, so every report/summary pipeline (Standup,
// Status Report, Annual Review, Trend View, Kickoff/Review digests,
// etc) gets the exclusion applied uniformly without each caller
// needing its own filtering logic.
func concatLedgerFiles(files []string) string {
	excludeTags := LoadConfig().ReportExcludeTags
	var sb strings.Builder
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sb.WriteString("# " + filepath.Base(path) + "\n")
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if lineHasExcludedTag(line, excludeTags) {
				continue
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		f.Close()
	}
	return sb.String()
}

// lineHasExcludedTag reports whether line contains any of
// excludeTags (each an exact "#tag" token, matched the same way tags
// are matched everywhere else -- see extractTags -- except
// case-insensitively). Case-insensitive matching was added 2026-09-03
// after a real bug report: Settings' exclude-tags field and a ledger
// line can easily end up with differently-cased versions of "the
// same" tag (e.g. "#Home" logged vs "#home" configured) since nothing
// else in the app normalizes tag casing, and every other tag-matching
// path (autocomplete, KnownTags) is exact-match by design for display
// purposes -- exclusion is different: a user configuring "#home" as
// noise to filter out clearly means to catch "#Home" too, not silently
// let it through.
func lineHasExcludedTag(line string, excludeTags []string) bool {
	if len(excludeTags) == 0 {
		return false
	}
	tags := extractTags(line)
	if len(tags) == 0 {
		return false
	}
	excluded := make(map[string]bool, len(excludeTags))
	for _, t := range excludeTags {
		excluded[strings.ToLower(t)] = true
	}
	for _, t := range tags {
		if excluded[strings.ToLower(t)] {
			return true
		}
	}
	return false
}

// concatLedgerFilesFiltered is like concatLedgerFiles, but only
// includes lines whose category is in categories (and, same as
// concatLedgerFiles, still skips any line matching a
// Config.ReportExcludeTags tag).
func concatLedgerFilesFiltered(files []string, categories map[string]bool) string {
	excludeTags := LoadConfig().ReportExcludeTags
	var sb strings.Builder
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		var wroteHeader bool
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			cat, _, ok := parseLedgerLine(line)
			if !ok || !categories[cat] {
				continue
			}
			if lineHasExcludedTag(line, excludeTags) {
				continue
			}
			if !wroteHeader {
				sb.WriteString("# " + filepath.Base(path) + "\n")
				wroteHeader = true
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		f.Close()
	}
	return sb.String()
}

// summarizeWithCopilotPrompt is like summarizeWithCopilot, but lets
// the caller supply the full instruction text prepended before the
// ledger content -- used by FR-22/FR-23's differently-framed prompts
// (performance-review-flavored, private-vs-shareable) while reusing
// the exact same gh copilot invocation plumbing.
func summarizeWithCopilotPrompt(instructions, ledgerText string) (string, error) {
	prompt := instructions + "\n\n" + ledgerText

	cmd := exec.Command("gh", "copilot", "-p", prompt, "--silent", "--allow-all-tools")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh copilot failed: %w\n%s", err, out)
	}
	return string(out), nil
}

// summarizeWithCopilot shells out to `gh copilot` with a one-shot prompt
// asking it to summarize the given ledger text into a brief impact
// report. Requires the `gh` CLI with the Copilot extension available.
func summarizeWithCopilot(ledgerText string) (string, error) {
	return summarizeWithCopilotPrompt(
		"Summarize this ledger of daily activity entries into a brief "+
			"impact report suitable for a standup or status update. Be concise "+
			"and group related work together.", ledgerText)
}

// showSummarizeDialog lets the user pick a period, then runs the
// summary and displays the result in a new window.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showSummarizeDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Summarize")

	options := []string{string(periodDay), string(periodWeek), string(periodMonth), string(periodQuarter)}
	periodSelect := widget.NewSelect(options, nil)
	periodSelect.SetSelected(string(periodDay))

	generate := func() {
		period := summaryPeriod(periodSelect.Selected)
		w.Close()
		runSummarize(a, period)
	}

	content := container.NewVBox(
		widget.NewLabel("Summarize accomplishments for:"),
		periodSelect,
		container.NewHBox(
			widget.NewButton("Generate", generate),
			widget.NewButton("Cancel", func() { w.Close() }),
		),
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(300, 140))
	w.Show()
}

func runSummarize(a fyne.App, period summaryPeriod) {
	ledgerText := gatherLedgerText(period)
	if strings.TrimSpace(ledgerText) == "" {
		w := a.NewWindow("Dunnit: Summary")
		w.SetContent(windowPad(widget.NewLabel("No ledger entries found for that period.")))
		w.Show()
		return
	}

	progress := a.NewWindow("Dunnit: Summarizing\u2026")
	progress.SetContent(windowPad(widget.NewLabel(
		"Asking gh copilot to summarize, please wait\u2026\n" +
			"The generated report will be copied to your clipboard automatically.")))
	progress.Show()

	go func() {
		summary, err := summarizeWithCopilot(ledgerText)
		fyne.Do(func() {
			progress.Close()
			w := a.NewWindow(fmt.Sprintf("Dunnit: %s Summary", period))
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
