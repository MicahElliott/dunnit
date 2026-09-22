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
// line isn't well-formed. Used by lifecycle and Ditto actions to
// change a lifecycle row's category when Start, Done, or Ditto is used.
func replaceLedgerLineCategoryAt(idx int, newCategory string) error {
	return replaceLedgerItemCategoryAt(OpenItem{LineIndex: idx}, newCategory)
}

func replaceLedgerItemCategoryAt(item OpenItem, newCategory string) error {
	lines := readLedgerLinesForItem(item)
	if item.LineIndex < 0 || item.LineIndex >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[item.LineIndex], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[item.LineIndex] = parts[0] + " " + newCategory + " " + parts[2]
	return writeLedgerLinesForItem(item, lines)
}

// replaceLedgerLineTextAt rewrites the line at idx to use newText
// instead of its current text, preserving its original timestamp and
// category, and trimming newText's leading/trailing whitespace. No-op
// if idx is out of range or the line isn't well-formed. Used by
// showEditItemDialog's Save action.
func replaceLedgerLineTextAt(idx int, newText string) error {
	return replaceLedgerItemTextAt(OpenItem{LineIndex: idx}, newText)
}

func replaceLedgerItemTextAt(item OpenItem, newText string) error {
	lines := readLedgerLinesForItem(item)
	if item.LineIndex < 0 || item.LineIndex >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[item.LineIndex], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[item.LineIndex] = parts[0] + " " + parts[1] + " " + normalizeLedgerText(newText)
	return writeLedgerLinesForItem(item, lines)
}

// replaceLedgerLineAt rewrites the line at idx to use newCategory and
// newText, preserving its original timestamp, and trimming newText's
// leading/trailing whitespace. No-op if idx is out of range or the
// line isn't well-formed. Used by showEditItemDialog's Save action
// (which lets the user change both category and text together, not
// just text as replaceLedgerLineTextAt alone did).
func replaceLedgerLineAt(idx int, newCategory, newText string) error {
	return replaceLedgerItemAt(OpenItem{LineIndex: idx}, newCategory, newText)
}

func replaceLedgerItemAt(item OpenItem, newCategory, newText string) error {
	lines := readLedgerLinesForItem(item)
	if item.LineIndex < 0 || item.LineIndex >= len(lines) {
		return nil
	}
	parts := strings.SplitN(lines[item.LineIndex], " ", 3)
	if len(parts) < 3 {
		return nil
	}
	lines[item.LineIndex] = parts[0] + " " + newCategory + " " + normalizeLedgerText(newText)
	return writeLedgerLinesForItem(item, lines)
}

// deleteLedgerLineAt removes the line at idx entirely (unlike
// removeLastLedgerLine, which only ever removes the literal last
// line -- this targets an arbitrary line by index, same as
// replaceLedgerLineTextAt/replaceLedgerLineCategoryAt). No-op if idx
// is out of range. Used by showEditItemDialog's Delete action.
func deleteLedgerLineAt(idx int) error {
	return deleteLedgerItemLine(OpenItem{LineIndex: idx})
}

func deleteLedgerItemLine(item OpenItem) error {
	lines := readLedgerLinesForItem(item)
	if item.LineIndex < 0 || item.LineIndex >= len(lines) {
		return nil
	}
	return writeLedgerLinesForItem(item,
		append(lines[:item.LineIndex], lines[item.LineIndex+1:]...))
}

// showEditItemDialog opens a small modal (dialog.NewCustomWithout
// Buttons -- not a separate window) pre-filled with item.Text, letting the
// user edit its text and category. Ordinary entries are restricted to their
// own Group; lifecycle entries can move among TODO/DOING/DONE/HANDLED/FAIL/WASTED.
// Calls onSave after a successful save/delete so callers can refresh
// dependent UI. Used by Daybook's inline ✏️ Edit action across
// Planned/Endings/Hilites.
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
	showEditItemDialogForCategory(parent, item, item.Category, onSave)
}

