package dun

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"

	"bufio"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"strings"
)

// Get today's ledger file path and name.
func getLedger() (string, string) {
	return ledgerPathFor(time.Now())
}

// lastActivityAt tracks the wall-clock time of the most recent
// recordActivity() call, so the scheduler (sched.go) can suppress a
// periodic nudge if the user already logged something recently (see
// FR-01). Zero value means "nothing recorded since process start".
var lastActivityAt time.Time

// LastActivityAt returns the time of the most recent recorded entry
// (zero time if none yet this run).
func LastActivityAt() time.Time {
	return lastActivityAt
}

// mainInputEntry holds the Daybook window's main text-entry widget,
// captured once by BuildMainWindow, so FocusMainInput (called
// whenever Daybook is raised, e.g. by sched.go's nudges and the tray
// menu's "Show") can request keyboard focus land there directly,
// rather than wherever focus happened to be left (or nowhere).
var mainInputEntry *closeShortcutEntry

// prepareDaybookAutoPopup selects the category used by a scheduler-raised
// Daybook window. It is installed by BuildMainWindow once the picker exists.
var prepareDaybookAutoPopup func()

// prepareRecurringItemReminder fills Daybook for a timed recurring item. It
// is separate from prepareDaybookAutoPopup so a scheduled item can retain its
// own category and text instead of being reset to the normal DOING default.
var prepareRecurringItemReminder func(RecurringItem)

// daybookInputEmpty reports whether the Daybook entry controls contain any
// unsaved text. It is set by BuildMainWindow once both controls exist.
var daybookInputEmpty func() bool

const daybookAutoHideAfter = 2 * time.Minute

var (
	daybookAutoHideMu        sync.Mutex
	daybookAutoHideTimer     *time.Timer
	daybookAutoHideEnabled   bool
	daybookAutoHideFocusLost bool
)

// trayApp/trayWindow cache BuildMainWindow's fyne.App/main-window
// references so RebuildTrayMenu (called after Settings saves a
// changed Kickoff/Review toggle) can rebuild+reapply the tray menu
// without needing BuildMainWindow itself to be re-run. Set once by
// BuildMainWindow; nil until then (RebuildTrayMenu no-ops if so).
var trayApp fyne.App
var trayWindow fyne.Window

// trayRefreshAll refreshes Daybook's picker and open/completed/
// reflections/last-item sections -- set once by BuildMainWindow (which
// owns the actual refreshX closures, scoped to its own widget state),
// called by buildTrayMenu's "Show" item and after Settings saves. nil-
// checked before use since it isn't set until BuildMainWindow has run.
var trayRefreshAll func()

// refreshStartOfDayNotice updates the small Daybook reminder shown while
// today's Start of Day routine is still pending. It is set by
// BuildMainWindow and called when Start of Day runs, including when the
// Daybook window is already open.
var refreshStartOfDayNotice func()

// FocusMainInput requests keyboard focus on Daybook's main entry box,
// if it's been built yet. Safe to call even before BuildMainWindow
// has run (no-op).
func FocusMainInput() {
	if mainInputEntry == nil {
		return
	}
	fyne.Do(func() {
		if c := fyne.CurrentApp().Driver().CanvasForObject(mainInputEntry); c != nil {
			c.Focus(mainInputEntry)
		}
	})
}

func stopDaybookAutoHide() {
	daybookAutoHideMu.Lock()
	defer daybookAutoHideMu.Unlock()
	if daybookAutoHideTimer != nil {
		daybookAutoHideTimer.Stop()
		daybookAutoHideTimer = nil
	}
}

func daybookAutoHideState() (enabled, focusLost bool) {
	daybookAutoHideMu.Lock()
	defer daybookAutoHideMu.Unlock()
	return daybookAutoHideEnabled, daybookAutoHideFocusLost
}

func daybookEntryIsEmpty() bool {
	if daybookInputEmpty != nil {
		return daybookInputEmpty()
	}
	return mainInputEntry != nil && strings.TrimSpace(mainInputEntry.Text) == ""
}

func setDaybookAutoHideMode(enabled bool) {
	daybookAutoHideMu.Lock()
	defer daybookAutoHideMu.Unlock()
	daybookAutoHideEnabled = enabled
	daybookAutoHideFocusLost = false
	if daybookAutoHideTimer != nil {
		daybookAutoHideTimer.Stop()
		daybookAutoHideTimer = nil
	}
}

func daybookFocusGained() {
	daybookAutoHideMu.Lock()
	defer daybookAutoHideMu.Unlock()
	daybookAutoHideFocusLost = false
	if daybookAutoHideTimer != nil {
		daybookAutoHideTimer.Stop()
		daybookAutoHideTimer = nil
	}
}

func daybookFocusLost(w fyne.Window) {
	daybookAutoHideMu.Lock()
	daybookAutoHideFocusLost = true
	enabled := daybookAutoHideEnabled
	daybookAutoHideMu.Unlock()
	if enabled && daybookEntryIsEmpty() {
		armDaybookAutoHide(w)
	}
}

// hideDaybook hides the tray window and cancels any pending auto-hide timer.
func hideDaybook(w fyne.Window) {
	daybookAutoHideMu.Lock()
	daybookAutoHideEnabled = false
	daybookAutoHideFocusLost = false
	if daybookAutoHideTimer != nil {
		daybookAutoHideTimer.Stop()
		daybookAutoHideTimer = nil
	}
	daybookAutoHideMu.Unlock()
	w.Hide()
}

func armDaybookAutoHide(w fyne.Window) {
	daybookAutoHideMu.Lock()
	if daybookAutoHideTimer != nil {
		daybookAutoHideTimer.Stop()
	}
	daybookAutoHideTimer = time.AfterFunc(daybookAutoHideAfter, func() {
		fyne.Do(func() {
			daybookAutoHideMu.Lock()
			daybookAutoHideTimer = nil
			active := daybookAutoHideEnabled
			focusLost := daybookAutoHideFocusLost
			daybookAutoHideMu.Unlock()
			mainInputStillFocused := mainInputEntry != nil && w.Canvas().Focused() == mainInputEntry
			if active && focusLost && mainInputStillFocused && daybookEntryIsEmpty() {
				hideDaybook(w)
			}
		})
	})
	daybookAutoHideMu.Unlock()
}

// ShowDaybook raises the main window, refreshing its date-sensitive sections
// first. Scheduler nudges pass autoHide=true; tray/manual shows stay open.
func ShowDaybook(w fyne.Window, autoHide bool) {
	if trayRefreshAll != nil {
		trayRefreshAll()
	}
	if autoHide && prepareDaybookAutoPopup != nil {
		prepareDaybookAutoPopup()
	}
	setDaybookAutoHideMode(autoHide)
	w.Show()
	w.RequestFocus()
	FocusMainInput()
}

// ShowRecurringItemReminder raises Daybook with a configured recurring item
// ready to review and save, while retaining the normal auto-popup behavior.
func ShowRecurringItemReminder(w fyne.Window, item RecurringItem) {
	ShowDaybook(w, true)
	if prepareRecurringItemReminder != nil {
		prepareRecurringItemReminder(item)
	}
}

// snoozedUntil tracks a "not now, remind me later" request (FR-26) --
// while non-zero and in the future, the periodic capture nudge
// (sched.go) skips firing. Doesn't affect other nudges (SOD/EOD/
// meeting/etc), only the recurring "what are you working on?" one.
var snoozedUntil time.Time

// Snooze suppresses the next periodic capture nudge(s) until now+d
// (FR-26).
func Snooze(d time.Duration) {
	snoozedUntil = time.Now().Add(d)
}

