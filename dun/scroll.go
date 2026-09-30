package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// newSingleLineEntry avoids Fyne's private nested scroller for ordinary
// one-line fields. Without this, a mouse wheel over an unfocused Entry is
// captured by that inner scroller and never reaches the surrounding form's
// VScroll. One-line fields should clip long values; keyboard navigation and
// the surrounding form remain usable.
func newSingleLineEntry() *widget.Entry {
	e := widget.NewEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Scroll = fyne.ScrollNone
	return e
}
