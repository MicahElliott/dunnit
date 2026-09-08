package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
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

// tooltipPopup wraps widget.PopUp to fix a real click-swallowing bug:
// Fyne's desktop driver only hit-tests against the topmost canvas
// overlay when one exists (see internal/driver/glfw/window.go's
// findObjectAtPositionMatching, which passes canvas.Overlays().Top()
// and does *not* fall through to the window's normal content when an
// overlay is present). Once this tooltip is shown (after hovering the
// owning button for hoverButtonTooltipDelay), any click -- including
// one landing squarely on the button itself -- gets routed to this
// popup instead of the button underneath. widget.PopUp's own Tapped
// just hides itself if the click was outside its own small content
// area, silently eating the click that was actually meant for the
// button (this is the root cause of Done/Postpone/Discard sometimes
// doing nothing: it only reproduces once the tooltip has had time to
// appear before the click). Fix: track the owning button's absolute
// bounds, and if a dismissing click falls within them, forward it to
// the button's own tap handler instead of just swallowing it.
type tooltipPopup struct {
	*widget.PopUp
	ownerPos    fyne.Position
	ownerSize   fyne.Size
	ownerTapped func()
}

func (t *tooltipPopup) Tapped(e *fyne.PointEvent) {
	within := e.AbsolutePosition.X >= t.ownerPos.X && e.AbsolutePosition.Y >= t.ownerPos.Y &&
		e.AbsolutePosition.X <= t.ownerPos.X+t.ownerSize.Width &&
		e.AbsolutePosition.Y <= t.ownerPos.Y+t.ownerSize.Height
	t.PopUp.Tapped(e) // still lets PopUp's own outside-click-dismiss logic run/hide as normal
	if within && t.ownerTapped != nil {
		t.ownerTapped()
	}
}

func (b *hoverButton) showTooltip() {
	if b.popup != nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(b)
	if canvas == nil {
		return
	}
	label := widget.NewLabel(b.tooltip)
	pop := &tooltipPopup{
		PopUp:       widget.NewPopUp(label, canvas),
		ownerPos:    fyne.CurrentApp().Driver().AbsolutePositionForObject(b),
		ownerSize:   b.Size(),
		ownerTapped: b.OnTapped,
	}
	b.popup = pop
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
	pop.ShowAtPosition(pos.Add(fyne.NewPos(0, b.Size().Height)))
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
