package dun

import (
	"hash/fnv"
	"image/color"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

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
// row's text -- " ~N[mhd]" (duration, ui.go's withMins) and
// " s/YYYY-MM-DD" (carry-forward annotation, carryforward.go) -- so this
// bookkeeping visually recedes behind the item's actual content.
var metaTextColor = color.NRGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}

// personTextColor distinguishes people markers from project/topic tags while
// keeping them readable in both ordinary rows and the Daybook prefix.
var personTextColor = color.NRGBA{R: 0x9a, G: 0x4f, B: 0x00, A: 0xff}

const personIcon = "👤\ufe0e"

const displayIconTextSizeRatio = 0.78

var ageIndicatorColors = map[string]color.NRGBA{
	"yellow": {R: 0xd0, G: 0x9b, B: 0x00, A: 0xff},
	"orange": {R: 0xc4, G: 0x6a, B: 0x00, A: 0xff},
	"red":    {R: 0xc0, G: 0x32, B: 0x32, A: 0xff},
}

// metaTextSizeRatio shrinks the trailing metadata run's font size
// relative to the theme's normal text size, in addition to graying it
// out. The slightly smaller size keeps emoji indicators aligned with
// their adjacent Latin text.
const metaTextSizeRatio = 0.80

// daybookMaxTextRunes keeps a long ledger entry from determining Daybook's
// minimum window width. The complete entry remains available from the
// hoverable ellipsis appended to truncated rows.
const daybookMaxTextRunes = 80

// trailingMetaPattern matches one or more of the known trailing
// display-metadata suffixes back-to-back at the very end of an item's
// text: " ~N[mhd]" (duration), " s/YYYY-MM-DD" (carry-forward), and
// lifecycle metadata. Matched as a repeating group so any
// combination/order of these (in practice at most one or two ever
// co-occur -- see splitTrailingMeta's doc comment) is captured as one
// contiguous trailing run.
var trailingMetaPattern = regexp.MustCompile(
	`(?:` +
		` ~\d+[mhd]` +
		`| s/\d{4}-\d{2}-\d{2}` +
		`| \(since \d{4}-\d{2}-\d{2}\)` +
		`| \(via [A-Z_]+\)` +
		`)+$`)

var metadataTokenPattern = regexp.MustCompile(
	`(?:` +
		` ~\d+[mhd]` +
		`| s/\d{4}-\d{2}-\d{2}` +
		`| \(since \d{4}-\d{2}-\d{2}\)` +
		`| \(via [A-Z_]+\)` +
		`)`)

