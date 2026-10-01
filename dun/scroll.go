package dun

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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

// newMultiLineEntry keeps multiline editing while leaving mouse-wheel
// handling to the form or scroll container around the field. Fyne's default
// wrapping creates a private nested scroller even when Scroll is disabled,
// so both settings are needed for the surrounding scroll container to see
// the wheel. Explicit newlines are still preserved; long lines are clipped
// in the field, just like long values in newSingleLineEntry.
func newMultiLineEntry() *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.Wrapping = fyne.TextWrapOff
	e.Scroll = fyne.ScrollNone
	return e
}

// multiLineEntryField gives a no-scroll entry a useful empty height. Fyne's
// SetMinRowsVisible only affects its internal scroller, which these fields
// deliberately do not create.
func multiLineEntryField(e *widget.Entry, rows int) fyne.CanvasObject {
	if rows < 1 {
		rows = 1
	}
	lineHeight := fyne.MeasureText("M", theme.Size(theme.SizeNameText), e.TextStyle).Height
	minHeight := lineHeight*float32(rows) + theme.Size(theme.SizeNameInnerPadding)*2
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, minHeight))
	return container.NewMax(spacer, e)
}
