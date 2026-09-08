package dun

import (
	"image/color"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// tagTextColor is the dark green used to highlight #tag substrings
// inline within an item row's text (Planned/Endings/Hilites
// sections) -- same dark green showHelp already uses for positive-
// sentiment category rows, reused here for visual consistency rather
// than inventing a second "this text is notable" color.
var tagTextColor = color.NRGBA{R: 0, G: 100, B: 0, A: 255}

// metaTextColor is a medium-light gray (not so light it's hard to
// read) used for trailing display-only metadata appended to an item
// row's text -- " @Nm" (mins, ui.go's withMins), " (since ...)"
// (carry-forward annotation, carryforward.go), and " \u26a0 Nd" (the
// stale badge, also carryforward.go) -- so this bookkeeping visually
// recedes behind the item's actual content.
var metaTextColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}

// metaTextSizeRatio shrinks the trailing metadata run's font size
// relative to the theme's normal text size, in addition to graying it
// out -- purely cosmetic, "de-emphasize further."
const metaTextSizeRatio = 0.85

// trailingMetaPattern matches one or more of the known trailing
// display-metadata suffixes back-to-back at the very end of an item's
// text: " @Nm" (mins), " (since YYYY-MM-DD)" (carry-forward), and
// " \u26a0 Nd" (stale badge). Matched as a repeating group so any
// combination/order of these (in practice at most one or two ever
// co-occur -- see splitTrailingMeta's doc comment) is captured as one
// contiguous trailing run.
var trailingMetaPattern = regexp.MustCompile(
	`(?:` +
		` @\d+m` +
		`| \(since \d{4}-\d{2}-\d{2}\)` +
		`| \x{26a0} \d+d` +
		`)+$`)

// splitTrailingMeta splits text into (core, meta), where meta is the
// longest trailing run of known display-metadata suffixes (see
// trailingMetaPattern) and core is everything before it. meta is ""
// if text has no such trailing suffix. In practice a single row only
// ever carries one flavor of trailing metadata at a time (Planned
// rows show at most a stale badge; Endings/Hilites rows show at most
// a mins suffix -- carry-forward's own "(since ...)" is stripped
// before display via stripCarryForwardSince), but the pattern handles
// any combination generically rather than assuming that stays true.
func splitTrailingMeta(text string) (core, meta string) {
	loc := trailingMetaPattern.FindStringIndex(text)
	if loc == nil {
		return text, ""
	}
	return text[:loc[0]], text[loc[0]:]
}

// itemTextLabel renders text as a row of canvas.Text runs: any
// trailing display-metadata suffix (see splitTrailingMeta) is peeled
// off and rendered smaller/grayed out (metaTextColor/
// metaTextSizeRatio); within the remaining "core" text, every #tag
// substring (per extractTags/tagPattern, tags.go) is colored
// tagTextColor, everything else left in the theme's normal foreground
// color. Falls back to a single plain run if core has no tags. Used
// for item rows in Daybook's Planned/Endings/Hilites sections. Uses
// tightRowLayout (not container.NewHBox) so adjacent runs render
// flush against each other -- HBox's normal inter-child theme.Padding
// would otherwise show as a visible extra gap wherever a run
// boundary falls mid-word/without a real space (e.g. right before a
// tag).
func itemTextLabel(text string) fyne.CanvasObject {
	core, meta := splitTrailingMeta(text)

	var runs []fyne.CanvasObject
	tags := extractTags(core)
	if len(tags) == 0 {
		runs = append(runs, canvas.NewText(core, theme.Color(theme.ColorNameForeground)))
	} else {
		remaining := core
		for _, tag := range tags {
			idx := strings.Index(remaining, tag)
			if idx == -1 {
				continue // shouldn't happen, tag came from extractTags(core) itself
			}
			if before := remaining[:idx]; before != "" {
				runs = append(runs, canvas.NewText(before, theme.Color(theme.ColorNameForeground)))
			}
			runs = append(runs, canvas.NewText(tag, tagTextColor))
			remaining = remaining[idx+len(tag):]
		}
		if remaining != "" {
			runs = append(runs, canvas.NewText(remaining, theme.Color(theme.ColorNameForeground)))
		}
	}

	if meta != "" {
		metaTxt := canvas.NewText(meta, metaTextColor)
		metaTxt.TextSize = theme.TextSize() * metaTextSizeRatio
		runs = append(runs, metaTxt)
	}

	return container.New(newTightRowLayout(), runs...)
}
