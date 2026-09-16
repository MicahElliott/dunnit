package dun

import (
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// searchResult is one matching line found by searchLedgers, with the
// source file's base name for context.
type searchResult struct {
	file string
	line string
}

// searchLedgers scans every ledger entry (via FilterLedgerEntries,
// backed by the shared AllLedgerEntries() index -- see
// ledgerindex.go) for entries containing query as a case-insensitive
// substring of their Text, returning matches in chronological order.
// Empty query matches nothing (avoids an accidental full dump).
//
// Note: this only matches against parsed Text, not the raw line
// (e.g. the "[HH:MM:SS] CATEGORY " prefix) -- a query like "DONE"
// will match entries whose *text* contains "DONE" but not match
// purely by category the way the old raw-substring scan incidentally
// could. Searching by category is better served by a category filter
// (LedgerQuery.Categories) than by relying on substring matching the
// category code.
func searchLedgers(query string) []searchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	var out []searchResult
	for _, e := range FilterLedgerEntries(LedgerQuery{Text: query}) {
		out = append(out, searchResult{
			file: filepath.Base(e.Source),
			line: "[" + e.Time.Format("15:04:05") + "] " + e.Category + " " + e.Text,
		})
	}
	return out
}

// showSearchDialog lets the user search across all ledger history by
// keyword/tag/category (FR-21), showing matches with file context.
//
// Own standalone window (not a dialog parented on Daybook) -- Daybook
// is normally hidden, and this is a tray-invoked, occasional workflow
// with no dependency on Daybook being open.
func showSearchDialog(a fyne.App) {
	w := a.NewWindow("Dunnit: Search Ledger History")

	queryEntry := widget.NewEntry()
	queryEntry.SetPlaceHolder("Search term (tag, category, or keyword)\u2026")

	resultsBox := container.NewVBox()
	resultsScroll := container.NewVScroll(resultsBox)
	resultsScroll.SetMinSize(fyne.NewSize(0, 300))

	runSearch := func() {
		matches := searchLedgers(queryEntry.Text)
		resultsBox.RemoveAll()
		if len(matches) == 0 {
			resultsBox.Add(widget.NewLabel("(no matches)"))
			resultsBox.Refresh()
			return
		}
		for _, m := range matches {
			resultsBox.Add(itemTextLabel(m.file + ": " + m.line))
		}
		resultsBox.Refresh()
	}
	queryEntry.OnSubmitted = func(string) { runSearch() }
	searchBtn := widget.NewButton("Search", runSearch)

	content := container.NewVBox(
		container.NewBorder(nil, nil, nil, searchBtn, queryEntry),
		resultsScroll,
	)

	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(560, 480))
	w.Show()
}
