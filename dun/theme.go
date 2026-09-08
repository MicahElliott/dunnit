package dun

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// compactTheme wraps another fyne.Theme (LightTheme, per
// BuildMainWindow's forced light-mode -- see eod.go's comment on why
// LightTheme is forced), only overriding the spacing sizes that drive
// "how spacey everything looks": Padding (the gap VBox/HBox/etc. put
// between sibling widgets -- this is what makes bulleted/checklist
// lists and stacked sections feel airy), InnerPadding (the padding
// widgets like Button put around their own content -- shrinking this
// is what "shrink buttons whenever possible" asks for, since a
// button's height is largely driven by this value), and LineSpacing
// (gap between wrapped text lines within a single Label/RichText).
// Text size and everything else (colors, icons, radii) are left at
// the wrapped theme's defaults -- this is purely a density tweak, not
// a full custom theme.
//
// IMPORTANT: this must be applied *after* (or instead of) any other
// a.Settings().SetTheme call, since SetTheme fully replaces whatever
// was set before -- BuildMainWindow's own
// a.Settings().SetTheme(theme.LightTheme()) previously silently
// clobbered this theme when it was set earlier in MakeUI, which was
// the actual reason two earlier tightening passes (2026-09-02)
// appeared to have no visible effect at all -- the compact theme was
// never actually active. Fixed by having newCompactTheme wrap
// LightTheme directly and having BuildMainWindow set *this* theme
// instead of a bare LightTheme.
//
// Default values (theme/size.go): Padding=4, InnerPadding=8,
// LineSpacing=4. Currently set to Padding=2, InnerPadding=4,
// LineSpacing=2 -- a lesser-ground middle point between the earlier
// floor values (1/2/1, which visually read as "things touching") and
// the defaults; still noticeably tighter than stock Fyne but with a
// little more breathing room between sibling widgets/lines. Window-
// edge spacing is handled separately (see contentPad in ui.go) since
// these Size overrides only affect *inter*-widget spacing, not the
// gap between the outermost content and the window frame.
type compactTheme struct {
	fyne.Theme
}

func newCompactTheme() fyne.Theme {
	return compactTheme{Theme: theme.LightTheme()}
}

func (compactTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 2
	case theme.SizeNameInnerPadding:
		return 4
	case theme.SizeNameLineSpacing:
		return 2
	default:
		return theme.LightTheme().Size(name)
	}
}

// Color darkens the idle button background (LightTheme's default,
// 0xf5f5f5, is nearly indistinguishable from the window background)
// and the hover overlay a bit further on top of that -- both requested
// together since hover is normally a subtle semi-transparent black
// wash over whatever the idle button color is; leaving hover at
// LightTheme's default alpha on top of a darker idle button would
// have made hover barely different from idle. Everything else falls
// through to LightTheme unchanged.
func (compactTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameButton:
		return color.NRGBA{R: 0xd8, G: 0xd8, B: 0xd8, A: 0xff}
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x30}
	default:
		return theme.LightTheme().Color(name, variant)
	}
}
