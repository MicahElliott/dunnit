package dun

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// tagLinkColor is the "looks clickable" blue used for #tag chips in
// the Frecent tags row (and anywhere else a tag is rendered as a
// clickable link).
var tagLinkColor = color.NRGBA{R: 0x1a, G: 0x73, B: 0xe8, A: 0xff}

// tagLink is a small tappable label rendering a #tag in blue,
// clicking it invokes onTap (e.g. inserting the tag into the main
// entry box). Modeled loosely on hoverButton, but simpler: no
// tooltip, just a colored canvas.Text wrapped as a widget so it can
// receive Tapped.
type tagLink struct {
	widget.BaseWidget
	text  string
	onTap func()
}

func newTagLink(text string, onTap func()) *tagLink {
	t := &tagLink{text: text, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tagLink) CreateRenderer() fyne.WidgetRenderer {
	txt := canvas.NewText(t.text, tagLinkColor)
	txt.TextStyle = fyne.TextStyle{Underline: true}
	return &tagLinkRenderer{txt: txt}
}

func (t *tagLink) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tagLink) MouseIn(*desktop.MouseEvent)    {}
func (t *tagLink) MouseMoved(*desktop.MouseEvent) {}
func (t *tagLink) MouseOut()                      {}

func (t *tagLink) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

var _ fyne.Tappable = (*tagLink)(nil)
var _ desktop.Hoverable = (*tagLink)(nil)
var _ desktop.Cursorable = (*tagLink)(nil)

type tagLinkRenderer struct {
	txt *canvas.Text
}

func (r *tagLinkRenderer) Layout(size fyne.Size) {
	r.txt.Resize(size)
}

func (r *tagLinkRenderer) MinSize() fyne.Size {
	return r.txt.MinSize()
}

func (r *tagLinkRenderer) Refresh() {
	canvas.Refresh(r.txt)
}

func (r *tagLinkRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.txt}
}

func (r *tagLinkRenderer) Destroy() {}
