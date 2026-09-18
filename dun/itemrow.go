package dun

import (
	"hash/fnv"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// tagTextColors are semi-dark colors that remain readable on a white
// background. Blue is deliberately absent because blue is reserved for
// clickable links elsewhere in the UI.
var tagTextColors = []color.NRGBA{
	{R: 0x8b, G: 0x3a, B: 0x3a, A: 0xff}, // red
	{R: 0xa0, G: 0x52, B: 0x2d, A: 0xff}, // orange
	{R: 0x8b, G: 0x65, B: 0x08, A: 0xff}, // gold
	{R: 0x55, G: 0x6b, B: 0x2f, A: 0xff}, // olive
	{R: 0x2e, G: 0x7d, B: 0x32, A: 0xff}, // green
	{R: 0x00, G: 0x7f, B: 0x7f, A: 0xff}, // teal
	{R: 0x6b, G: 0x3f, B: 0xa0, A: 0xff}, // purple
	{R: 0x8b, G: 0x3a, B: 0x62, A: 0xff}, // magenta
	{R: 0x5d, G: 0x40, B: 0x37, A: 0xff}, // brown
	{R: 0x4f, G: 0x5f, B: 0x3f, A: 0xff}, // moss
}

// tagTextColor gives one deterministic color to each tag, so the same
// tag remains visually grouped even when it appears in several rows.
func tagTextColor(tag string) color.NRGBA {
	h := fnv.New32a()
	if _, err := h.Write([]byte(strings.ToLower(tag))); err != nil {
		return tagTextColors[0]
	}
	return tagTextColors[h.Sum32()%uint32(len(tagTextColors))]
}

// metaTextColor is a medium-light gray (not so light it's hard to
// read) used for trailing display-only metadata appended to an item
// row's text -- " @N[mhd]" (duration, ui.go's withMins) and
// " s/YYYY-MM-DD" (carry-forward annotation, carryforward.go) -- so this
// bookkeeping visually recedes behind the item's actual content.
var metaTextColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}

// metaTextSizeRatio shrinks the trailing metadata run's font size
// relative to the theme's normal text size, in addition to graying it
// out. The slightly smaller size keeps emoji indicators aligned with
// their adjacent Latin text.
const metaTextSizeRatio = 0.80

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
// peeled off and rendered smaller/grayed out; #tags use a deterministic
// per-tag color; Markdown/bare URLs are rendered as small blue links. Used
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

// daybookItemTextLabel displays a Daybook row with its last tag moved to
// an italic prefix. The supplied prefix is kept before the tag, usually
// the category icon. The original text remains the value used for edits
// and ledger writes; only this canvas representation is rearranged.
func daybookItemTextLabel(prefix, text string, stats map[string]*tagStat) fyne.CanvasObject {
	core, meta := splitTrailingMeta(text)
	tag, body := splitPrimaryTag(core)

	runs := make([]fyne.CanvasObject, 0, 4)
	if prefix != "" {
		runs = append(runs, canvas.NewText(prefix, theme.Color(theme.ColorNameForeground)))
	}
	if tag != "" {
		runs = append(runs, newTagLinkWithStyle(
			"["+tag+"] ", tagUsageTooltip(tag, stats[tag]), tagTextColor(tag), true, nil))
	}
	appendEntryRuns(&runs, body)
	appendMetadataRuns(&runs, meta)
	return container.New(newTightRowLayout(), runs...)
}

func appendEntryRuns(runs *[]fyne.CanvasObject, text string) {
	links := parseEntryLinks(text)
	position := 0
	for _, link := range links {
		appendTextAndTags(runs, text[position:link.Start])
		*runs = append(*runs, newURLLink(link.Text, link.URL))
		position = link.End
	}
	appendTextAndTags(runs, text[position:])
}

func appendMetadataRuns(runs *[]fyne.CanvasObject, meta string) {
	if meta == "" {
		return
	}
	for _, match := range metadataTokenPattern.FindAllStringIndex(meta, -1) {
		token := meta[match[0]:match[1]]
		label, tooltip := displayMetadataToken(token)
		if tooltip == "" {
			metaTxt := canvas.NewText(label, metaTextColor)
			metaTxt.TextSize = theme.TextSize() * metaTextSizeRatio
			*runs = append(*runs, metaTxt)
			continue
		}
		*runs = append(*runs, newHoverText(label, metaTextColor,
			theme.TextSize()*metaTextSizeRatio, tooltip))
	}
}

func displayMetadataToken(token string) (label, tooltip string) {
	trimmed := strings.TrimSpace(token)
	if match := durationMetadataPattern.FindStringSubmatch(trimmed); match != nil {
		// U+FE0E requests text presentation, avoiding the wider/taller
		// emoji presentation that makes the stopwatch look separated.
		label = " \u23f1\ufe0e" + match[1] + match[2]
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
		tag := text[match[0]:match[1]]
		*runs = append(*runs, canvas.NewText(tag, tagTextColor(tag)))
		position = match[1]
	}
	if position < len(text) {
		*runs = append(*runs, canvas.NewText(text[position:], theme.Color(theme.ColorNameForeground)))
	}
}
