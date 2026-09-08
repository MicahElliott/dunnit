package dun

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// removeLastLedgerLine rewrites today's ledger file with its last
// line removed. Always operates on the literal current contents of
// the file (re-read fresh, not a cached value), so it stays correct
// even if the file was edited externally since the app last wrote to
// it (FR-08). No-op if the ledger is empty/missing.
func removeLastLedgerLine() error {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return nil
	}
	return writeLedgerLines(lines[:len(lines)-1])
}

// replaceLastLedgerLine rewrites today's ledger file with its last
// line replaced by newLine. Same freshness guarantee as
// removeLastLedgerLine. No-op if the ledger is empty/missing.
func replaceLastLedgerLine(newLine string) error {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return nil
	}
	lines[len(lines)-1] = newLine
	return writeLedgerLines(lines)
}

// replaceLedgerLineCategoryAt rewrites the line at idx to use
// newCategory instead of its current category, preserving its
// original timestamp and text. No-op if idx is out of range or the
// line isn't well-formed. Used by recordExtended to retroactively
// mark a prior DONE entry as ONGOING when Ditto is used on it.
func replaceLedgerLineCategoryAt(idx int, newCategory string) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[idx], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[idx] = parts[0] + " " + newCategory + " " + parts[2]
	return writeLedgerLines(lines)
}

// replaceLedgerLineTextAt rewrites the line at idx to use newText
// instead of its current text, preserving its original timestamp and
// category, and trimming newText's leading/trailing whitespace. No-op
// if idx is out of range or the line isn't well-formed. Used by
// showEditItemDialog's Save action.
func replaceLedgerLineTextAt(idx int, newText string) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[idx], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[idx] = parts[0] + " " + parts[1] + " " + strings.TrimSpace(newText)
	return writeLedgerLines(lines)
}

// replaceLedgerLineAt rewrites the line at idx to use newCategory and
// newText, preserving its original timestamp, and trimming newText's
// leading/trailing whitespace. No-op if idx is out of range or the
// line isn't well-formed. Used by showEditItemDialog's Save action
// (which lets the user change both category and text together, not
// just text as replaceLedgerLineTextAt alone did).
func replaceLedgerLineAt(idx int, newCategory, newText string) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[idx], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[idx] = parts[0] + " " + newCategory + " " + strings.TrimSpace(newText)
	return writeLedgerLines(lines)
}

// deleteLedgerLineAt removes the line at idx entirely (unlike
// removeLastLedgerLine, which only ever removes the literal last
// line -- this targets an arbitrary line by index, same as
// replaceLedgerLineTextAt/replaceLedgerLineCategoryAt). No-op if idx
// is out of range. Used by showEditItemDialog's Delete action.
func deleteLedgerLineAt(idx int) error {
	lines := readLedgerLines()
	if idx < 0 || idx >= len(lines) {
		return nil
	}
	return writeLedgerLines(append(lines[:idx], lines[idx+1:]...))
}

