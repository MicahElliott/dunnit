package dun

import (
	"image/color"
	"regexp"
	"strconv"
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
// row's text -- " @N[mhd]" (duration, ui.go's withMins) and
// " s/YYYY-MM-DD" (carry-forward annotation, carryforward.go) -- so this
// bookkeeping visually recedes behind the item's actual content.
var metaTextColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}

// metaTextSizeRatio shrinks the trailing metadata run's font size
// relative to the theme's normal text size, in addition to graying it
// out -- purely cosmetic, "de-emphasize further."
const metaTextSizeRatio = 0.85

// trailingMetaPattern matches one or more of the known trailing
// display-metadata suffixes back-to-back at the very end of an item's
// text: " @N[mhd]" (duration), " s/YYYY-MM-DD" (carry-forward), and
// lifecycle metadata. Matched as a repeating group so any
// combination/order of these (in practice at most one or two ever
// co-occur -- see splitTrailingMeta's doc comment) is captured as one
// contiguous trailing run.
var trailingMetaPattern = regexp.MustCompile(
	`(?:` +
		` @\d+[mhd]` +
		`| s/\d{4}-\d{2}-\d{2}` +
		`| \(since \d{4}-\d{2}-\d{2}\)` +
		`| \(via [A-Z_]+\)` +
		`)+$`)

var metadataTokenPattern = regexp.MustCompile(
	`(?:` +
		` @\d+[mhd]` +
		`| s/\d{4}-\d{2}-\d{2}` +
		`| \(since \d{4}-\d{2}-\d{2}\)` +
		`| \(via [A-Z_]+\)` +
		`)`)

var (
	durationMetadataPattern  = regexp.MustCompile(`^@(\d+)([mhd])$`)
	lifecycleMetadataPattern = regexp.MustCompile(`^\(via ([A-Z_]+)\)$`)
)

const (
	yellowAgeMaxDays = 3
	orangeAgeMaxDays = 7
)

// splitTrailingMeta splits text into (core, meta), where meta is the
// longest trailing run of known display-metadata suffixes (see
// trailingMetaPattern) and core is everything before it. meta is ""
// if text has no such trailing suffix. In practice a single row only
// ever carries one flavor of trailing metadata at a time (Planned
// rows show at most a since-date badge; Endings/Hilites rows show at most
// a mins suffix. Carry-forward's own "s/YYYY-MM-DD" is kept here so it
// can render as the age/date badge; the pattern handles any combination
// generically rather than assuming that stays true.
func splitTrailingMeta(text string) (core, meta string) {
	loc := trailingMetaPattern.FindStringIndex(text)
	if loc == nil {
		return text, ""
	}
	return text[:loc[0]], text[loc[0]:]
}

// stripDisplayMetadata removes the trailing bookkeeping that is useful in
// ledger-backed item lists but distracts from prose previews and summaries.
func stripDisplayMetadata(text string) string {
	core, _ := splitTrailingMeta(text)
	return strings.TrimRight(core, " \t")
}

// itemTextLabel renders text as a row of canvas.Text and clickable link
// runs: any trailing display-metadata suffix (see splitTrailingMeta) is
// peeled off and rendered smaller/grayed out; #tags are colored green and
// Markdown/bare URLs are rendered as small blue links. Used
// for item rows in Daybook's Planned/Endings/Hilites sections. Uses
// tightRowLayout (not container.NewHBox) so adjacent runs render
// flush against each other -- HBox's normal inter-child theme.Padding
// would otherwise show as a visible extra gap wherever a run
// boundary falls mid-word/without a real space (e.g. right before a
// tag).
func itemTextLabel(text string) fyne.CanvasObject {
	core, meta := splitTrailingMeta(text)

	var runs []fyne.CanvasObject
	links := parseEntryLinks(core)
	position := 0
	for _, link := range links {
		appendTextAndTags(&runs, core[position:link.Start])
		runs = append(runs, newURLLink(link.Text, link.URL))
		position = link.End
	}
	appendTextAndTags(&runs, core[position:])

	if meta != "" {
		for _, match := range metadataTokenPattern.FindAllStringIndex(meta, -1) {
			token := meta[match[0]:match[1]]
			label, tooltip := displayMetadataToken(token)
			if tooltip == "" {
				metaTxt := canvas.NewText(label, metaTextColor)
				metaTxt.TextSize = theme.TextSize() * metaTextSizeRatio
				runs = append(runs, metaTxt)
				continue
			}
			runs = append(runs, newHoverText(label, metaTextColor,
				theme.TextSize()*metaTextSizeRatio, tooltip))
		}
	}

	return container.New(newTightRowLayout(), runs...)
}

func displayMetadataToken(token string) (label, tooltip string) {
	trimmed := strings.TrimSpace(token)
	if match := durationMetadataPattern.FindStringSubmatch(trimmed); match != nil {
		label = " \u23f1" + match[1] + match[2]
		n, _ := strconv.Atoi(match[1])
		unit := map[byte]string{'m': "min", 'h': "hour", 'd': "day"}[match[2][0]]
		if n != 1 {
			unit += "s"
		}
		return label, "Spent " + match[1] + " " + unit
	}
	if since, ok := parseCarryForwardSince("item" + token); ok {
		days := daysSince(since)
		return " " + ageIndicator(days) + strconv.Itoa(days) + "d",
			"Open for " + strconv.Itoa(days) + " days"
	}
	if match := lifecycleMetadataPattern.FindStringSubmatch(trimmed); match != nil {
		return token, "Lifecycle source: " + match[1]
	}
	return token, ""
}

func ageIndicator(days int) string {
	switch {
	case days <= yellowAgeMaxDays:
		return "🟡"
	case days <= orangeAgeMaxDays:
		return "🟠"
	default:
		return "🔴"
	}
}

func appendTextAndTags(runs *[]fyne.CanvasObject, text string) {
	position := 0
	for _, match := range tagPattern.FindAllStringIndex(text, -1) {
		if match[0] > position {
			*runs = append(*runs, canvas.NewText(text[position:match[0]], theme.Color(theme.ColorNameForeground)))
		}
		*runs = append(*runs, canvas.NewText(text[match[0]:match[1]], tagTextColor))
		position = match[1]
	}
	if position < len(text) {
		*runs = append(*runs, canvas.NewText(text[position:], theme.Color(theme.ColorNameForeground)))
	}
}
