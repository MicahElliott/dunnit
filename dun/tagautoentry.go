package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// maxTagSuggestions caps how many matches tagAutoEntry will display
// at once, so a broad fragment (e.g. a single common letter) doesn't
// blow up the suggestion box to an unreasonable height.
const maxTagSuggestions = 8

// tagAutoEntry is a widget.Entry that shows tag-autocomplete
// suggestions (FR-10) as the user types a "#tag" fragment, entirely
// inline -- as a plain sibling widget in the surrounding layout
// (see SuggestionBox) -- rather than in a canvas overlay.
//
// History/why not an overlay (widget.PopUpMenu, then widget.PopUp
// were both tried and both failed): Fyne's canvas keyboard-focus
// routing (internal/driver/common/canvas.go's focusManager()) always
// prefers the *top-most overlay's own FocusManager* over the window
// content's FocusManager whenever ANY overlay exists on the canvas's
// overlay stack -- irrespective of which object canvas.Focus() was
// last called on. Our suggestion popups contained only plain buttons/
// menu items (no Focusable text entry), so that overlay's
// FocusManager.Focused() was always nil -- meaning *all* keyboard
// input (typing, Escape, Enter) went nowhere the instant the popup
// appeared, no matter how aggressively we tried to refocus `input`.
// This explains both the "can't type past the first letter" bug and
// the "can't even Esc out" regression: they're the same root cause,
// an overlay existing at all, not something a better refocus
// workaround could fix.
//
// The only real fix is to never create an overlay for this feature:
// suggestions render as an ordinary VBox of buttons that the caller
// keeps hidden/shown as a plain sibling of the entry in its own
// layout, so keyboard focus never leaves the Entry in the first
// place. Arrow-key navigation of the suggestion list (Up/Down to
// move the highlighted suggestion, Enter to accept it, Escape to
// dismiss without accepting) is handled by overriding TypedKey here;
// everything else falls through to the embedded Entry's normal
// handling (including Up/Down's ordinary cursor-movement behavior
// when no suggestions are currently showing).
type tagAutoEntry struct {
	widget.Entry

	suggestions []string
	selected    int
	fragStart   int
	box         *fyne.Container
}

func newTagAutoEntry() *tagAutoEntry {
	e := &tagAutoEntry{}
	e.ExtendBaseWidget(e)
	e.initTagAutoEntry()
	return e
}

// initTagAutoEntry wires up the suggestion box and OnChanged handler.
// Factored out (rather than folded into newTagAutoEntry) so a struct
// that embeds tagAutoEntry (e.g. closeShortcutEntry) can call this
// itself after its own ExtendBaseWidget(outer) call, keeping Fyne's
// "extend with the outermost type" widget contract intact.
func (e *tagAutoEntry) initTagAutoEntry() {
	e.box = container.NewVBox()
	e.box.Hide()
	e.Entry.OnChanged = e.handleChanged
}

// SuggestionBox returns the (initially hidden) inline suggestion list
// -- add this as a plain sibling of the entry itself in the
// surrounding layout (directly below it), not in any overlay/popup.
func (e *tagAutoEntry) SuggestionBox() fyne.CanvasObject {
	return e.box
}

func (e *tagAutoEntry) handleChanged(text string) {
	start, fragment, ok := currentTagFragment(text, e.CursorColumn)
	if !ok || len(fragment) < 2 { // need at least "#" + 1 char
		e.clearSuggestions()
		return
	}
	matches := matchingTags(KnownTags(), fragment[1:])
	if len(matches) == 0 {
		e.clearSuggestions()
		return
	}
	if len(matches) > maxTagSuggestions {
		matches = matches[:maxTagSuggestions]
	}
	e.suggestions = matches
	e.selected = 0
	e.fragStart = start
	e.render()
}

func (e *tagAutoEntry) clearSuggestions() {
	if len(e.suggestions) == 0 {
		return
	}
	e.suggestions = nil
	e.selected = 0
	e.box.RemoveAll()
	e.box.Hide()
	e.box.Refresh()
}

// render rebuilds the suggestion box's buttons, marking the currently
// arrow-selected one with a leading "▸" -- clicking a button still
// works (mouse remains supported), but Up/Down/Enter/Escape is the
// primary intended path per Dunnit's mouseless-workflow goal.
func (e *tagAutoEntry) render() {
	e.box.RemoveAll()
	for i, tag := range e.suggestions {
		i, tag := i, tag
		label := "  " + tag
		if i == e.selected {
			label = "\u25b8 " + tag
		}
		e.box.Add(widget.NewButton(label, func() { e.accept(tag) }))
	}
	e.box.Show()
	e.box.Refresh()
}

// accept replaces the in-progress "#fragment" with tag and dismisses
// the suggestion list, leaving the cursor immediately after the
// inserted tag.
func (e *tagAutoEntry) accept(tag string) {
	runes := []rune(e.Entry.Text)
	end := e.CursorColumn
	if end > len(runes) {
		end = len(runes)
	}
	start := e.fragStart
	if start > end {
		start = end
	}
	newText := string(runes[:start]) + tag + string(runes[end:])
	e.SetText(newText)
	e.CursorColumn = start + len([]rune(tag))
	e.clearSuggestions()
}

// TypedKey intercepts Up/Down/Escape/Enter while suggestions are
// showing (arrow-navigate/dismiss/accept); everything else -- and all
// of the above when no suggestions are showing -- falls through to
// the embedded Entry's normal key handling.
func (e *tagAutoEntry) TypedKey(key *fyne.KeyEvent) {
	if len(e.suggestions) > 0 {
		switch key.Name {
		case fyne.KeyDown:
			e.selected = (e.selected + 1) % len(e.suggestions)
			e.render()
			return
		case fyne.KeyUp:
			e.selected = (e.selected - 1 + len(e.suggestions)) % len(e.suggestions)
			e.render()
			return
		case fyne.KeyEscape:
			e.clearSuggestions()
			return
		case fyne.KeyReturn, fyne.KeyEnter:
			e.accept(e.suggestions[e.selected])
			return
		}
	}
	e.Entry.TypedKey(key)
}