// showEditItemDialog opens a small modal (dialog.NewCustomWithout
// Buttons -- not a separate window) pre-filled with item.Text,
// letting the user edit its text, change its category (via a
// dropdown restricted to categories sharing the item's own Group --
// see GroupForCode/CategoryOptionsForGroup -- shown with their emoji,
// e.g. "✔️ DONE", and excluding EODOnly codes like ONGOING/SUMMARY/
// PRODUCTIVITY/MEETING_HOURS, which are always machine-written and
// never meant to be hand-picked), Save, Cancel, or Delete the entry
// outright. Calls onSave after a successful save/delete so callers
// can refresh dependent UI. Used by Daybook's inline ✏️ Edit action
// across Planned/Endings/Hilites.
//
// Built on NewCustomWithoutButtons (rather than NewCustomConfirm,
// which only ever creates a fixed Save/Cancel pair) so a third
// Delete action can share the same button row. Esc-to-cancel and
// Tab-to-Save: see dialogEntry's doc comment (dialogentry.go) for why
// a plain widget.Entry can't do either of these correctly, and why
// the fix lives on the Entry itself rather than as a window/canvas-
// level shortcut. Any other custom dialog added to this codebase
// later should use dialogEntry the same way, rather than
// reintroducing the same two bugs.
func showEditItemDialog(parent fyne.Window, item OpenItem, onSave func()) {
	group := GroupForCode(item.Category)
	catOptions := CategoryOptionsForGroup(group)
	itemLabel := item.Category
	for _, c := range Categories {
		if c.Code == item.Category {
			itemLabel = c.Label()
			break
		}
	}
	if len(catOptions) == 0 {
		catOptions = []string{itemLabel} // fallback: shouldn't happen for a real item
	}
	catSelect := widget.NewSelect(catOptions, nil)
	catSelect.SetSelected(itemLabel)

	var d *dialog.CustomDialog
	var saveBtn *widget.Button

	entry := newDialogEntry(
		nil, // Esc wired below, once d exists
		nil, // Tab-forward wired below, once saveBtn exists
	)
	entry.SetText(item.Text)
	// SetMinRowsVisible(2) keeps the box from defaulting to a much
	// taller multi-line entry -- 2 rows is enough for wrapping a
	// typical entry without wasting vertical space on an empty dialog
	// most of the time (dialogEntry is MultiLine so Wrapping can take
	// effect at all, see its doc comment, but that doesn't mean it
	// needs to look tall).
	entry.SetMinRowsVisible(2)

	doSave := func() {
		newCat := item.Category
		if sel := catSelect.Selected; sel != "" {
			if parts := strings.Split(sel, " "); len(parts) == 2 {
				newCat = parts[1]
			}
		}
		if err := replaceLedgerLineAt(item.LineIndex, newCat, entry.Text); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		onSave()
	}

	d = dialog.NewCustomWithoutButtons("Edit Entry",
		container.NewBorder(nil, nil, catSelect, nil, entry), parent)

	entry.onEscape = func() { d.Hide() }
	entry.OnSubmitted = func(string) { doSave() } // Enter submits, same as Save

	saveBtn = widget.NewButton("Save", doSave)
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })
	// Delete removes the entry outright, distinct from Save/Cancel.
	// Confirms nothing further here (no "are you sure?") -- the
	// ledger's append-only design means the raw line is still
	// recoverable by hand from the file if this is ever a mistake,
	// same tradeoff already accepted by Discard/Postpone elsewhere.
	deleteBtn := widget.NewButton("Delete", func() {
		if err := deleteLedgerLineAt(item.LineIndex); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		showToast(parent.Canvas(), "Deleted")
		onSave()
	})
	deleteBtn.Importance = widget.DangerImportance
	d.SetButtons([]fyne.CanvasObject{cancelBtn, deleteBtn, saveBtn})

	// Tab from the entry jumps straight to Save (not Cancel, which
	// Fyne's default Objects-slice-order Tab traversal would land on
	// first -- see dialogEntry's doc comment).
	entry.onTabForward = func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(entry); c != nil {
			c.Focus(saveBtn)
		}
	}

	// Widened ~20% over the prior fixed dialog width (520 -> 624) to
	// fit the extra category dropdown alongside the entry without
	// feeling cramped. Entry itself wraps (dialogEntry's Wrapping)
	// rather than requiring one long unwrapped line -- no literal
	// newlines allowed regardless (see dialogEntry's Return handling).
	// Height trimmed from the original 180 -> 140 now that
	// SetMinRowsVisible(2) keeps the entry itself from defaulting to
	// a much taller box.
	d.Resize(fyne.NewSize(624, 140))
	d.Show()
}

// writeLedgerLines overwrites today's ledger file with the given
// lines (each gets a trailing newline). Invalidates the shared
// ledger entry index (ledgerindex.go) afterward, since every caller
// here (undo/edit/category-rewrite) changes ledger contents outside
// of recordActivity's own append path.
func writeLedgerLines(lines []string) error {
	_, fname := getLedger()
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l + "\n"); err != nil {
			return err
		}
	}
	InvalidateLedgerIndex()
	return nil
}

// showUndoEditLastEntry opens a small window showing the literal last
// line of today's ledger (freshly read from disk), with an editable
// text field and two actions: "Undo" (remove the line entirely) and
// "Save Edit" (replace the line with the edited text) (FR-08). Calls
// onChange after either action succeeds, so callers can refresh any
// dependent UI (e.g. Daybook's "last entry" label, Upcoming list).
func showUndoEditLastEntry(a fyne.App, onChange func()) {
	lines := readLedgerLines()
	if len(lines) == 0 {
		dialog.ShowInformation("Nothing to Undo/Edit",
			"Today's ledger is empty.", nil)
		return
	}
	last := lines[len(lines)-1]

	w := a.NewWindow("Dunnit: Undo/Edit Last Entry")

	entry := widget.NewMultiLineEntry()
	entry.SetText(last)

	undoBtn := widget.NewButton("Undo (Remove)", func() {
		if err := removeLastLedgerLine(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onChange()
		w.Close()
	})
	saveBtn := widget.NewButton("Save Edit", func() {
		if err := replaceLastLedgerLine(entry.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		onChange()
		w.Close()
	})

	w.SetContent(windowPad(container.NewBorder(
		widget.NewLabel("Last ledger line:"), nil, nil, nil,
		container.NewBorder(nil, container.NewHBox(undoBtn, saveBtn), nil, nil, entry),
	)))
	w.Resize(fyne.NewSize(420, 200))
	w.Show()
}
