package dun

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hoverText is a small, read-only text run with the same delayed tooltip
// behavior as hoverButton. It is used for compact metadata whose meaning is
// clearer on demand than in the row itself.
type hoverText struct {
	widget.BaseWidget
	text      string
	textColor color.Color
	textSize  float32
	tooltip   string

	popup      *tooltipPopup
	hoverTimer *time.Timer
}

func newHoverText(text string, textColor color.Color, textSize float32, tooltip string) *hoverText {
	h := &hoverText{text: text, textColor: textColor, textSize: textSize, tooltip: tooltip}
	h.ExtendBaseWidget(h)
	return h
}

func (h *hoverText) CreateRenderer() fyne.WidgetRenderer {
	txt := canvas.NewText(h.text, h.textColor)
	txt.TextSize = h.textSize
	return &hoverTextRenderer{txt: txt}
}

func (h *hoverText) Tapped(*fyne.PointEvent) {}

func (h *hoverText) showTooltip() {
	if h.popup != nil || h.tooltip == "" || fyne.CurrentApp() == nil {
		return
	}
	host := fyne.CurrentApp().Driver().CanvasForObject(h)
	if host == nil {
		return
	}
	label := widget.NewLabel(h.tooltip)
	ownerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h)
	popup := &tooltipPopup{
		host:       host,
		ownerPos:   ownerPos,
		ownerSize:  h.Size(),
		tooltipPos: tooltipPositionAbove(ownerPos, label),
		label:      label,
	}
	popup.ExtendBaseWidget(popup)
	popup.Resize(host.Size())
	h.popup = popup
	host.Overlays().Add(popup)
}

func (h *hoverText) hideTooltip() {
	if h.hoverTimer != nil {
		h.hoverTimer.Stop()
		h.hoverTimer = nil
	}
	if h.popup != nil {
		h.popup.Hide()
		h.popup = nil
	}
}

func (h *hoverText) MouseIn(*desktop.MouseEvent) {
	h.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(h.showTooltip)
	})
}

func (h *hoverText) MouseMoved(*desktop.MouseEvent) {}

func (h *hoverText) MouseOut() { h.hideTooltip() }

func (h *hoverText) Cursor() desktop.Cursor { return desktop.PointerCursor }

type hoverTextRenderer struct {
	txt *canvas.Text
}

func (r *hoverTextRenderer) Layout(size fyne.Size) { r.txt.Resize(size) }
func (r *hoverTextRenderer) MinSize() fyne.Size    { return r.txt.MinSize() }
func (r *hoverTextRenderer) Refresh()              { r.txt.Refresh() }
func (r *hoverTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.txt}
}
func (r *hoverTextRenderer) Destroy() {}

var _ fyne.Tappable = (*hoverText)(nil)
var _ desktop.Hoverable = (*hoverText)(nil)
var _ desktop.Cursorable = (*hoverText)(nil)
