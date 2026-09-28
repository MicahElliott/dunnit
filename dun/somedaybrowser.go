package dun

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// somedayCategory is the single category browsed/re-promoted here.
// Factored into a const since it's referenced in several places below
// (matching resolution suffixes, filtering AllLedgerEntries, etc).
const somedayCategory = "SOMEDAY"

// gatherSomedayItems scans every ledger entry (all history, via
// AllLedgerEntries) for SOMEDAY lines that have not since been
// re-promoted back to TODO/GOAL (via promoteSomedayItem below) or
// discarded from the browser -- see docs/todo-carryforward-design.md's
// "SOMEDAY becomes ledger-history-only" section for why this view
// exists: since Postpone (recordPostponed) deliberately removes an
// item from daily carry-forward, SOMEDAY items are otherwise only
// ever visible by manually reading ledger files.
//
// Uses the exact same resolution-suffix convention as
// todos.go's parseOpenItems (convertedSuffix("SOMEDAY"), i.e.
// "(via SOMEDAY)") so promoting/discarding here reuses
// recordConvertedDone-style bookkeeping rather than inventing a new
// marker -- a SOMEDAY item is considered "handled" here once a later
// ledger entry's text matches "<original text> (via SOMEDAY)",
// regardless of what category that later entry is (TODO, GOAL, or
// DISCARDED, all written by this file's actions below).
func gatherSomedayItems() []OpenItem {
	entries := AllLedgerEntries()
	handledSuffix := convertedSuffix(somedayCategory)

	handled := make(map[string]bool)
	itemsByKey := make(map[string]OpenItem)
	var itemOrder []string
	for _, e := range entries {
		if strings.HasSuffix(e.Text, handledSuffix) {
			orig := strings.TrimSuffix(e.Text, handledSuffix)
			handled[somedayItemKey(orig)] = true
			continue
		}
		if e.Category == somedayCategory {
			text := stripCarryForwardSince(e.Text)
			key := somedayItemKey(text)
			if _, exists := itemsByKey[key]; !exists {
				itemOrder = append(itemOrder, key)
			}
			itemsByKey[key] = OpenItem{Category: somedayCategory, Text: text}
		}
	}

	var out []OpenItem
	for _, key := range itemOrder {
		if handled[key] {
			continue
		}
		out = append(out, itemsByKey[key])
	}
	return out
}

// somedayItemKey is the logical identity used only by this browser. SOMEDAY
// rows are a backlog, so repeated postponements of the same task should still
// produce one triage row even when copied metadata differs.
func somedayItemKey(text string) string {
	key, ok := carryForwardEntryText(LedgerEntry{Category: somedayCategory, Text: text})
	if ok {
		return key
	}
	return strings.ToLower(strings.Join(strings.Fields(text), " "))
}

// promoteSomedayItem re-activates a browsed SOMEDAY item as a fresh
// TODO or GOAL entry (logged today, append-only -- the original
// SOMEDAY line is untouched), and stamps a "(via SOMEDAY)" resolution
// marker so gatherSomedayItems stops listing it afterward. Mirrors
// recordConvertedDone/recordPostponed's shape (todos.go).
func promoteSomedayItem(item OpenItem, newCategory string) error {
	text := inflectLifecycleText(item.Text, newCategory)
	if err := recordActivity(text, newCategory); err != nil {
		return err
	}
	return recordActivity(text+convertedSuffix(somedayCategory), "DISCARDED")
}

// discardSomedayItem marks a browsed SOMEDAY item as permanently
// handled (removed from this browser) without promoting it anywhere
// -- for cleaning out SOMEDAY items that turned out not to be worth
// keeping after all.
func discardSomedayItem(item OpenItem) error {
	return recordActivity(item.Text+convertedSuffix(somedayCategory), "DISCARDED")
}

// showSomedayBrowserWindow lists every still-open SOMEDAY item across
// all ledger history (gatherSomedayItems) with per-item "-> TODO",
// "-> GOAL", and "Discard" actions -- the one place SOMEDAY items
// (postponed via Daybook's Planned section, EOD, or period Review)
// can actually be found and re-promoted, since Postpone deliberately
// removes them from daily carry-forward. See
// docs/todo-carryforward-design.md.
func showSomedayBrowserWindow(a fyne.App) {
	w := a.NewWindow("Dunnit: SOMEDAY Items")

	listBox := container.NewVBox()

	var refresh func()
	refresh = func() {
		listBox.RemoveAll()
		items := gatherSomedayItems()
		if len(items) == 0 {
			listBox.Add(widget.NewLabel("No SOMEDAY items \u2014 nothing postponed right now."))
			listBox.Refresh()
			return
		}
		listBox.Add(widget.NewLabel(fmt.Sprintf("%d SOMEDAY item(s):", len(items))))
		for _, item := range items {
			item := item // capture
			row := itemTextLabel(categoryIconPrefix(item.Category) + item.Text)
			promoteTodoBtn := newHoverIconButton(theme.Icon(theme.IconNameConfirm), "Make TODO", func() {
				if err := promoteSomedayItem(item, "TODO"); err != nil {
					logOperationError("promote SOMEDAY item to TODO", err)
					dialog.ShowError(err, w)
					return
				}
				refresh()
			})
			promoteGoalBtn := newHoverIconButton(theme.Icon(theme.IconNameDocumentCreate), "Make GOAL", func() {
				if err := promoteSomedayItem(item, "GOAL"); err != nil {
					logOperationError("promote SOMEDAY item to GOAL", err)
					dialog.ShowError(err, w)
					return
				}
				refresh()
			})
			discardBtn := newHoverIconButton(theme.Icon(theme.IconNameDelete), "Discard", func() {
				if err := discardSomedayItem(item); err != nil {
					logOperationError("discard SOMEDAY item", err)
					dialog.ShowError(err, w)
					return
				}
				refresh()
			})
			listBox.Add(container.NewBorder(nil, nil, nil,
				container.NewHBox(promoteTodoBtn, promoteGoalBtn, discardBtn), row))
		}
		listBox.Refresh()
	}
	refresh()

	content := container.NewBorder(
		container.NewVBox(
			newWindowHeading("🕰️ SOMEDAY items"),
			newExplanatoryLabel("These are postponed possibilities. Make one a TODO or GOAL when it deserves attention, or discard it when it no longer matters."),
		),
		widget.NewButton("Close", w.Close),
		nil, nil,
		container.NewVScroll(listBox),
	)
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(520, 480))
	w.Show()
}
