package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// hoverButton is a widget.Button that also shows a small tooltip
// popup (its own fyne.CanvasObject, not a native OS tooltip -- Fyne
// has no built-in tooltip widget) after the mouse hovers over it for
// a short delay. Used where a button's label is just an emoji/icon
// and needs a text hint (e.g. Daybook's Discard/Postpone/Done
// actions).
type hoverButton struct {
	widget.Button
	tooltip string

	popup      *tooltipPopup
	hoverTimer *time.Timer
}

// newHoverButton creates a hoverButton with the given label (shown on
// the button itself, typically just an emoji) and tooltip text (shown
// after a brief hover).
func newHoverButton(label, tooltip string, tapped func()) *hoverButton {
	b := &hoverButton{tooltip: tooltip}
	b.Text = label
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// newHoverIconButton is newHoverButton's icon-based sibling, for
// actions with a good Fyne theme.IconName* match (e.g. Delete,
// Discard, Done) -- theme icons are consistent/theme-aware (respect
// light/dark, OS look) and preferred over emoji glyphs where a good
// conceptual match exists. Actions without a good match (e.g. Edit --
// Fyne's icon set has no pencil icon; Postpone -- no snooze/defer
// icon) should keep using newHoverButton's emoji instead.
func newHoverIconButton(icon fyne.Resource, tooltip string, tapped func()) *hoverButton {
	b := &hoverButton{tooltip: tooltip}
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

const hoverButtonTooltipDelay = 400 * time.Millisecond

// tooltipPopup is a full-canvas overlay, rather than a widget.PopUp. Fyne's
// overlay container receives clicks outside a PopUp's small content area and
// dismisses the popup without calling the popup content's Tapped method. A
// full-canvas tappable overlay lets us see those clicks, dismiss the tooltip,
// and forward clicks within the owner's bounds to the owner.
type tooltipPopup struct {
	widget.BaseWidget
	host             fyne.Canvas
	ownerPos         fyne.Position
	ownerSize        fyne.Size
	tooltipPos       fyne.Position
	label            *widget.Label
	ownerTapped      func()
	ownerTappedEvent func(*fyne.PointEvent)
}

func (t *tooltipPopup) Tapped(e *fyne.PointEvent) {
	within := t.containsOwner(e.AbsolutePosition)
	t.dismiss()
	if within {
		if t.ownerTappedEvent != nil {
			t.ownerTappedEvent(e)
		} else if t.ownerTapped != nil {
			t.ownerTapped()
		}
	}
}

func (t *tooltipPopup) containsOwner(pos fyne.Position) bool {
	return pos.X >= t.ownerPos.X && pos.Y >= t.ownerPos.Y &&
		pos.X <= t.ownerPos.X+t.ownerSize.Width &&
		pos.Y <= t.ownerPos.Y+t.ownerSize.Height
}

// Keep the tooltip alive while the pointer moves within its owner. Once the
// pointer leaves the owner, match the old hover behavior and dismiss it.
func (t *tooltipPopup) MouseIn(*desktop.MouseEvent) {}

func (t *tooltipPopup) MouseMoved(e *desktop.MouseEvent) {
	if !t.containsOwner(e.AbsolutePosition) {
		t.dismiss()
	}
}

func (t *tooltipPopup) MouseOut() {
	t.dismiss()
}

func (t *tooltipPopup) dismiss() {
	t.Hide()
}

func (t *tooltipPopup) Hide() {
	if t.host != nil {
		t.host.Overlays().Remove(t)
		t.host = nil
	}
	t.BaseWidget.Hide()
}

func (t *tooltipPopup) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	return &tooltipPopupRenderer{
		popup:      t,
		background: background,
		label:      t.label,
	}
}

type tooltipPopupRenderer struct {
	popup      *tooltipPopup
	background *canvas.Rectangle
	label      *widget.Label
}

func tooltipPositionAbove(ownerPos fyne.Position, label *widget.Label) fyne.Position {
	padding := theme.Padding()
	tooltipHeight := label.MinSize().Height + 2*padding
	// The tooltip is a full-canvas overlay. Starting at the owner's X
	// coordinate makes tooltips for right-edge controls render offscreen;
	// keep them anchored near the window's left edge instead.
	return fyne.NewPos(padding, ownerPos.Y-tooltipHeight)
}

func (r *tooltipPopupRenderer) Layout(_ fyne.Size) {
	padding := theme.Padding()
	labelSize := r.label.MinSize()
	backgroundSize := fyne.NewSize(labelSize.Width+2*padding, labelSize.Height+2*padding)
	r.background.Move(r.popup.tooltipPos)
	r.background.Resize(backgroundSize)
	r.label.Move(r.popup.tooltipPos.Add(fyne.NewPos(padding, padding)))
	r.label.Resize(labelSize)
}

func (r *tooltipPopupRenderer) MinSize() fyne.Size {
	return r.label.MinSize()
}

func (r *tooltipPopupRenderer) Refresh() {
	r.background.FillColor = theme.Color(theme.ColorNameMenuBackground)
	r.background.Refresh()
	r.label.Refresh()
}

func (r *tooltipPopupRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.label}
}

func (r *tooltipPopupRenderer) Destroy() {}

var _ fyne.Tappable = (*tooltipPopup)(nil)
var _ desktop.Hoverable = (*tooltipPopup)(nil)

func (b *hoverButton) showTooltip() {
	if b.popup != nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(b)
	if canvas == nil {
		return
	}
	label := widget.NewLabel(b.tooltip)
	ownerPos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
	pop := &tooltipPopup{
		host:        canvas,
		ownerPos:    ownerPos,
		ownerSize:   b.Size(),
		tooltipPos:  tooltipPositionAbove(ownerPos, label),
		label:       label,
		ownerTapped: b.OnTapped,
	}
	pop.ExtendBaseWidget(pop)
	pop.Resize(canvas.Size())
	b.popup = pop
	canvas.Overlays().Add(pop)
}

func (b *hoverButton) hideTooltip() {
	if b.hoverTimer != nil {
		b.hoverTimer.Stop()
		b.hoverTimer = nil
	}
	if b.popup != nil {
		b.popup.Hide()
		b.popup = nil
	}
}

// MouseIn/MouseMoved/MouseOut implement desktop.Hoverable.
func (b *hoverButton) MouseIn(*desktop.MouseEvent) {
	b.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(b.showTooltip)
	})
}

func (b *hoverButton) MouseMoved(*desktop.MouseEvent) {}

func (b *hoverButton) MouseOut() {
	b.hideTooltip()
}

var _ desktop.Hoverable = (*hoverButton)(nil)
