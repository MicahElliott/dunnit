package dun

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// toastDuration is how long a flash message stays visible before
// auto-dismissing.
const toastDuration = 2 * time.Second

// showToast briefly flashes message near the bottom of the given
// canvas, auto-dismissing after toastDuration with no user action
// required -- used to confirm actions whose triggering button
// otherwise just silently disappears (e.g. Daybook's icon-only
// Discard/Postpone actions, and Edit Entry's Delete button), so the
// user gets some visible confirmation of what just happened. Not
// modal (uses plain widget.NewPopUp, not NewModalPopUp) -- it must
// not steal keyboard focus/input the way any canvas overlay can (see
// tagAutoEntry's doc comment on that exact hazard), so it's purely
// visual and never blocks interaction with anything underneath it.
func showToast(canvas fyne.Canvas, message string) {
	if canvas == nil {
		return
	}
	label := widget.NewLabel(message)
	pop := widget.NewPopUp(label, canvas)
	size := canvas.Size()
	popSize := pop.MinSize()
	pos := fyne.NewPos(
		(size.Width-popSize.Width)/2,
		size.Height-popSize.Height-24, // hover just above the bottom edge
	)
	pop.ShowAtPosition(pos)
	time.AfterFunc(toastDuration, func() {
		fyne.Do(pop.Hide)
	})
}