// SnoozedUntil returns the current snooze expiry (zero if not
// snoozed / already expired).
func SnoozedUntil() time.Time {
	if time.Now().After(snoozedUntil) {
		return time.Time{}
	}
	return snoozedUntil
}

// IsDoNotDisturb reports whether the user has manually toggled Do Not
// Disturb on (FR-27) -- while true, the periodic capture nudge is
// suppressed entirely, same scope as Snooze.
func IsDoNotDisturb() bool {
	return LoadConfig().DoNotDisturb
}

// SetDoNotDisturb persists the Do Not Disturb flag (FR-27) to
// config.toml so it survives app restarts.
func SetDoNotDisturb(on bool) {
	cfg := LoadConfig()
	cfg.DoNotDisturb = on
	if err := writeConfig(cfg); err != nil {
		log.Println("Error saving Do Not Disturb setting:", err)
	}
}

// defaultSnoozeDuration returns the configured default snooze
// duration (cfg.SnoozeMinutes), falling back to 15 minutes if unset/
// invalid.
func defaultSnoozeDuration() time.Duration {
	minutes := LoadConfig().SnoozeMinutes
	if minutes <= 0 {
		minutes = 15
	}
	return time.Duration(minutes) * time.Minute
}

// RecordActivity is recordActivity's exported form, for callers
// outside this package (currently just cmd/dunnit's CLI, see
// docs/cli-design.md or the CLI's own doc comment) that want to
// append a ledger entry the same way Daybook's Save button does --
// same tag-cache invalidation and ledger-index invalidation. Does NOT validate that category is a
// real Category code; callers should check that themselves (see
// CategoryExists in categories.go) before calling. It returns any
// filesystem error so command-line callers can report failed writes.
func RecordActivity(text, category string) error {
	return recordActivity(text, category)
}

func recordActivity(text, category string) error {
	text = normalizeLedgerText(text)
	log.Println("Content was:", text)
	fpath, fname := getLedger()
	if err := os.MkdirAll(fpath, 0755); err != nil {
		err = fmt.Errorf("make ledger directory %q: %w", fpath, err)
		log.Println("Error making ledger dir:", err)
		return err
	}
	f, err := os.OpenFile(fname, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		err = fmt.Errorf("open ledger %q: %w", fname, err)
		log.Println("Error opening ledger:", err)
		return err
	}
	stamp := time.Now().Format("[15:04:05]")
	outstr := stamp + " " + category + " " + text + "\n"
	if _, err := f.WriteString(outstr); err != nil {
		err = fmt.Errorf("write ledger %q: %w", fname, err)
		log.Println("Error writing ledger:", err)
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		err = fmt.Errorf("close ledger %q: %w", fname, err)
		log.Println("Error closing ledger:", err)
		return err
	}
	lastActivityAt = time.Now()
	if len(extractTags(text)) > 0 {
		InvalidateTagCache()
	}
	InvalidateLedgerIndex()
	return nil
}

func normalizeLedgerText(text string) string {
	text = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(text)
	return strings.TrimSpace(text)
}

// readLedgerLines returns all lines from today's ledger file (empty if
// none exist yet).
func readLedgerLines() []string {
	_, fname := getLedger()
	return readLedgerLinesFrom(fname)
}

// readLedgerLinesFrom returns all lines from the given ledger file
// path (empty if it doesn't exist). Factored out of readLedgerLines
// so callers needing a specific day's file (e.g. FR-17's standup
// export, which wants the last workday's ledger rather than today's)
// can reuse the same scan logic.
func readLedgerLinesFrom(fname string) []string {
	f, err := os.Open(fname)
	if err != nil {
		return nil
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// lastEntryText returns the free-text portion (after timestamp+category)
// of the most recent ledger line, or "" if there isn't one.
func lastEntryText() string {
	lines := readLedgerLines()
	if len(lines) == 0 {
		return ""
	}
	last := lines[len(lines)-1]
	parts := strings.SplitN(last, " ", 3)
	if len(parts) < 3 {
		return last
	}
	return parts[2]
}

// openInEditor opens the given file with $EDITOR, falling back to the
// OS default opener (macOS `open`, Linux `xdg-open`). $EDITOR may
// include flags (e.g. "emacsclient -c"), so it's split on whitespace.
func openInEditor(path string) {
	if editor := os.Getenv("EDITOR"); editor != "" {
		fields := strings.Fields(editor)
		args := append(fields[1:], path)
		cmd := exec.Command(fields[0], args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			log.Println("Error launching $EDITOR:", err)
		}
		return
	}
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	if err := exec.Command(opener, path).Start(); err != nil {
		log.Println("Error opening file:", err)
	}
}

// visualLabelWidth returns a Category Label()'s rendered-width-in-
// characters, for column-alignment purposes (showHelp) -- rune count
// alone overcounts emoji sequences that include an invisible Unicode
// variation selector (U+FE0F, "present this as emoji-style"), which
// several category emoji use (e.g. "✔️" DONE, "🗑️" WASTED, "⚠️" RISK,
// "🕰️" SOMEDAY, "🏎️" OPTIMIZE all end in one) -- those labels were
// rendering one column short in showHelp's alignment before this fix,
// since len([]rune(...)) counted the selector as an extra character
// that takes up no visible width.
func visualLabelWidth(label string) int {
	width := 0
	for _, r := range label {
		if r == '\uFE0F' { // VARIATION SELECTOR-16, zero visual width
			continue
		}
		width++
	}
	return width
}

// showHelp opens a static window listing every category's emoji/code
// and one-line intended-use description (FR-06), grouped by Now/Plan/
// Reflect with section headers. Text comes directly from Categories
// (categories.go), so it can't drift out of sync with the actual
// picker options. Positive-sentiment categories render bold dark
// green; negative-sentiment ones render dark red. Named "Help" (not
// "Category Legend") in the UI -- broader framing for a window that
// may grow beyond just categories later. EODOnly categories (SUMMARY/
// PRODUCTIVITY/MEETING_HOURS) are excluded entirely -- they're always
// machine-written bookkeeping from eod.go's Finalize Day flow, never
// hand-picked, so they'd only clutter a legend meant to help someone
// choose a category from the live picker.
//
// Note: this coloring only applies to the static Help window -- Fyne's
// widget.Select doesn't support per-option rich text/color in its
// dropdown, so the live category picker itself stays plain text.
func showHelp(a fyne.App) {
	darkGreen := color.NRGBA{R: 0, G: 100, B: 0, A: 255}
	darkRed := color.NRGBA{R: 139, G: 0, B: 0, A: 255}

	// labelColWidth is the fixed column width (in characters, since
	// labels render Monospace) each category's Label() is padded out
	// to before appending its Help text -- without this, Help text
	// starts at a different column per row (labels vary quite a bit
	// in length, e.g. "✔️ DONE" vs "🏁 MILESTONE"), making the whole
	// legend look jagged. Computed as the longest non-EODOnly
	// visualLabelWidth + 1, so it stays correct as categories are
	// added/renamed rather than needing a hand-picked constant kept
	// in sync.
	labelColWidth := 0
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if n := visualLabelWidth(c.Label()); n > labelColWidth {
			labelColWidth = n
		}
	}
	labelColWidth++

	w := a.NewWindow("Dunnit: Help")
	rows := container.NewVBox()
	lastGroup := ""
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if c.Group != lastGroup {
			header := canvas.NewText(GroupLabel(c.Group), theme.Color(theme.ColorNameForeground))
			header.TextSize = 12
			header.TextStyle = fyne.TextStyle{Bold: true}
			rows.Add(header)
			lastGroup = c.Group
		}
		label := c.Label()
		if pad := labelColWidth - visualLabelWidth(label); pad > 0 {
			label += strings.Repeat(" ", pad)
		}
		txt := canvas.NewText(label+"\u2014 "+c.Help, theme.Color(theme.ColorNameForeground))
		txt.TextSize = 10
		txt.TextStyle = fyne.TextStyle{Monospace: true}
		switch c.Sentiment {
		case "positive":
			txt.Color = darkGreen
			txt.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
		case "negative":
			txt.Color = darkRed
		}
		rows.Add(txt)
	}
	scroll := container.NewVScroll(rows)
	scroll.SetMinSize(fyne.NewSize(560, 340))
	w.SetContent(windowPad(scroll))
	w.Show()
}