// showEditItemDialogForCategory is used by the inline Edit action and
// Planned's checkmark. Lifecycle entries can move between TODO, DOING, DONE,
// HANDLED, FAIL, and WASTED; the leading verb follows the selected category.
func showEditItemDialogForCategory(parent fyne.Window, item OpenItem, initialCategory string, onSave func()) {
	group := GroupForCode(item.Category)
	catOptions := CategoryOptionsForGroup(group)
	if isLifecycleCategory(item.Category) || isLifecycleEndpoint(item.Category) {
		catOptions = lifecycleCategoryOptions()
	}
	if !categoryLabelInOptions(initialCategory, catOptions) {
		initialCategory = item.Category
	}

	entry := newDialogEntry(nil, nil)
	entry.SetText(inflectLifecycleText(item.Text, initialCategory))
	entry.SetMinRowsVisible(2)

	selectedCategory := initialCategory
	catSelect := widget.NewSelect(catOptions, nil)
	catSelect.SetSelected(categoryLabelForCode(initialCategory))
	catSelect.OnChanged = func(selected string) {
		newCategory := categoryCodeFromLabel(selected)
		if newCategory == "" || newCategory == selectedCategory {
			return
		}
		entry.SetText(inflectLifecycleText(entry.Text, newCategory))
		selectedCategory = newCategory
	}

	var d *dialog.CustomDialog
	var saveBtn *widget.Button
	doSave := func() {
		newCat := categoryCodeFromLabel(catSelect.Selected)
		if newCat == "" {
			newCat = item.Category
		}
		text := strings.TrimSpace(entry.Text)
		var err error
		if isLifecycleCategory(item.Category) && isLifecycleEndpoint(newCat) {
			err = completePlannedEndpoint(item, newCat, text)
		} else {
			err = replaceLedgerItemAt(item, newCat, inflectLifecycleText(text, newCat))
		}
		if err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		onSave()
	}

	d = dialog.NewCustomWithoutButtons("Edit Entry",
		container.NewBorder(nil, nil, catSelect, nil, entry), parent)
	entry.onEscape = func() { d.Hide() }
	entry.OnSubmitted = func(string) { doSave() }

	saveBtn = widget.NewButton("Save", doSave)
	saveBtn.Importance = widget.HighImportance
	cancelBtn := widget.NewButton("Cancel", func() { d.Hide() })
	deleteBtn := widget.NewButton("Delete", func() {
		if err := deleteLedgerItemLine(item); err != nil {
			dialog.ShowError(err, parent)
			return
		}
		d.Hide()
		showToast(parent.Canvas(), "Deleted")
		onSave()
	})
	deleteBtn.Importance = widget.DangerImportance
	d.SetButtons([]fyne.CanvasObject{cancelBtn, deleteBtn, saveBtn})
	entry.onTabForward = func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(entry); c != nil {
			c.Focus(saveBtn)
		}
	}
	d.Resize(fyne.NewSize(624, 140))
	d.Show()
}

func lifecycleCategoryOptions() []string {
	var options []string
	for _, code := range []string{"TODO", "DOING", "DONE", "HANDLED", "FAIL", "WASTED"} {
		options = append(options, categoryLabelForCode(code))
	}
	return options
}

func categoryCodeFromLabel(label string) string {
	parts := strings.Split(label, " ")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}

func categoryLabelInOptions(code string, options []string) bool {
	label := categoryLabelForCode(code)
	for _, option := range options {
		if option == label {
			return true
		}
	}
	return false
}

func categoryLabelForCode(code string) string {
	for _, category := range Categories {
		if category.Code == code {
			return category.Label()
		}
	}
	return code
}

// writeLedgerLines overwrites today's ledger file with the given
// lines (each gets a trailing newline). Invalidates the shared
// ledger entry index (ledgerindex.go) afterward, since every caller
// here (undo/edit/category-rewrite) changes ledger contents outside
// of recordActivity's own append path.
func writeLedgerLines(lines []string) error {
	_, fname := getLedger()
	return writeLedgerLinesForPath(fname, lines)
}

func readLedgerLinesForItem(item OpenItem) []string {
	if item.Source != "" {
		return readLedgerLinesFrom(item.Source)
	}
	return readLedgerLines()
}

func writeLedgerLinesForItem(item OpenItem, lines []string) error {
	if item.Source != "" {
		return writeLedgerLinesForPath(item.Source, lines)
	}
	return writeLedgerLines(lines)
}

func writeLedgerLinesForPath(fname string, lines []string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(normalizeLedgerText(l) + "\n"); err != nil {
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
			"Today’s ledger is empty.", nil)
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
