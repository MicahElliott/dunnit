package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// stretchMinWidth is the nominal minimum width contributed by the
// stretch object to the row's own MinSize -- deliberately small and
// fixed, NOT the stretch object's actual MinSize(). A widget.Entry's
// MinSize() grows with the length of its current text content, so
// using it directly here would make the whole row (and therefore the
// window, since Fyne sizes windows from content MinSize) grow/shrink
// based on what's currently typed into the entry -- the bug behind
// Daybook's width following the longest entry ever typed and
// spuriously showing a horizontal scrollbar. The stretch object is
// still given all remaining width at actual layout time (see
// Layout), just not counted by its content-dependent MinSize here.
const stretchMinWidth = 80

// stretchRowLayout lays out objects left-to-right (like
// container.NewHBox), except one designated object ("stretch") is
// given all the remaining width instead of just its MinSize -- every
// other object gets exactly its own MinSize width. Unlike
// container.NewBorder, this does NOT reorder objects in the
// container's Objects slice (NewBorder always appends its edge
// objects after its center/variadic ones, which silently breaks
// Fyne's Tab/Shift+Tab focus order -- see BuildMainWindow's comment
// on this). Objects here are laid out, and therefore focus-ordered,
// in exactly the slice order passed to container.New.
type stretchRowLayout struct {
	stretch fyne.CanvasObject
}

func newStretchRowLayout(stretch fyne.CanvasObject) fyne.Layout {
	return &stretchRowLayout{stretch: stretch}
}

func (s *stretchRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := theme.Padding()

	fixedWidth := float32(0)
	for _, o := range objects {
		if !o.Visible() || o == s.stretch {
			continue
		}
		fixedWidth += o.MinSize().Width + padding
	}
	stretchWidth := size.Width - fixedWidth
	if stretchWidth < 0 {
		stretchWidth = 0
	}

	x := float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		w := o.MinSize().Width
		if o == s.stretch {
			w = stretchWidth
		}
		o.Move(fyne.NewPos(x, 0))
		o.Resize(fyne.NewSize(w, size.Height))
		x += w + padding
	}
}

func (s *stretchRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var totalW, maxH float32
	first := true
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		w := m.Width
		if o == s.stretch {
			w = stretchMinWidth
		}
		totalW += w
		if !first {
			totalW += theme.Padding()
		}
		first = false
		if m.Height > maxH {
			maxH = m.Height
		}
	}
	return fyne.NewSize(totalW, maxH)
}