func MakeUI() *fyne.App {
	a := app.New()
	// Theme is set in BuildMainWindow (which also needs LightTheme's
	// color scheme, see eod.go's comment) rather than here -- setting
	// it in both places is harmless (SetTheme just replaces), but
	// BuildMainWindow is the actual single point of truth so it's
	// only done there to avoid the two calls silently drifting out of
	// sync the way they did before (see compactTheme's doc comment).
	return &a
}

// closeShortcutEntry is a widget.Entry that additionally recognizes
// the given close shortcut (Cmd+W/Ctrl+W) even while it has focus.
// Fyne's Entry widget has its own ShortcutHandler and normally
// swallows all TypedShortcut calls when focused, so a shortcut added
// only via Canvas().AddShortcut never reaches the window-level
// handler in that case (FR-02). We check for our specific shortcut
// first and invoke onClose directly; anything else falls through to
// the embedded Entry's normal shortcut handling (cut/copy/paste etc).
type closeShortcutEntry struct {
	tagAutoEntry
	closeKey      fyne.KeyName
	closeMod      fyne.KeyModifier
	onClose       func()
	onFocusGained func()
	onFocusLost   func()
}

func newCloseShortcutEntry(closeKey fyne.KeyName, closeMod fyne.KeyModifier, onClose func()) *closeShortcutEntry {
	e := &closeShortcutEntry{closeKey: closeKey, closeMod: closeMod, onClose: onClose}
	e.ExtendBaseWidget(e)
	e.initTagAutoEntry()
	return e
}

func (e *closeShortcutEntry) TypedShortcut(shortcut fyne.Shortcut) {
	if cs, ok := shortcut.(*desktop.CustomShortcut); ok &&
		cs.KeyName == e.closeKey && cs.Modifier == e.closeMod {
		e.onClose()
		return
	}
	e.tagAutoEntry.TypedShortcut(shortcut)
}

func (e *closeShortcutEntry) FocusGained() {
	e.tagAutoEntry.FocusGained()
	if e.onFocusGained != nil {
		e.onFocusGained()
	}
}

func (e *closeShortcutEntry) FocusLost() {
	e.tagAutoEntry.FocusLost()
	if e.onFocusLost != nil {
		e.onFocusLost()
	}
}

