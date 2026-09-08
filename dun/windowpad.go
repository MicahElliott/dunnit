package dun

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

// windowPad wraps content in fixed edge padding (10 on all 4 sides),
// independent of compactTheme's Size overrides (theme.go) -- those
// only affect spacing *between* sibling widgets, not the gap between
// the outermost content and the window frame. Every window's
// SetContent call should pass its top-level content through this
// (or already-equivalent padding) rather than setting raw/unpadded
// content directly -- see BuildMainWindow's original contentPad
// (ui.go), the first window this fix was applied to, for the fuller
// history of why an inner per-row padding fix (e.g. a single row's
// own CustomPaddedLayout) is NOT sufficient: it only visibly helps
// whichever child happens to sit flush against the window edge (e.g.
// the very first row in a VBox), not the other three edges or any
// other section. Factored out here (2026-09-03) after an audit found
// every other window in the app was still unpadded despite Daybook
// having been fixed already.
func windowPad(content fyne.CanvasObject) fyne.CanvasObject {
	return container.New(layout.NewCustomPaddedLayout(10, 10, 10, 10), content)
}
