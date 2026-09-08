package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// hoverSelect is a widget.Select that also shows a small hover
// tooltip (reusing tooltipPopup's click-forwarding fix, see
// hoverbutton.go's doc comment on why a plain widget.PopUp swallows
// clicks once shown) describing whatever option is currently
// selected. Used for Daybook's category picker so hovering shows the
// active category's Help text (Category.Help, categories.go) --
// reusing that same string keeps this in sync with the Help window's
// legend rather than duplicating the wording.
type hoverSelect struct {
	widget.Select
	tooltip func() string // returns current tooltip text; called lazily on hover

	popup      *tooltipPopup
	hoverTimer *time.Timer
}

// newHoverSelect creates a hoverSelect with the given options and
// selection callback (passed straight through to widget.NewSelect),
// plus a tooltip func called fresh each time a hover begins -- so it
// always reflects whatever is selected at hover time, not what was
// selected when the widget was built.
func newHoverSelect(options []string, changed func(string), tooltip func() string) *hoverSelect {
	s := &hoverSelect{tooltip: tooltip}
	s.Options = options
	s.OnChanged = changed
	s.ExtendBaseWidget(s)
	return s
}

func (s *hoverSelect) showTooltip() {
	if s.popup != nil {
		return
	}
	if s.tooltip == nil {
		return
	}
	text := s.tooltip()
	if text == "" {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(s)
	if canvas == nil {
		return
	}
	label := widget.NewLabel(text)
	pop := &tooltipPopup{
		PopUp:     widget.NewPopUp(label, canvas),
		ownerPos:  fyne.CurrentApp().Driver().AbsolutePositionForObject(s),
		ownerSize: s.Size(),
		// No ownerTapped forward here (unlike hoverButton): Select's
		// Tapped(*fyne.PointEvent) opens its dropdown menu and isn't
		// safe to call with a synthetic/nil event. A click that lands
		// on the Select while this tooltip is showing just dismisses
		// the tooltip (PopUp's normal outside-click behavior); a
		// second click then opens the dropdown as usual.
	}
	s.popup = pop
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(s)
	pop.ShowAtPosition(pos.Add(fyne.NewPos(0, s.Size().Height)))
}

func (s *hoverSelect) hideTooltip() {
	if s.hoverTimer != nil {
		s.hoverTimer.Stop()
		s.hoverTimer = nil
	}
	if s.popup != nil {
		s.popup.Hide()
		s.popup = nil
	}
}

// MouseIn/MouseMoved/MouseOut implement desktop.Hoverable.
func (s *hoverSelect) MouseIn(*desktop.MouseEvent) {
	s.hoverTimer = time.AfterFunc(hoverButtonTooltipDelay, func() {
		fyne.Do(s.showTooltip)
	})
}

func (s *hoverSelect) MouseMoved(*desktop.MouseEvent) {}

func (s *hoverSelect) MouseOut() {
	s.hideTooltip()
}

var _ desktop.Hoverable = (*hoverSelect)(nil)
