package dun

import (
	"image/color"
	"net/url"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
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
	text    string
	tooltip string
	onTap   func()

	popup      *tooltipPopup
	hoverTimer *time.Timer
}

func newTagLink(text, tooltip string, onTap func()) *tagLink {
	t := &tagLink{text: text, tooltip: tooltip, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tagLink) CreateRenderer() fyne.WidgetRenderer {
	tag, count := splitTagCount(t.text)
	tagText := canvas.NewText(tag, tagLinkColor)
	if count == "" {
		return &tagLinkRenderer{tag: tagText}
	}
	countText := canvas.NewText(count, metaTextColor)
	countText.TextSize = theme.TextSize() * metaTextSizeRatio
	return &tagLinkRenderer{tag: tagText, count: countText}
}

func (t *tagLink) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tagLink) MouseIn(*desktop.MouseEvent) {
	t.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(t.showTooltip)
	})
}

func (t *tagLink) MouseMoved(*desktop.MouseEvent) {}

func (t *tagLink) MouseOut() { t.hideTooltip() }

func (t *tagLink) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

var _ fyne.Tappable = (*tagLink)(nil)
var _ desktop.Hoverable = (*tagLink)(nil)
var _ desktop.Cursorable = (*tagLink)(nil)

type tagLinkRenderer struct {
	tag   *canvas.Text
	count *canvas.Text
}

func (r *tagLinkRenderer) Layout(size fyne.Size) {
	r.tag.Resize(fyne.NewSize(r.tag.MinSize().Width, size.Height))
	if r.count != nil {
		r.count.Move(fyne.NewPos(r.tag.MinSize().Width, 0))
		r.count.Resize(fyne.NewSize(r.count.MinSize().Width, size.Height))
	}
}

func (r *tagLinkRenderer) MinSize() fyne.Size {
	tagSize := r.tag.MinSize()
	if r.count == nil {
		return tagSize
	}
	countSize := r.count.MinSize()
	return fyne.NewSize(tagSize.Width+countSize.Width, maxFloat32(tagSize.Height, countSize.Height))
}

func (r *tagLinkRenderer) Refresh() {
	canvas.Refresh(r.tag)
	if r.count != nil {
		canvas.Refresh(r.count)
	}
}

func (r *tagLinkRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.tag}
	if r.count != nil {
		objects = append(objects, r.count)
	}
	return objects
}

func (r *tagLinkRenderer) Destroy() {}

func splitTagCount(text string) (tag, count string) {
	idx := strings.LastIndex(text, "(")
	if idx <= 0 || !strings.HasSuffix(text, ")") {
		return text, ""
	}
	return text[:idx], text[idx:]
}

func maxFloat32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (t *tagLink) showTooltip() {
	if t.popup != nil || t.tooltip == "" || fyne.CurrentApp() == nil {
		return
	}
	host := fyne.CurrentApp().Driver().CanvasForObject(t)
	if host == nil {
		return
	}
	label := widget.NewLabel(t.tooltip)
	ownerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(t)
	popup := &tooltipPopup{
		host:        host,
		ownerPos:    ownerPos,
		ownerSize:   t.Size(),
		tooltipPos:  tooltipPositionAbove(ownerPos, label),
		label:       label,
		ownerTapped: func() { t.Tapped(nil) },
	}
	popup.ExtendBaseWidget(popup)
	popup.Resize(host.Size())
	t.popup = popup
	host.Overlays().Add(popup)
}

func (t *tagLink) hideTooltip() {
	if t.hoverTimer != nil {
		t.hoverTimer.Stop()
		t.hoverTimer = nil
	}
	if t.popup != nil {
		t.popup.Hide()
		t.popup = nil
	}
}

// urlLink is the compact clickable link used inside entry rows. It keeps the
// label small and blue, but deliberately leaves it un-underlined; the pointer
// cursor and tooltip provide the interaction cue without adding visual noise.
type urlLink struct {
	widget.BaseWidget
	text   string
	target *url.URL

	popup      *tooltipPopup
	hoverTimer *time.Timer
}

func newURLLink(text string, target *url.URL) *urlLink {
	link := &urlLink{text: text, target: target}
	link.ExtendBaseWidget(link)
	return link
}

func (l *urlLink) CreateRenderer() fyne.WidgetRenderer {
	txt := canvas.NewText(l.text, tagLinkColor)
	txt.TextSize = theme.TextSize() * 0.85
	return &urlLinkRenderer{txt: txt}
}

func (l *urlLink) Tapped(*fyne.PointEvent) {
	if l.target != nil && fyne.CurrentApp() != nil {
		if err := fyne.CurrentApp().OpenURL(l.target); err != nil {
			fyne.LogError("Failed to open entry URL", err)
		}
	}
}

func (l *urlLink) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (l *urlLink) MouseIn(*desktop.MouseEvent) {
	l.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(l.showTooltip)
	})
}

func (l *urlLink) MouseMoved(*desktop.MouseEvent) {}

func (l *urlLink) MouseOut() {
	if l.hoverTimer != nil {
		l.hoverTimer.Stop()
		l.hoverTimer = nil
	}
	if l.popup != nil {
		l.popup.Hide()
		l.popup = nil
	}
}

func (l *urlLink) showTooltip() {
	if l.popup != nil || fyne.CurrentApp() == nil {
		return
	}
	host := fyne.CurrentApp().Driver().CanvasForObject(l)
	if host == nil {
		return
	}
	label := widget.NewLabel(l.target.String())
	ownerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(l)
	popup := &tooltipPopup{
		host:        host,
		ownerPos:    ownerPos,
		ownerSize:   l.Size(),
		tooltipPos:  tooltipPositionAbove(ownerPos, label),
		label:       label,
		ownerTapped: func() { l.Tapped(nil) },
	}
	popup.ExtendBaseWidget(popup)
	popup.Resize(host.Size())
	l.popup = popup
	host.Overlays().Add(popup)
}

type urlLinkRenderer struct {
	txt *canvas.Text
}

func (r *urlLinkRenderer) Layout(size fyne.Size) { r.txt.Resize(size) }
func (r *urlLinkRenderer) MinSize() fyne.Size    { return r.txt.MinSize() }
func (r *urlLinkRenderer) Refresh()              { canvas.Refresh(r.txt) }
func (r *urlLinkRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.txt}
}
func (r *urlLinkRenderer) Destroy() {}

var _ fyne.Tappable = (*urlLink)(nil)
var _ desktop.Hoverable = (*urlLink)(nil)
var _ desktop.Cursorable = (*urlLink)(nil)
