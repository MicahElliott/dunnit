package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// newExplanatoryLabel creates the shared compact, wrapping treatment for
// instructional copy at the top of windows and alongside form sections.
// Keeping the font at the caption size prevents a long sentence from making
// its window wider than the intended layout.
func newExplanatoryLabel(text string) *widget.Label {
	label := widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	label.Wrapping = fyne.TextWrapWord
	label.SizeName = theme.SizeNameCaptionText
	return label
}

// newWindowHeading creates the matching bold heading used by standalone
// workflow windows that also have explanatory copy below it.
func newWindowHeading(text string) *widget.Label {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}
