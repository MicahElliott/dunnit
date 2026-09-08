package dun

import "fyne.io/fyne/v2"

// tightRowLayout lays out objects left-to-right with zero gap between
// them (unlike container.NewHBox, which inserts theme.Padding()
// between siblings) -- used by itemTextLabel (itemrow.go) so split
// text/tag canvas.Text runs render as one continuous flowing line of
// text rather than showing a visible extra gap at each run boundary.
type tightRowLayout struct{}

func newTightRowLayout() fyne.Layout {
	return tightRowLayout{}
}

func (tightRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	x := float32(0)
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		w := o.MinSize().Width
		o.Move(fyne.NewPos(x, 0))
		o.Resize(fyne.NewSize(w, size.Height))
		x += w
	}
}

func (tightRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var totalW, maxH float32
	for _, o := range objects {
		if !o.Visible() {
			continue
		}
		m := o.MinSize()
		totalW += m.Width
		if m.Height > maxH {
			maxH = m.Height
		}
	}
	return fyne.NewSize(totalW, maxH)
}