// BuildMainWindow constructs the main Dunnit entry window and tray menu,
// but does not show it or start the Fyne event loop -- call a.Run()
// yourself after this (see dun.go). Returns the window so callers
// (e.g. the scheduler) can Show()/RequestFocus() it later.
func BuildMainWindow(a fyne.App) fyne.Window {
	a.Settings().SetTheme(newCompactTheme())
	cfg := LoadConfig()

	w4 := a.NewWindow("Dunnit: Daybook")
	// label1 := widget.NewLabel("Label 1")
	// value1 := widget.NewLabel("Value")
	// label2 := widget.NewLabel("Label 2")
	// value2 := widget.NewLabel("Something")

	// TODO show day's GOALs

	input := newCloseShortcutEntry(fyne.KeyW, fyne.KeyModifierShortcutDefault, func() { hideDaybook(w4) })
	input.SetPlaceHolder("Enter text\u2026")
	mainInputEntry = input
	input.onFocusGained = daybookFocusGained
	input.onFocusLost = func() { daybookFocusLost(w4) }
	previousInputChanged := input.Entry.OnChanged
	input.Entry.OnChanged = func(text string) {
		previousInputChanged(text)
		enabled, focusLost := daybookAutoHideState()
		if !enabled {
			return
		}
		if strings.TrimSpace(text) == "" {
			if focusLost {
				armDaybookAutoHide(w4)
			}
		} else {
			stopDaybookAutoHide()
		}
	}
	// input.Resize(fyne.NewSize(100.0, 50.0))

	// Tag autocomplete (FR-10): as the user types a "#tag" fragment,
	// show suggestions of matching previously-used tags (scanned from
	// ledger history, cached -- see tags.go). Selecting an entry
	// (via Up/Down + Enter, or a mouse click) replaces the in-progress
	// fragment with the full tag.
	//
	// This is `input`'s built-in tagAutoEntry behavior (tagautoentry.go)
	// -- see that file's doc comment for why the previous two attempts
	// (widget.PopUpMenu, then plain widget.PopUp) were both fundamentally
	// broken: ANY canvas overlay steals all keyboard routing away from
	// `input` the moment it exists, regardless of explicit refocus
	// calls, which is why typing past the first letter -- and even
	// Escape -- stopped working. inputSuggestions is added to the
	// surrounding layout, immediately below input's row, as a plain
	// sibling widget (no overlay).
	inputSuggestions := input.SuggestionBox()

	startOfDayNotice := container.NewVBox()
	refreshStartOfDayNotice = func() {
		startOfDayNotice.RemoveAll()
		if startOfDayPending(LoadConfig(), time.Now()) {
			notice := newExplanatoryLabel("Start of Day hasn’t run yet today; run it to bring forward open items.")
			startOfDayNotice.Add(container.New(newStretchRowLayout(notice), notice,
				widget.NewButton("Start of Day…", func() { showSODWindow(a) })))
		}
		startOfDayNotice.Refresh()
	}
	refreshStartOfDayNotice()

	// minsInput is an optional free-text "minutes spent" field (very
	// informal time tracking). When non-empty and numeric, its value
	// is appended to the recorded text as " @Nm" (e.g. "@20m").
	// Wrapped in a fixed-size container (minsWrapper) so it renders
	// at a comfortable width regardless of its own placeholder-driven
	// MinSize -- stretchRowLayout below treats it as a fixed-width
	// object (like groupFilter/category), giving all remaining space
	// to `input` instead.
	minsInput := widget.NewEntry()
	minsInput.SetPlaceHolder("mins")
	minsWrapper := container.NewGridWrap(fyne.NewSize(64, minsInput.MinSize().Height), minsInput)
	daybookInputEmpty = func() bool {
		return strings.TrimSpace(input.Text) == "" && strings.TrimSpace(minsInput.Text) == ""
	}
	minsInput.OnChanged = func(text string) {
		enabled, focusLost := daybookAutoHideState()
		if !enabled {
			return
		}
		if strings.TrimSpace(text) == "" {
			if focusLost && daybookEntryIsEmpty() {
				armDaybookAutoHide(w4)
			}
		} else {
			stopDaybookAutoHide()
		}
	}

	// setMinsWrapperVisibility shows/hides the mins field based on
	// whether the currently selected category is time-trackable (see
	// IsTimeTrackable/timeTrackableCategories, categories.go) --
	// added 2026-09-03 so the mins box only ever appears for DONE/
	// FAIL/WASTED (completed-effort entries), never for "Plan"-group
	// items like TODO/GOAL where a mins value would misleadingly read
	// as a time *estimate* rather than an actual duration.
	setMinsWrapperVisibility := func(cat string) {
		if IsTimeTrackable(cat) {
			minsWrapper.Show()
		} else {
			minsInput.SetText("")
			minsWrapper.Hide()
		}
	}

	selectedCat := "DONE"

	// withMins appends " @Nm" to text if minsInput has a valid
	// non-negative integer in it; otherwise returns text unchanged.
	withMins := func(text string) string {
		if !IsTimeTrackable(selectedCat) {
			return text
		}
		raw := strings.TrimSpace(minsInput.Text)
		if raw == "" {
			return text
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return text
		}
		return text + " @" + raw + "m"
	}

	// widget.NewSelectEntry
	//
	// Note: Fyne's widget.Select renders its selected-text label via
	// an internal RichText segment that it resets (alignment, color)
	// on every refresh, with no exposed hook to set TextStyle.
	// Monospace -- so unlike the Upcoming list and Category Legend
	// (both of which use canvas.Text/widget.Label directly, which we
	// do control), this live picker can't be made monospace without
	// forking/reimplementing Select's renderer. Not worth that cost
	// for a cosmetic detail; left in the default font.
	//
	// defaultGroupFilter/defaultGroupOptions: if cfg.FavoriteCategories
	// is non-empty, "Faves" is the picker's default-active filter
	// (replacing "whatever group was last used" -- Micah wants his
	// chosen bucket active every time Daybook pops up, not sticky
	// last-used state) and is added to the groupFilter dropdown's
	// options; otherwise behaves exactly as before (defaults to
	// "End", no Faves option shown at all).
	faves := CategoryLabelsForFaves(cfg)
	defaultGroupFilter := "End"
	defaultGroupOptions := []string{"End", "Plan", "Hilite"}
	defaultCategoryOptions := CategoryLabelsForGroup(cfg, "end")
	if len(faves) > 0 {
		defaultGroupFilter = "Faves"
		defaultGroupOptions = []string{"Faves", "End", "Plan", "Hilite"}
		defaultCategoryOptions = faves
	}

	// category is a hoverSelect (not a plain widget.Select) so
	// hovering it shows the currently-selected category's Help text
	// (Category.Help, categories.go) as a tooltip -- reusing the same
	// string shown in the Help window's legend, so the two can never
	// drift out of sync.
	category := newHoverSelect(defaultCategoryOptions,
		func(cat string) {
			fmt.Println("saw a category:", cat)
			res := strings.Split(cat, " ")
			newCat := res[1]
			if newCat != selectedCat {
				minsInput.SetText("")
			}
			selectedCat = newCat
			setMinsWrapperVisibility(selectedCat)
		},
		func() string { return HelpForCode(selectedCat) })
	category.SetSelected(defaultCategoryOptions[0])

	// groupFilter narrows the category picker to a subset (FR-06
	// follow-up): "Faves" (user-configured, see
	// Config.FavoriteCategories -- shown/default only if configured),
	// "end" (day-to-day capture), "plan" (future-facing), or "hilite"
	// (notable-moment callouts). Purely a UI convenience over
	// the same Categories list. (An "All" option existed here
	// previously but was removed -- Micah didn't see a strong need
	// for it given Faves/End/Plan/Hilite already cover the picker's
	// practical use cases; CategoryLabelsForGroup itself still
	// supports "" / "all" as "every category" for any other caller
	// that wants it.)
	groupFilter := widget.NewSelect(defaultGroupOptions,
		func(g string) {
			cfg := LoadConfig()
			if g == "Faves" {
				faves = CategoryLabelsForFaves(cfg)
				category.Options = faves
			} else {
				category.Options = CategoryLabelsForGroup(cfg, strings.ToLower(g))
			}
			category.SetSelected(category.Options[0])
			category.Refresh()
		})
	groupFilter.SetSelected(defaultGroupFilter)

	// selectCategoryCode is used by timed recurring-item reminders. If the
	// selected category is outside the current quick-filter, switch to its
	// group first so the category can still be selected visibly.
	selectCategoryCode := func(code string) {
		minsInput.SetText("")
		var label string
		for _, c := range Categories {
			if c.Code == code {
				label = c.Label()
				break
			}
		}
		if label == "" {
			return
		}
		containsLabel := func(options []string) bool {
			for _, option := range options {
				if option == label {
					return true
				}
			}
			return false
		}
		if !containsLabel(category.Options) {
			switch GroupForCode(code) {
			case "end":
				groupFilter.SetSelected("End")
			case "hilite":
				groupFilter.SetSelected("Hilite")
			default:
				groupFilter.SetSelected("Plan")
			}
		}
		category.SetSelected(label)
		selectedCat = code
		setMinsWrapperVisibility(code)
	}
	prepareDaybookAutoPopup = func() {
		selectCategoryCode("DOING")
	}
	prepareRecurringItemReminder = func(item RecurringItem) {
		selectCategoryCode(item.Category)
		input.SetText(item.Text)
	}

	// refreshCategoryPicker re-reads the favorites and category filters
	// into the already-open Daybook. Settings used to rebuild only the
	// tray menu, leaving this picker with the Config snapshot captured
	// when BuildMainWindow ran until the next full app restart.
	var refreshCategoryPicker func()
	refreshCategoryPicker = func() {
		cfg := LoadConfig()
		faves = CategoryLabelsForFaves(cfg)
		groupOptions := []string{"End", "Plan", "Hilite"}
		selectedGroup := groupFilter.Selected
		if len(faves) > 0 {
			groupOptions = []string{"Faves", "End", "Plan", "Hilite"}
		} else if selectedGroup == "Faves" {
			selectedGroup = "End"
		}
		groupFilter.Options = groupOptions
		groupFilter.Selected = selectedGroup
		if selectedGroup == "Faves" {
			category.Options = faves
		} else {
			category.Options = CategoryLabelsForGroup(cfg, strings.ToLower(selectedGroup))
		}
		if len(category.Options) == 0 {
			return
		}
		category.SetSelected(category.Options[0])
		groupFilter.Refresh()
		category.Refresh()
	}

	// Dunnit aims to be a mouseless/mouse-optional UI -- keyboard-only
	// operation should always be possible. IMPORTANT: Fyne's focus
	// traversal (Tab/Shift+Tab) follows each container's *Objects
	// slice order*, not visual/layout position. container.NewBorder's
	// constructor appends objects in the fixed order (center content
	// first, then top, bottom, left, right) regardless of which
	// visual position they occupy -- so a NewBorder-based row can
	// easily end up with Tab order that doesn't match what's on
	// screen. stretchRowLayout (stretchrow.go) is used instead: it
	// preserves the exact slice order passed to container.New (so Tab
	// order matches visual left-to-right order) while still letting
	// `input` stretch to fill available width like NewBorder's center
	// content would have.
	// saveBtn is created here (empty OnTapped, filled in once saveEntry
	// is defined further below -- it needs refreshOpenItems/
	// refreshCompleted/refreshReflections/refreshLastItem, which
	// aren't defined yet at this point) so it can sit in doneWrapper's
	// row (right of minsWrapper), matching Tab/visual order -- see
	// doneWrapper's comment below.
	saveBtn := widget.NewButton("Save", nil)

	// doneWrapper is the main entry row (groupFilter/category/input/
	// minsWrapper/saveBtn) -- Daybook's single most important row,
	// where virtually every interaction starts. Save sits at the end
	// of this row (not in a separate buttons row) since it's the most
	// important action and belongs right next to what it saves; this
	// also means Tab from input/minsInput naturally lands on Save
	// next (stretchRowLayout preserves Objects slice order for
	// Tab/Shift+Tab focus, matching visual left-to-right order -- see
	// its own comment above). Window-edge spacing is now handled
	// once, uniformly, by the outer contentPad wrap below (see its
	// comment) -- this local wrap only adds a bit of extra *bottom*
	// padding to visually separate this row from commonTagsRow
	// underneath it, since it's the primary piece of Daybook and
	// deserves to stand apart from the rest. Value is an arbitrary
	// "looks reasonable" pick, not theme-driven -- adjust here if it
	// looks off.
	doneWrapper := container.New(layout.NewCustomPaddedLayout(0, 6, 0, 0),
		container.New(newStretchRowLayout(input), groupFilter, category, input, minsWrapper, saveBtn))

	fmt.Println(input.MinSize())

	// openItemsBox displays currently-open item lines together under
	// one "Upcoming" heading (FR-07, extended for WAITING/QUESTION/
	// FIXME/RISK), split into per-category sub-sections, each with
	// "Done" (convert to DONE) and "Postpone" (defer to SOMEDAY,
	// keeping the list from growing unbounded) actions.
	// refreshOpenItems rebuilds it from the current ledger contents;
	// called after any save/convert/postpone action so the list stays
	// in sync. Wrapped in a widget.Accordion (itemsAccordion, below)
	// so it can be collapsed out of the way once reviewed.
	openItemsBox := container.NewVBox()
	var refreshOpenItems func()
	var refreshCompleted func()          // forward decl -- used inside refreshOpenItems's Done button, defined below
	var refreshLastItem func()           // forward decl -- used inside refreshOpenItems's Done button and saveEntry, defined further below (needs lastItemRow)
	var itemsAccordion *widget.Accordion // forward decl -- used inside refreshOpenItems's Done/Postpone/Discard buttons, defined below
	// showAllPlanned toggles whether Planned's non-TODO categories
	// (GOAL/WAITING/QUESTION/FIXME/RISK) are shown -- default false
	// (TODO/DOING-only) since the full set together was feeling
	// overwhelming; a "Show all" / "Show TODOs only" toggle button
	// reveals/hides the rest without losing them.
	showAllPlanned := false
	// showExcludedPlanned toggles whether items whose leading tag is
	// in Config.ReportExcludeTags are shown -- default false, same
	// "hidden behind a toggle rather than gone" pattern as
	// showAllPlanned, but for tag-based noise rather than category.
	showExcludedPlanned := false
	refreshOpenItems = func() {
		openItemsBox.RemoveAll()
		items := getOpenItems()
		if len(items) == 0 {
			openItemsBox.Add(widget.NewLabel("Nothing open right now."))
			openItemsBox.Refresh()
			return
		}
		excludeTags := LoadConfig().ReportExcludeTags
		addRow := func(item OpenItem) {
			// Planned's icon-only controls stay as plain Fyne buttons.
			// A hover tooltip is a full-canvas overlay in Fyne, so it can
			// take the first click while the button is unfocused.
			actions := []fyne.CanvasObject{
				widget.NewButtonWithIcon("", theme.Icon(theme.IconNameContentClear), func() {
					recordDiscarded(item)
					fyne.Do(func() {
						refreshOpenItems()
						itemsAccordion.Refresh()
						showToast(w4.Canvas(), "Discarded")
					})
				}),
				widget.NewButtonWithIcon("", theme.Icon(theme.IconNameHistory), func() {
					recordPostponed(item)
					fyne.Do(func() {
						refreshOpenItems()
						itemsAccordion.Refresh()
						showToast(w4.Canvas(), "Postponed (to SOMEDAY)")
					})
				}),
				widget.NewButtonWithIcon("", theme.Icon(theme.IconNameConfirm), func() {
					showEditItemDialogForCategory(w4, item, "DONE", func() {
						minsInput.SetText("")
						refreshOpenItems()
						refreshCompleted()
						refreshLastItem()
						itemsAccordion.Refresh()
						showToast(w4.Canvas(), "Completed")
					})
				}),
			}
			if item.Category == "TODO" {
				actions = append(actions, widget.NewButtonWithIcon("", theme.Icon(theme.IconNameMediaPlay), func() {
					if err := startPlannedItem(item); err != nil {
						log.Println("Error starting planned item:", err)
					}
					minsInput.SetText("")
					fyne.Do(func() {
						refreshOpenItems()
						itemsAccordion.Refresh()
						showToast(w4.Canvas(), "Started")
					})
				}))
			}
			actions = append(actions, widget.NewButtonWithIcon("", theme.Icon(theme.IconNameDocumentCreate), func() {
				showEditItemDialog(w4, item, func() {
					fyne.Do(func() {
						refreshOpenItems()
						refreshCompleted()
						refreshLastItem()
						itemsAccordion.Refresh()
					})
				})
			}))
			row := container.NewBorder(nil, nil, nil, container.NewHBox(actions...),
				itemTextLabel(categoryIconPrefix(item.Category)+openItemDisplayText(item.Text)))
			openItemsBox.Add(row)
		}
		cats, grouped := groupOpenItemsByCategory(items)
		otherCount := 0
		excludedCount := 0
		addCatRows := func(cat string) {
			openItemsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			visible, excluded := splitExcludedTagItems(grouped[cat], excludeTags)
			excludedCount += len(excluded)
			for _, item := range visible {
				addRow(item)
			}
			if showExcludedPlanned {
				for _, item := range excluded {
					addRow(item)
				}
			}
		}
		for _, cat := range cats {
			if cat != "TODO" && cat != "DOING" {
				otherCount += len(grouped[cat])
				continue
			}
			addCatRows(cat)
		}
		if otherCount > 0 {
			toggleLabel := fmt.Sprintf("Show all (%d more)", otherCount)
			if showAllPlanned {
				toggleLabel = "Show TODOs only"
			}
			openItemsBox.Add(widget.NewButton(toggleLabel, func() {
				showAllPlanned = !showAllPlanned
				refreshOpenItems()
				itemsAccordion.Refresh()
			}))
			if showAllPlanned {
				for _, cat := range cats {
					if cat == "TODO" || cat == "DOING" {
						continue
					}
					addCatRows(cat)
				}
				// SOMEDAY items (postponed via the "Postpone" action
				// above) deliberately stop appearing in Planned once
				// postponed (see docs/todo-carryforward-design.md),
				// so this button is the discoverable escape hatch
				// back to them, placed at the bottom of the expanded
				// "Show all" view rather than its own always-visible
				// row.
				openItemsBox.Add(widget.NewButton("Browse SOMEDAY Items…", func() {
					showSomedayBrowserWindow(a)
				}))
			}
		}
		// Excluded-tag items (Config.ReportExcludeTags) are hidden by
		// default -- surfaced via their own toggle at the very bottom
		// of the section, below even the "Show all"/SOMEDAY controls,
		// since they're the lowest-priority tier of all (see
		// splitExcludedTagItems's doc comment).
		if excludedCount > 0 {
			excludedToggleLabel := fmt.Sprintf("Show excluded-tag items (%d more)", excludedCount)
			if showExcludedPlanned {
				excludedToggleLabel = "Hide excluded-tag items"
			}
			openItemsBox.Add(widget.NewButton(excludedToggleLabel, func() {
				showExcludedPlanned = !showExcludedPlanned
				refreshOpenItems()
				itemsAccordion.Refresh()
			}))
		}
		openItemsBox.Refresh()
	}
	refreshOpenItems()

	// completedBox displays today's "end"-group entries (DONE/
	// FAIL/WASTED) grouped by category with per-category
	// sub-headings, mirroring how Planned already splits
	// TODO/GOAL/etc into their own sections (see groupOpenItemsByCategory).
	// Also collapsible, placed right below Planned in the same accordion.
	// Section is labeled "Endings" (not "Completed"/"Activity") since
	// it covers all "end" cats -- the terminal states a Plan item
	// resolves into -- not just finished/DONE ones.
	completedBox := container.NewVBox()
	showExcludedCompleted := false
	refreshCompleted = func() {
		completedBox.RemoveAll()
		items := getCategoryGroupItems("end")
		if len(items) == 0 {
			completedBox.Add(widget.NewLabel("Nothing logged yet today."))
			completedBox.Refresh()
			return
		}
		excludeTags := LoadConfig().ReportExcludeTags
		cats, grouped := groupCategoryItemsByGroup("end", items)
		excludedCount := 0
		for _, cat := range cats {
			completedBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			visible, excluded := splitExcludedTagItems(grouped[cat], excludeTags)
			excludedCount += len(excluded)
			shown := visible
			if showExcludedCompleted {
				shown = append(append([]OpenItem{}, visible...), excluded...)
			}
			for _, item := range shown {
				item := item // capture
				row := container.NewBorder(nil, nil, nil,
					newHoverIconButton(theme.Icon(theme.IconNameDocumentCreate), "Edit", func() {
						showEditItemDialog(w4, item, func() {
							fyne.Do(func() {
								refreshOpenItems()
								refreshCompleted()
								refreshLastItem()
								itemsAccordion.Refresh()
							})
						})
					}),
					itemTextLabel(categoryIconPrefix(item.Category)+item.Text))
				completedBox.Add(row)
			}
		}
		if excludedCount > 0 {
			toggleLabel := fmt.Sprintf("Show excluded-tag items (%d more)", excludedCount)
			if showExcludedCompleted {
				toggleLabel = "Hide excluded-tag items"
			}
			completedBox.Add(widget.NewButton(toggleLabel, func() {
				showExcludedCompleted = !showExcludedCompleted
				refreshCompleted()
				itemsAccordion.Refresh()
			}))
		}
		completedBox.Refresh()
	}
	refreshCompleted()

	// reflectionsBox displays today's "hilite"-group entries
	// (TIL/KUDOS/WIN/PSA/OVERCOMING/INNOVATION/LEADERSHIP/IMPACT/
	// MILESTONE/CAREER), excluding EODOnly SUMMARY/PRODUCTIVITY/
	// MEETING_HOURS metadata, grouped by category with sub-headings,
	// same pattern as Completed/Planned.
	reflectionsBox := container.NewVBox()
	showExcludedReflections := false
	var refreshReflections func()
	refreshReflections = func() {
		reflectionsBox.RemoveAll()
		items := getCategoryGroupItems("hilite")
		if len(items) == 0 {
			reflectionsBox.Add(widget.NewLabel("Nothing reflected on yet today."))
			reflectionsBox.Refresh()
			return
		}
		excludeTags := LoadConfig().ReportExcludeTags
		cats, grouped := groupCategoryItemsByGroup("hilite", items)
		excludedCount := 0
		for _, cat := range cats {
			reflectionsBox.Add(widget.NewLabelWithStyle(categoryPlural(cat), fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
			visible, excluded := splitExcludedTagItems(grouped[cat], excludeTags)
			excludedCount += len(excluded)
			shown := visible
			if showExcludedReflections {
				shown = append(append([]OpenItem{}, visible...), excluded...)
			}
			for _, item := range shown {
				item := item // capture
				row := container.NewBorder(nil, nil, nil,
					newHoverIconButton(theme.Icon(theme.IconNameDocumentCreate), "Edit", func() {
						showEditItemDialog(w4, item, func() {
							fyne.Do(func() {
								refreshReflections()
								itemsAccordion.Refresh()
							})
						})
					}),
					itemTextLabel(categoryIconPrefix(item.Category)+item.Text))
				reflectionsBox.Add(row)
			}
		}
		if excludedCount > 0 {
			toggleLabel := fmt.Sprintf("Show excluded-tag items (%d more)", excludedCount)
			if showExcludedReflections {
				toggleLabel = "Hide excluded-tag items"
			}
			reflectionsBox.Add(widget.NewButton(toggleLabel, func() {
				showExcludedReflections = !showExcludedReflections
				refreshReflections()
				itemsAccordion.Refresh()
			}))
		}
		reflectionsBox.Refresh()
	}
	refreshReflections()

	// Planned/Activity/Reflections are all collapsible (via
	// widget.Accordion) so any can be tucked out of the way once
	// reviewed. All three start collapsed -- Daybook pops up briefly
	// (per Micah) and the last-DONE-item label + buttons row above
	// already surface the most immediately relevant info, so nothing
	// needs to auto-expand.
	upcomingItem := widget.NewAccordionItem("Planned", openItemsBox)
	completedItem := widget.NewAccordionItem("Endings", completedBox)
	reflectionsItem := widget.NewAccordionItem("Hilites", reflectionsBox)
	itemsAccordion = widget.NewAccordion(completedItem, upcomingItem, reflectionsItem)

	saveEntry := func() {
		if strings.TrimSpace(input.Text) == "" {
			return
		}
		recordActivity(withMins(input.Text), selectedCat) // TODO trim emoji off front, and shorten to 4-char code
		input.SetText("")
		minsInput.SetText("")
		refreshOpenItems()
		refreshCompleted()
		refreshReflections()
		refreshLastItem()
		if enabled, _ := daybookAutoHideState(); enabled {
			hideDaybook(w4)
		}
	}
	input.OnSubmitted = func(string) { saveEntry() }
	minsInput.OnSubmitted = func(string) { saveEntry() }

	// saveBtn was created earlier (empty OnTapped) so it could be
	// placed in doneWrapper's row; wire up its actual handler now
	// that saveEntry exists.
	saveBtn.OnTapped = saveEntry

	// lastItemRow shows the current DOING item just below the buttons
	// row. Ditto extends this active item; once it reaches DONE it is no
	// longer offered as the item to continue. It uses the same inline
	// renderer as the item lists so any link remains clickable.
	lastItemRow := container.NewHBox()
	refreshLastItem = func() {
		lastItemRow.RemoveAll()
		item, ok := lastDoingItem()
		if !ok {
			lastItemRow.Add(widget.NewLabel("(nothing doing right now)"))
			lastItemRow.Refresh()
			return
		}
		lastItemRow.Add(itemTextLabel("Last doing: " + openItemDisplayText(item.Text)))
		lastItemRow.Refresh()
	}
	refreshLastItem()

	dittoBtn := widget.NewButton("Ditto", func() {
		if item, ok := lastDoingItem(); ok {
			// Ditto represents one completed nudge interval. The minutes
			// field is for the initial entry only; using it here made the
			// first click work accidentally and later clicks become no-ops
			// after the field was cleared.
			if err := dittoLifecycleItem(item, nudgeIntervalMinutes(LoadConfig())); err != nil {
				log.Println("Error applying Ditto:", err)
			}
			minsInput.SetText("")
			refreshOpenItems()
			refreshCompleted()
			refreshLastItem()
			itemsAccordion.Refresh()
		}
	})
	lastDoneRow := container.NewHBox(dittoBtn, lastItemRow)

	// showAllTagsBtn opens a standalone window listing every known
	// tag (KnownTags(), full ledger-history scan) -- "Frecent tags:"
	// only shows the top few by commonAndRecentTags's blended
	// frequency+recency score, this is the escape hatch to see
	// everything. (Editing/deleting tags across history is a possible
	// future extension, not implemented here -- see tags.go.)
	//
	// Each tag is rendered as a clickable blue tagLink (not plain
	// text) -- clicking one inserts it at the cursor position in the
	// main entry box and refocuses it, so tags can be added without
	// typing "#" and waiting for autocomplete.
	insertTagAtCursor := func(tag string) {
		runes := []rune(input.Text)
		col := input.CursorColumn
		if col < 0 || col > len(runes) {
			col = len(runes)
		}
		insert := tag
		if col > 0 && !isTagBreak(runes[col-1]) {
			insert = " " + insert
		}
		insert += " "
		newText := string(runes[:col]) + insert + string(runes[col:])
		input.SetText(newText)
		input.CursorColumn = col + len([]rune(insert))
		input.Refresh()
		FocusMainInput()
	}
	frecentTags, frecentStats := commonAndRecentTagsWithStats(8)
	frecentTagsRow := container.NewHBox(widget.NewLabel("Frecent tags:"))
	for _, tag := range frecentTags {
		tag := tag // capture
		stat := frecentStats[tag]
		frecentTagsRow.Add(newTagLink(formatTagWithCount(tag, stat), tagUsageTooltip(tag, stat), func() {
			insertTagAtCursor(tag)
		}))
	}
	commonTagsRow := container.NewBorder(nil, nil, nil,
		widget.NewButton("Show all", func() { showAllTagsWindow(a) }),
		frecentTagsRow)

	// tangentialRow holds Snooze and Help -- both tangential to the
	// normal capture flow (Micah: "not sure where [they go], maybe
	// both at very bottom of window"), so they're placed at the very
	// bottom of Daybook's content, well away from Save/the main entry
	// row. Right-aligned via NewBorder (fine here, unlike doneWrapper
	// -- this is the last row in the window with nothing after it, so
	// NewBorder's Tab-order-scrambling quirk doesn't matter).
	tangentialRow := container.NewBorder(nil, nil, nil, container.NewHBox(
		widget.NewButton("Hide", func() { hideDaybook(w4) }),
		widget.NewButton("Snooze", func() {
			Snooze(defaultSnoozeDuration())
			hideDaybook(w4)
		}),
		widget.NewButton("Help…", func() { showHelp(a) }),
	))

	content := container.NewVBox(
		startOfDayNotice,
		widget.NewLabelWithStyle("Time to record what’s just been DONE/DOING.", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		doneWrapper,
		inputSuggestions,
		// category, input,
		commonTagsRow,
		lastDoneRow,
		widget.NewSeparator(),
		itemsAccordion,
		tangentialRow,
	)
	log.Println(content)

	// grid := container.New(layout.NewFormLayout(),
	// 	label1, value1, label2, value2, content)

	// contentPad wraps the entire window content in fixed edge
	// padding via the shared windowPad helper (windowpad.go) --
	// independent of compactTheme's Size overrides (theme.go), which
	// only affect spacing *between* sibling widgets, not the gap
	// between the outermost content and the window frame. This is
	// why the earlier attempt at window-edge spacing (doneWrapper's
	// own internal CustomPaddedLayout) only visibly helped the very
	// top row (it happens to sit flush against the window's top/left/
	// right edges as the first VBox child) and did nothing for the
	// left/right/bottom edges of every other section below it, or the
	// bottom edge overall. This single outer wrap covers all edges,
	// for every section, in one place.
	contentPad := windowPad(content)

	w4.SetContent(contentPad)
	w4.Resize(fyne.NewSize(560, 400))
	w4.SetCloseIntercept(func() { hideDaybook(w4) })
	w4.Canvas().AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyW,
		Modifier: fyne.KeyModifierShortcutDefault, // Cmd+W on macOS, Ctrl+W elsewhere
	}, func(fyne.Shortcut) { hideDaybook(w4) })

	trayRefreshAll = func() {
		refreshCategoryPicker()
		refreshOpenItems()
		refreshCompleted()
		refreshReflections()
		refreshLastItem()
		if refreshStartOfDayNotice != nil {
			refreshStartOfDayNotice()
		}
	}

	// Menu
	if desk, ok := a.(desktop.App); ok {
		trayApp = a
		trayWindow = w4
		desk.SetSystemTrayMenu(buildTrayMenu(a, w4))
	}
	w4.SetMainMenu(fyne.NewMainMenu(
		fyne.NewMenu("Dunnit",
			fyne.NewMenuItem("About", func() { showAbout(a, w4) }),
		),
	))

	ShowDaybook(w4, false)

	return w4
}

// buildTrayMenu constructs the full tray/system-menu structure for
// BuildMainWindow's Fyne app+main-window (a, w4) -- factored out so
// RebuildTrayMenu can call it again after Settings changes a Kickoff/
// Review toggle, without needing BuildMainWindow itself to re-run.
func buildTrayMenu(a fyne.App, w4 fyne.Window) *fyne.Menu {
	desk, ok := a.(desktop.App)
	if !ok {
		return nil
	}

	meetingsMenu := fyne.NewMenu("Meetings",
		fyne.NewMenuItem("Meeting Prep…", func() { showMeetingPrepDialog(a) }),
		fyne.NewMenuItem("Post-Meeting Capture…", func() { showPostMeetingCapture(a, "") }),
		fyne.NewMenuItem("Standup Summary…", func() { showStandupExport(a) }),
		fyne.NewMenuItem("Recurring Meetings…", func() {
			showMiniCalendarDialog(a, w4)
		}),
	)
	meetingsItem := fyne.NewMenuItem("Meetings", nil)
	meetingsItem.ChildMenu = meetingsMenu

	reportsMenu := fyne.NewMenu("Reports",
		fyne.NewMenuItem("Summarize…", func() { showSummarizeDialog(a) }),
		fyne.NewMenuItem("Standup Summary…", func() { showStandupExport(a) }),
		fyne.NewMenuItem("Status Report…", func() { showStatusReportDialog(a) }),
		fyne.NewMenuItem("Annual Review…", func() { showAnnualReviewDialog(a) }),
		fyne.NewMenuItem("Trend View…", func() { showTrendView(a) }),
		fyne.NewMenuItem("Reports Library…", func() { showReportsLibraryWindow(a) }),
	)
	reportsItem := fyne.NewMenuItem("Reports", nil)
	reportsItem.ChildMenu = reportsMenu

	// Kickoff.../Review... submenus (docs/kickoff-review-design.md),
	// replacing the old flat Start of Day/End of Day/Start of Month
	// items. Day keeps its existing bespoke dialogs (SOD/EOD); Month
	// has its own dedicated Kickoff/Review windows
	// (monthkickoff.go/monthreview.go); Week/Quarter/Year route
	// through the generic showPeriodKickoffWindow/
	// showPeriodReviewWindow (periodkickoff.go/periodreview.go),
	// which have no bespoke dialog of their own since their shape is
	// identical. Each included item is gated by its own
	// kickoffEnabled/reviewEnabled Config toggle, re-read fresh each
	// time this function runs -- Quarter/Year default off (see
	// Config's doc comment), so they won't appear until explicitly
	// enabled via Settings.
	cfg := LoadConfig()
	var kickoffItems, reviewItems []*fyne.MenuItem
	if kickoffEnabled(cfg, periodDay) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Day…", func() { showSODWindow(a) }))
	}
	if kickoffEnabled(cfg, periodWeek) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Week…", func() { showPeriodKickoffWindow(a, periodWeek, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodMonth) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Month…", func() { showMonthKickoffWindow(a, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodQuarter) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Quarter…", func() { showPeriodKickoffWindow(a, periodQuarter, time.Now()) }))
	}
	if kickoffEnabled(cfg, periodYear) {
		kickoffItems = append(kickoffItems, fyne.NewMenuItem("Year…", func() { showPeriodKickoffWindow(a, periodYear, time.Now()) }))
	}
	if reviewEnabled(cfg, periodDay) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Day…", func() { showEODWindow(a) }))
	}
	// Week/Month/Quarter/Year Review all route through
	// showPeriodPicker first (docs/kickoff-review-design.md's "which
	// period" fix) rather than hardcoding "the previous period" --
	// lets the user pick this period (so far), last period, or a
	// short back-list instead of always landing on the wrong month.
	if reviewEnabled(cfg, periodWeek) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Week…", func() {
			showPeriodPicker(a, cfg, periodWeek, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodWeek, anchor)
			})
		}))
	}
	// Month's Review and Kickoff are now split (showMonthReviewWindow/
	// showMonthKickoffWindow), matching the generic Week/Quarter/Year
	// pattern -- see docs/kickoff-review-design.md's "Scope note".
	if reviewEnabled(cfg, periodMonth) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Month…", func() {
			showPeriodPicker(a, cfg, periodMonth, func(anchor time.Time) {
				showMonthReviewWindow(a, anchor)
			})
		}))
	}
	if reviewEnabled(cfg, periodQuarter) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Quarter…", func() {
			showPeriodPicker(a, cfg, periodQuarter, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodQuarter, anchor)
			})
		}))
	}
	if reviewEnabled(cfg, periodYear) {
		reviewItems = append(reviewItems, fyne.NewMenuItem("Year…", func() {
			showPeriodPicker(a, cfg, periodYear, func(anchor time.Time) {
				showPeriodReviewWindow(a, periodYear, anchor)
			})
		}))
	}
	kickoffItem := fyne.NewMenuItem("Kickoff", nil)
	kickoffItem.ChildMenu = fyne.NewMenu("Kickoff", kickoffItems...)
	reviewItem := fyne.NewMenuItem("Review", nil)
	reviewItem.ChildMenu = fyne.NewMenu("Review", reviewItems...)

	ledgerMenu := fyne.NewMenu("Ledger",
		fyne.NewMenuItem("Show Today’s Ledger…", func() {
			w3 := a.NewWindow("Dunnit: Today")
			w3.SetContent(windowPad(widget.NewLabel(strings.Join(readLedgerLines(), "\n"))))
			w3.Resize(fyne.NewSize(500, 400))
			w3.Show()
		}),
		fyne.NewMenuItem("Edit Today’s Ledger…", func() {
			_, fname := getLedger()
			openInEditor(fname)
		}),
		fyne.NewMenuItem("Undo/Edit Last Entry…", func() {
			showUndoEditLastEntry(a, func() {
				if trayRefreshAll != nil {
					trayRefreshAll()
				}
			})
		}),
		fyne.NewMenuItem("Search…", func() { showSearchDialog(a) }),
		fyne.NewMenuItem("Navigator…", func() { showNavigatorWindow(a) }),
		fyne.NewMenuItem("SOMEDAY Items…", func() { showSomedayBrowserWindow(a) }),
		fyne.NewMenuItem("Recurring Items…", func() {
			showRecurringItemsDialog(a, w4)
		}),
		fyne.NewMenuItem("EOD Report…", func() {
			go func() {
				path, _, err := ensureEODReport(time.Now())
				if err != nil {
					log.Println("Error drafting EOD report:", err)
					return
				}
				if path != "" {
					openInEditor(path)
				}
			}()
		}),
	)
	ledgerItem := fyne.NewMenuItem("Ledger", nil)
	ledgerItem.ChildMenu = ledgerMenu
	var syncItem *fyne.MenuItem
	if cfg.GitSyncEnabled {
		syncMenu := fyne.NewMenu("Sync",
			fyne.NewMenuItem("Push", func() { runGitSyncAction(a, "push") }),
			fyne.NewMenuItem("Pull", func() { runGitSyncAction(a, "pull") }),
		)
		syncItem = fyne.NewMenuItem("Sync", nil)
		syncItem.ChildMenu = syncMenu
	}

	snoozeMenu := fyne.NewMenu("Snooze",
		fyne.NewMenuItem("15 min", func() { Snooze(15 * time.Minute) }),
		fyne.NewMenuItem("30 min", func() { Snooze(30 * time.Minute) }),
		fyne.NewMenuItem("1 hour", func() { Snooze(60 * time.Minute) }),
	)
	snoozeItem := fyne.NewMenuItem("Snooze", func() { Snooze(defaultSnoozeDuration()) })
	snoozeItem.ChildMenu = snoozeMenu

	var dndItem *fyne.MenuItem
	var m *fyne.Menu
	dndItem = fyne.NewMenuItem("Do Not Disturb", func() {
		on := !dndItem.Checked
		SetDoNotDisturb(on)
		dndItem.Checked = on
		desk.SetSystemTrayMenu(m)
	})
	dndItem.Checked = IsDoNotDisturb()

	// Since Daybook is normally hidden and only pops up briefly (per
	// Micah), the tray menu -- not Daybook -- is the primary surface
	// for anything that isn't a direct reaction to Daybook already
	// being on screen. Frequent/time-sensitive items (Show, Kickoff/
	// Review, Snooze) stay top-level and un-buried; everything else
	// groups into a submenu by domain (Meetings/Reports/Ledger)
	// rather than by FR number or chronology.
	menuItems := []*fyne.MenuItem{
		fyne.NewMenuItem("Show", func() {
			ShowDaybook(w4, false)
		}),
		fyne.NewMenuItemSeparator(),
		kickoffItem,
		reviewItem,
		snoozeItem,
		dndItem,
		fyne.NewMenuItemSeparator(),
		meetingsItem,
		reportsItem,
		ledgerItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Help…", func() { showHelp(a) }),
		fyne.NewMenuItem("Settings…", func() { showSettings(a) }),
	}
	if syncItem != nil {
		prefix := append([]*fyne.MenuItem{}, menuItems[:len(menuItems)-3]...)
		suffix := append([]*fyne.MenuItem{}, menuItems[len(menuItems)-3:]...)
		menuItems = append(prefix, syncItem)
		menuItems = append(menuItems, suffix...)
	}
	m = fyne.NewMenu("Dunnit", menuItems...)
	return m
}

// RebuildTrayMenu re-reads Config and reapplies the tray menu, so a
// Kickoff/Review toggle changed in Settings (or any other config.toml
// change affecting menu contents) takes effect immediately rather
// than requiring an app restart. No-op if BuildMainWindow hasn't run
// yet (trayApp/trayWindow unset) or the platform has no desktop tray.
func RebuildTrayMenu() {
	if trayApp == nil || trayWindow == nil {
		return
	}
	desk, ok := trayApp.(desktop.App)
	if !ok {
		return
	}
	desk.SetSystemTrayMenu(buildTrayMenu(trayApp, trayWindow))
	if trayRefreshAll != nil {
		trayRefreshAll()
	}
}

func updateTime(clock *widget.Label) {
	formatted := time.Now().Format("Dunnit: 03:04:05")
	clock.SetText(formatted)
}
