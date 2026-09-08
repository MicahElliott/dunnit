package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// dialogEntry is a widget.Entry meant for use inside small custom
// modals (e.g. showEditItemDialog, undo.go) that fixes three real
// Fyne gotchas around keyboard-only operation, all discovered from
// the same "Edit Entry" dialog going through a review:
//
//  1. Esc doesn't dismiss a dialog.NewCustomConfirm/NewCustomWithout
//     Buttons by default, and the seemingly-obvious fix -- registering
//     a bare (no-modifier) Escape via Canvas().AddShortcut -- silently
//     never fires: Fyne's desktop driver
//     (internal/driver/glfw/window.go's triggersShortcut) only
//     recognizes a *CustomShortcut* when a real modifier key
//     (Ctrl/Cmd/Alt/Shift) is held; a bare Escape key press never
//     reaches that shortcut-matching code path at all, it goes
//     straight to the focused widget's TypedKey instead. So the only
//     way to actually catch Esc is here, in the focused Entry's own
//     TypedKey, not via any window/canvas-level shortcut.
//  2. Tab, by default, moves focus to the *next* object in the
//     dialog's Objects slice order -- which for a Save/Cancel button
//     pair is Cancel, not Save (Fyne lays out dismiss-then-confirm).
//     onTabForward lets the caller redirect focus straight to
//     whichever control (typically Save) should be next, regardless
//     of slice order.
//  3. A plain multi-line Entry lets Return insert a literal newline.
//     Callers that want word-wrap (MultiLine is required for
//     Wrapping to have any effect at all -- see Entry.textWrap's own
//     "Entry cannot wrap single line" guard) but NOT an actual
//     embedded newline in the recorded text should set OnSubmitted;
//     dialogEntry always treats Return/Enter as "submit", never as
//     "insert newline", regardless of MultiLine.
//
// Any future custom dialog in this codebase wanting proper Esc/Tab/
// Enter behavior should use dialogEntry instead of a plain
// widget.Entry, rather than re-solving these three issues locally.
type dialogEntry struct {
	widget.Entry
	onEscape     func()
	onTabForward func()
}

// newDialogEntry creates a dialogEntry configured for word-wrapped,
// single-conceptual-line editing (MultiLine+Wrapping for visual wrap,
// but Return submits rather than inserting a newline -- see the type
// doc comment above).
func newDialogEntry(onEscape, onTabForward func()) *dialogEntry {
	e := &dialogEntry{onEscape: onEscape, onTabForward: onTabForward}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	return e
}

func (e *dialogEntry) TypedKey(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyEscape:
		if e.onEscape != nil {
			e.onEscape()
			return
		}
	case fyne.KeyTab:
		if e.onTabForward != nil {
			e.onTabForward()
			return
		}
	case fyne.KeyReturn, fyne.KeyEnter:
		if e.OnSubmitted != nil {
			e.OnSubmitted(e.Text)
			return
		}
	}
	e.Entry.TypedKey(k)
}