var (
	durationMetadataPattern  = regexp.MustCompile(`^~(\d+)([mhd])$`)
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
// peeled off and rendered smaller/grayed out; #tags and @people use
// distinct trackable colors; Markdown/bare URLs are rendered as small blue
// links. Used
// for item rows in Daybook's Planned/Endings/Hilites sections. Uses
// tightRowLayout (not container.NewHBox) so adjacent runs render
// flush against each other -- HBox's normal inter-child theme.Padding
// would otherwise show as a visible extra gap wherever a run
// boundary falls mid-word/without a real space (e.g. right before a
// tag).
func itemTextLabel(text string) fyne.CanvasObject {
	core, meta := splitTrailingMeta(text)
	flags := extractFlags(core)
	core = stripFlags(core)

	var runs []fyne.CanvasObject
	appendFlagRuns(&runs, flags)
	links := parseEntryLinks(core)
	position := 0
	for _, link := range links {
		appendTextAndTrackables(&runs, core[position:link.Start])
		runs = append(runs, newURLLink(link.Text, link.URL, link.LocalPath))
		position = link.End
	}
	appendTextAndTrackables(&runs, core[position:])

	if meta != "" {
		for _, match := range metadataTokenPattern.FindAllStringIndex(meta, -1) {
			token := meta[match[0]:match[1]]
			appendMetadataToken(&runs, token)
		}
	}

	return container.New(newTightRowLayout(), runs...)
}

// daybookItemTextLabel displays a Daybook row with its primary tag in a
// grouping prefix and as a colored, hoverable "#" at the tag's original
// position. Long core text is shortened with a hoverable ellipsis; trailing
// metadata remains visible. The original text remains the value used for
// edits and ledger writes.
func daybookItemTextLabel(prefix, text string, stats map[string]*tagStat, onTagTap func(string)) fyne.CanvasObject {
	core, meta := splitTrailingMeta(text)
	flags := extractFlags(core)
	core = stripFlags(core)
	display := daybookCoreDisplay(core)

	runs := make([]fyne.CanvasObject, 0, 6)
	if prefix != "" {
		if icon, rest, ok := splitCategoryIconPrefix(prefix); ok {
			runs = append(runs, newDisplayIconText(icon, theme.Color(theme.ColorNameForeground)))
			if rest != "" {
				runs = append(runs, canvas.NewText(rest, theme.Color(theme.ColorNameForeground)))
			}
		} else {
			runs = append(runs, canvas.NewText(prefix, theme.Color(theme.ColorNameForeground)))
		}
	}
	if display.primaryTag != "" {
		runs = append(runs, newTagLinkWithStyle(
			"["+display.primaryTag+"] ", daybookTagTooltip(display.primaryTag, stats[display.primaryTag]),
			tagTextColor(display.primaryTag), true, func() { onTagTap(display.primaryTag) }))
	}
	appendFlagRuns(&runs, flags)
	appendDaybookCoreRuns(&runs, display, stats, onTagTap)
	appendMetadataRuns(&runs, meta)
	return container.New(newTightRowLayout(), runs...)
}

var flagTextColors = map[string]color.NRGBA{
	"!!": {R: 0xc0, G: 0x32, B: 0x32, A: 0xff},
	"??": {R: 0xa0, G: 0x72, B: 0x00, A: 0xff},
	"@@": {R: 0x6b, G: 0x3f, B: 0xa0, A: 0xff},
	"++": {R: 0x2e, G: 0x7d, B: 0x32, A: 0xff},
}

func appendFlagRuns(runs *[]fyne.CanvasObject, flags []string) {
	for _, code := range flags {
		flag, ok := flagDefinition(code)
		if !ok {
			continue
		}
		textColor := color.Color(metaTextColor)
		if specific, exists := flagTextColors[code]; exists {
			textColor = specific
		}
		*runs = append(*runs, newHoverTextWithStyle(
			" "+flag.Icon+" ",
			textColor,
			theme.TextSize()*0.84,
			fyne.TextStyle{Bold: true},
			flag.Label+" — "+flag.Help,
		))
	}
}

type daybookCoreRender struct {
	text          []rune
	primaryTag    string
	markerIndex   int
	ellipsisIndex int
	elidedText    string
}

// daybookCoreDisplay replaces the primary tag with a single marker and
// truncates only the prose portion. If the primary tag would be cut away, its
// marker is retained after the ellipsis so every row still communicates that a
// tag was present.
func daybookCoreDisplay(core string) daybookCoreRender {
	display := []rune(core)
	primaryTag := ""
	markerIndex := -1
	if matches := tagPattern.FindAllStringIndex(core, -1); len(matches) > 0 {
		match := matches[len(matches)-1]
		primaryTag = core[match[0]:match[1]]
		start := len([]rune(core[:match[0]]))
		end := len([]rune(core[:match[1]]))
		display = append(append(append([]rune{}, display[:start]...), '#'), display[end:]...)
		markerIndex = start
	}
	if len(display) <= daybookMaxTextRunes {
		return daybookCoreRender{
			text: display, primaryTag: primaryTag, markerIndex: markerIndex, ellipsisIndex: -1,
		}
	}

	keep := daybookMaxTextRunes - 1 // reserve one rune for the ellipsis
	if markerIndex >= keep {
		keep = daybookMaxTextRunes - 2 // reserve the ellipsis and the tag marker
	}
	visible := trimDaybookCut(display, keep)
	if markerIndex >= len(visible) {
		elided := append([]rune{}, display[len(visible):]...)
		markerOffset := markerIndex - len(visible)
		if markerOffset >= 0 && markerOffset < len(elided) {
			elided = append(elided[:markerOffset], elided[markerOffset+1:]...)
		}
		visible = append(visible, '\u2026')
		markerIndex = len(visible)
		visible = append(visible, '#')
		return daybookCoreRender{
			text:          visible,
			primaryTag:    primaryTag,
			markerIndex:   markerIndex,
			ellipsisIndex: markerIndex - 1,
			elidedText:    strings.TrimSpace(string(elided)),
		}
	}
	ellipsisIndex := len(visible)
	elidedText := strings.TrimSpace(string(display[len(visible):]))
	visible = append(visible, '\u2026')
	return daybookCoreRender{
		text:          visible,
		primaryTag:    primaryTag,
		markerIndex:   markerIndex,
		ellipsisIndex: ellipsisIndex,
		elidedText:    elidedText,
	}
}

func trimDaybookCut(text []rune, limit int) []rune {
	if limit >= len(text) {
		return text
	}
	cut := limit
	for cut > limit/2 && !unicode.IsSpace(text[cut-1]) {
		cut--
	}
	if cut <= limit/2 {
		cut = limit
	}
	for cut > 0 && unicode.IsSpace(text[cut-1]) {
		cut--
	}
	return text[:cut]
}

func appendDaybookCoreRuns(runs *[]fyne.CanvasObject, display daybookCoreRender, stats map[string]*tagStat, onTagTap func(string)) {
	positions := []int{display.markerIndex, display.ellipsisIndex}
	sort.Ints(positions)
	position := 0
	for _, special := range positions {
		if special < 0 || special >= len(display.text) || special < position {
			continue
		}
		appendDaybookEntryRuns(runs, string(display.text[position:special]), stats, onTagTap)
		switch special {
		case display.markerIndex:
			*runs = append(*runs, newTagLinkWithStyle(
				"#", daybookTagTooltip(display.primaryTag, stats[display.primaryTag]),
				tagTextColor(display.primaryTag), false, func() { onTagTap(display.primaryTag) }))
		case display.ellipsisIndex:
			*runs = append(*runs, newHoverTextWithStyle(" …",
				theme.Color(theme.ColorNameForeground), theme.TextSize(),
				fyne.TextStyle{Bold: true}, "Elided: "+display.elidedText))
		}
		position = special + 1
	}
	appendDaybookEntryRuns(runs, string(display.text[position:]), stats, onTagTap)
}

func daybookTagTooltip(tag string, stat *tagStat) string {
	tooltip := "Tag " + tag
	if usage := tagUsageTooltip(tag, stat); usage != "" {
		tooltip += " — " + usage
	}
	return tooltip
}

func appendEntryRuns(runs *[]fyne.CanvasObject, text string) {
	links := parseEntryLinks(text)
	position := 0
	for _, link := range links {
		appendTextAndTrackables(runs, text[position:link.Start])
		*runs = append(*runs, newURLLink(link.Text, link.URL, link.LocalPath))
		position = link.End
	}
	appendTextAndTrackables(runs, text[position:])
}

func appendDaybookEntryRuns(runs *[]fyne.CanvasObject, text string, stats map[string]*tagStat, onTagTap func(string)) {
	links := parseEntryLinks(text)
	position := 0
	for _, link := range links {
		appendDaybookTextAndTrackables(runs, text[position:link.Start], stats, onTagTap)
		*runs = append(*runs, newURLLink(link.Text, link.URL, link.LocalPath))
		position = link.End
	}
	appendDaybookTextAndTrackables(runs, text[position:], stats, onTagTap)
}

func appendMetadataRuns(runs *[]fyne.CanvasObject, meta string) {
	if meta == "" {
		return
	}
	for _, match := range metadataTokenPattern.FindAllStringIndex(meta, -1) {
		token := meta[match[0]:match[1]]
		appendMetadataToken(runs, token)
	}
}

func appendMetadataToken(runs *[]fyne.CanvasObject, token string) {
	textSize := theme.TextSize() * metaTextSizeRatio
	if since, ok := parseCarryForwardSince("item" + token); ok {
		days := daysSince(since)
		dot := canvas.NewText(" "+ageIndicator(days), ageIndicatorColor(days))
		dot.TextSize = textSize
		*runs = append(*runs, dot)
		*runs = append(*runs, newHoverText(strconv.Itoa(days)+"d", metaTextColor,
			textSize, "Open for "+strconv.Itoa(days)+" days"))
		return
	}
	label, tooltip := displayMetadataToken(token)
	if tooltip == "" {
		metaTxt := canvas.NewText(label, metaTextColor)
		metaTxt.TextSize = textSize
		*runs = append(*runs, metaTxt)
		return
	}
	*runs = append(*runs, newHoverText(label, metaTextColor, textSize, tooltip))
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
	return "●"
}

func ageIndicatorColor(days int) color.NRGBA {
	switch {
	case days <= yellowAgeMaxDays:
		return ageIndicatorColors["yellow"]
	case days <= orangeAgeMaxDays:
		return ageIndicatorColors["orange"]
	default:
		return ageIndicatorColors["red"]
	}
}

func appendTextAndTrackables(runs *[]fyne.CanvasObject, text string) {
	appendDaybookTextAndTrackables(runs, text, nil, nil)
}

func appendDaybookTextAndTrackables(runs *[]fyne.CanvasObject, text string, stats map[string]*tagStat, onTagTap func(string)) {
	if icon, rest, ok := splitCategoryIconPrefix(text); ok {
		*runs = append(*runs, newDisplayIconText(icon, theme.Color(theme.ColorNameForeground)))
		text = rest
	}

	type trackableMatch struct {
		start, end int
		color      color.Color
	}
	var matches []trackableMatch
	for _, match := range tagPattern.FindAllStringIndex(text, -1) {
		matches = append(matches, trackableMatch{match[0], match[1], tagTextColor(text[match[0]:match[1]])})
	}
	for _, match := range personPattern.FindAllStringIndex(text, -1) {
		if !isPersonMatchBoundary(text, match[0]) {
			continue
		}
		matches = append(matches, trackableMatch{match[0], match[1], personTextColor})
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].start < matches[j].start })
	position := 0
	for _, match := range matches {
		if match.start < position {
			continue
		}
		if match.start > position {
			*runs = append(*runs, canvas.NewText(text[position:match.start], theme.Color(theme.ColorNameForeground)))
		}
		if text[match.start] == '@' && match.color == personTextColor {
			*runs = append(*runs, newDisplayIconText(personIcon, personTextColor))
			*runs = append(*runs, canvas.NewText(text[match.start+1:match.end], match.color))
		} else if text[match.start] == '#' && onTagTap != nil {
			tag := text[match.start:match.end]
			tooltip := daybookTagTooltip(tag, stats[tag])
			*runs = append(*runs, newTagLinkWithStyle(tag, tooltip, match.color, false, func() {
				onTagTap(tag)
			}))
		} else {
			*runs = append(*runs, canvas.NewText(text[match.start:match.end], match.color))
		}
		position = match.end
	}
	if position < len(text) {
		*runs = append(*runs, canvas.NewText(text[position:], theme.Color(theme.ColorNameForeground)))
	}
}

func newDisplayIconText(text string, color color.Color) *canvas.Text {
	icon := canvas.NewText(text, color)
	icon.TextSize = theme.TextSize() * displayIconTextSizeRatio
	return icon
}

func splitCategoryIconPrefix(text string) (icon, rest string, ok bool) {
	for _, category := range Categories {
		prefix := category.Emoji + " "
		if strings.HasPrefix(text, prefix) {
			// Keep the separator with the rendered icon. The stored prefix
			// is consumed here, so dropping its trailing space would make
			// every categorized row visually run into its entry text.
			return prefix, text[len(prefix):], true
		}
	}
	return "", text, false
}
