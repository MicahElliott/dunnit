package dun

import (
	"fmt"
	"strings"
)

// Category defines one selectable Daybook category: its short code
// (written verbatim into ledger lines) and the emoji-prefixed label
// shown in the picker UI. This is the single source of truth for the
// category list (FR-03) -- other code (category picker, legend UI in
// FR-06) should build off Categories rather than hardcoding its own
// copy.
type Category struct {
	Emoji string
	Code  string
	// Help is a one-line description of intended use, surfaced by
	// FR-06's in-app legend.
	Help string
	// Group buckets categories for the picker's quick-filter buttons:
	// "end" (the literal endpoints a "plan" item resolves into --
	// DONE/HANDLED/FAIL/WASTED), "plan"
	// (future-facing, open/tracked items), or "hilite" (freestanding
	// notable-moment callouts, not tied to resolving any specific
	// "plan" item -- see docs/category-taxonomy.md). Purely a UI
	// convenience -- doesn't affect what's written to the ledger.
	Group string
	// Sentiment is "positive", "negative", or "" (neutral), used to
	// color-code the Category Legend (dark green / dark red / default).
	Sentiment string
	// EODOnly marks a category as written only via a dedicated flow
	// (EOD's Finalize Day, SOM's wizard) rather than picked by hand
	// in Daybook's live category picker -- e.g. IMPACT/SUMMARY/
	// MEETING_HOURS are always recordActivity'd directly by eod.go/
	// som.go with a fixed code, never selected from the dropdown.
	// Still included in Categories (and thus the Help legend,
	// annual review scans, etc) for documentation purposes -- only
	// excluded from CategoryLabelsForGroup's picker options.
	EODOnly bool
}

// timeTrackableCategories are the codes for which the optional "mins"
// field on Daybook's main entry row is shown. DONE/FAIL/WASTED are
// completed-effort endpoints; DOING is included so an active lifecycle
// item can accumulate explicit minutes before it reaches DONE.
// Explicitly excludes:
//   - Other "Plan"-group items (TODO/GOAL/etc): a mins value here would
//     read as a time *estimate*, not an actual duration -- a
//     different concept this field was never meant to capture (no
//     hours/days-scale estimation feature exists, and adding one
//     would be overkill for large accomplishments anyway).
//   - MEETING/WAITING: previously included, but these aren't
//     completed-effort entries either (MEETING is scratch agenda
//     notes; WAITING is "blocked," not "done") -- narrowed out
//     alongside the Plan-group exclusion above.
var timeTrackableCategories = map[string]bool{
	"DONE": true, "DOING": true, "FAIL": true, "WASTED": true,
}

// IsTimeTrackable reports whether mins tracking is conventionally
// meaningful for this category code -- used by Daybook's main entry
// row (ui.go) to show/hide the "mins" field based on the currently
// selected category.
func IsTimeTrackable(code string) bool {
	return timeTrackableCategories[code]
}

// Label returns the picker-facing string, e.g. "✔️ DONE".
func (c Category) Label() string {
	return c.Emoji + " " + c.Code
}

// EmojiForCode returns the category icon for code, or "" when code is
// unknown. Readback lists use it in place of generic bullet markers.
func EmojiForCode(code string) string {
	for _, c := range Categories {
		if c.Code == code {
			return c.Emoji
		}
	}
	return ""
}

// GroupLabel returns a display name + description for a group, used
// as section headers in the Category Legend.
func GroupLabel(group string) string {
	switch group {
	case "end":
		return "End — terminal states a Planned item resolves into"
	case "plan":
		return "Plan — TODO/DOING open lifecycle items and other work tracked toward DONE"
	case "hilite":
		return "Hilite — freestanding notable-moment callouts, not tied to resolving any specific Plan item"
	}
	return group
}

// Categories is the full ordered list of categories offered in
// Daybook's picker, most common/important first within each group
// (DONE, TODO, IDEA, QUESTION, TIL, MEETING lead "end"/"plan"), negative-
// sentiment categories placed last within their group. `MTG` was
// dropped in favor of `MEETING` only (FR-05); `BLOCKER`/`BLOCKED` was
// replaced by `WAITING` (FR-03).
//
// "Endpoints" (2026-09-02 regroup, further narrowed later): "end" holds
// the terminal states a "Plan"-group item
// (TODO/IDEA/GOAL/FIXME/etc.) resolves into -- DONE/HANDLED/FAIL/WASTED/
// DISCARDED.
// TIL/KUDOS/WIN moved out of "end"
// and into "hilite" alongside IMPACT/MILESTONE/CAREER/PSA: all of
// these are freestanding notable-moment callouts that don't resolve
// any specific open item, a genuinely different concept from an
// endpoint. See docs/category-taxonomy.md for the fuller design
// discussion behind this split, including the still-informal "(from
// TODO)"-style promotion-annotation convention (a plain-text marker
// written by hand when closing a Plan item into one of these
// endpoints -- not yet a structured/enforced mechanism).
var Categories = []Category{
	// end: literal endpoints only -- DONE/HANDLED/FAIL/WASTED/DISCARDED, the terminal
	// states Plan-group items resolve into. WASTED is further
	// gated behind Config.WastedTimeTrackingEnabled (default false,
	// see config.go) -- an opt-in feature, hidden from the live
	// picker when off, though still present here for Help/legend and
	// historical ledger entries.
	{"✔️", "DONE", "Completed work.", "end", "positive", false},
	{"🤝", "HANDLED", "Someone else completed a TODO or DOING; add @Name when useful.", "end", "positive", false},
	{"❌", "FAIL", "Work that did not succeed.", "end", "negative", false},
	{"🗑️", "WASTED", "Unfocused or pointless work.", "end", "negative", false},
	{"🚫", "DISCARDED", "An open item deliberately dropped without completing it.", "end", "negative", true},

	// plan: future-facing -- includes the "open item, needs follow-up
	// or resolution" categories (WAITING/QUESTION/FIXME/RISK moved
	// here from "end", alongside TODO/GOAL, since they share the same
	// pattern: logged now, tracked as open, resolved/reviewed later
	// via SOD/SOM/Daybook's Upcoming list -- not truly "day-to-day
	// capture" like DONE/etc). IDEA also moved here (from "end")
	// -- it's future-facing/not-yet-actioned just like SOMEDAY, and
	// som.go's step 2 already treats IDEA/SOMEDAY as a matched pair
	// for triage, so grouping them together in the picker too keeps
	// that pairing consistent. DOING leads the group so active work sits
	// closest to Endings in Daybook; TODO follows as the next state in
	// the lifecycle.
	{"▶️", "DOING", "A TODO currently in progress.", "plan", "", false},
	{"📌", "TODO", "A small, actionable near-term task.", "plan", "", false},
	{"💡", "IDEA", "An idea not yet ready to act on.", "plan", "", false},
	{"🎯", "GOAL", "A larger aim that TODOs work toward.", "plan", "", false},
	{"❓", "QUESTION", "An open question to follow up on.", "plan", "", false},
	{"⏳", "WAITING", "Blocked on someone or something else.", "plan", "", false},
	{"🔧", "FIXME", "A bug or broken thing to fix.", "plan", "negative", false},
	{"⚠️", "RISK", "A risk worth tracking.", "plan", "negative", false},
	{"📅", "MEETING", "Agenda notes for an upcoming meeting.", "plan", "", false},
	{"🕰️", "SOMEDAY", "Something to do eventually, not now.", "plan", "", false},
	{"🏎️", "OPTIMIZE", "Working well, but worth improving.", "plan", "", false},

	// hilite: freestanding notable-moment callouts -- not tied to
	// resolving any specific "Plan" item (unlike DONE/HANDLED/FAIL/WASTED,
	// which stay in "end" -- see the Categories doc comment above).
	// TIL/KUDOS/WIN joined this group (moved from "end") alongside
	// IMPACT/MILESTONE/CAREER/PSA -- all stay pickable by hand (you
	// might want to log one the moment it happens, not just at EOD).
	// SUMMARY/PRODUCTIVITY/MEETING_HOURS are EODOnly: they're always
	// written by eod.go's Finalize Day flow with a fixed value/text,
	// never meaningfully hand-picked mid-day from the dropdown -- day-
	// level meta-notes, arguably a fourth concept of their own (see
	// docs/category-taxonomy.md) but left bundled into "hilite" for
	// now rather than splitting into a new group.
	{"🌱", "TIL", "Something new you learned today.", "hilite", "positive", false},
	{"🙌", "KUDOS", "Recognition given or received.", "hilite", "positive", false},
	{"🏆", "WIN", "A distinct success or completed task.", "hilite", "positive", false},
	{"📢", "PSA", "An announcement or team heads-up.", "hilite", "positive", false},
	{"💪", "OVERCOMING", "A setback or crisis you recovered from.", "hilite", "positive", false},
	{"✨", "INNOVATION", "A new process, tool, or idea you created.", "hilite", "positive", false},
	{"👑", "LEADERSHIP", "A moment of leadership or influence.", "hilite", "positive", false},
	{"💥", "IMPACT", "The result or value your action created.", "hilite", "positive", false},
	{"🏁", "MILESTONE", "A significant checkpoint in a longer journey.", "hilite", "positive", false},
	{"💼", "CAREER", "A resume-worthy accomplishment.", "hilite", "positive", false},
	{"🔚", "SUMMARY", "A wrap-up or summary note.", "hilite", "", true},
	{"📈", "PRODUCTIVITY", "A note about productivity or efficiency.", "hilite", "", true},
	{"🕑", "MEETING_HOURS", "The number of meeting hours today.", "hilite", "", true},
}

// GroupForCode returns the Group of the category with the given code
// (e.g. "DONE" -> "end"), or "" if no such category exists. Used by
// the Edit Entry dialog (undo.go) to populate its category dropdown
// with only the categories sharing the edited item's own group,
// rather than every category in the app.
func GroupForCode(code string) string {
	for _, c := range Categories {
		if c.Code == code {
			return c.Group
		}
	}
	return ""
}

// CategoryExists reports whether code is a real, current Category
// code (case-sensitive exact match, e.g. "DONE") -- used by cmd/
// dunnit's CLI to validate its CATEGORY argument before writing
// anything to the ledger. Deliberately does not exclude EODOnly
// categories (SUMMARY/PRODUCTIVITY/MEETING_HOURS) or gate WASTED on
// Config.WastedTimeTrackingEnabled -- unlike Daybook's live picker,
// the CLI is a deliberate power-user/automation entry point that
// accepts any real category code, not just what the picker currently
// offers.
func CategoryExists(code string) bool {
	for _, c := range Categories {
		if c.Code == code {
			return true
		}
	}
	return false
}

// HelpForCode returns the Help text for the category with the given
// code (e.g. "DONE"), or "" if no such category exists. Used by
// Daybook's live category picker (ui.go's hoverSelect) so its hover
// tooltip reuses the exact same wording as the Help window's legend,
// rather than a separate hardcoded copy.
func HelpForCode(code string) string {
	for _, c := range Categories {
		if c.Code == code {
			return c.Help
		}
	}
	return ""
}

// CategoryOptionsForGroup returns Label() strings (emoji + code, e.g.
// "✔️ DONE") for categories in the given group, excluding EODOnly
// ones (SUMMARY/PRODUCTIVITY/MEETING_HOURS -- these are
// always machine-written by a dedicated flow, never meant to be
// hand-picked). Unlike CategoryLabelsForGroup, this does NOT gate
// WASTED on Config.WastedTimeTrackingEnabled -- used by the Edit
// Entry dialog (undo.go), where a user editing an existing entry
// should be able to pick from the full real category set within a
// group regardless of that live-picker-only opt-in flag.
func CategoryOptionsForGroup(group string) []string {
	var labels []string
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if c.Group == group {
			labels = append(labels, c.Label())
		}
	}
	return labels
}

// CategoryLabels returns the Label() strings for all Categories, in
// order, for use in widget.NewSelect.
func CategoryLabels() []string {
	labels := make([]string, len(Categories))
	for i, c := range Categories {
		labels[i] = c.Label()
	}
	return labels
}

// CategoryLabelsForGroup returns Label() strings for categories
// matching the given group ("end", "plan", "hilite"), or all
// categories if group is "" or "all". Used by the picker's quick-
// filter buttons. Excludes EODOnly categories (e.g. SUMMARY/
// PRODUCTIVITY/MEETING_HOURS) -- those are only ever written via
// eod.go's Finalize Day flow, not meant to be hand-picked here. Also
// excludes WASTED unless cfg.WastedTimeTrackingEnabled is set (see
// Config.WastedTimeTrackingEnabled) -- same "hide from picker, still
// documented" pattern as EODOnly, just gated on a runtime flag
// instead of a fixed struct field.
func CategoryLabelsForGroup(cfg Config, group string) []string {
	var labels []string
	for _, c := range Categories {
		if c.EODOnly {
			continue
		}
		if c.Code == "WASTED" && !cfg.WastedTimeTrackingEnabled {
			continue
		}
		if group == "" || group == "all" || c.Group == group {
			labels = append(labels, c.Label())
		}
	}
	return labels
}

// CategoryLabelsForFaves returns Label() strings for the user's
// configured "Faves" bucket (Config.FavoriteCategories, a freely
// user-chosen set of category codes -- unlike End/Plan/Hilite, which
// are Categories' own fixed Group field, Faves is entirely
// user-defined and can mix codes across groups, e.g. the suggested
// default DONE/TODO/IDEA/FIXME/MEETING). Preserves Categories' overall
// order (not the order codes were added to the config) and silently
// skips any code that isn't a real/current category (e.g. after a
// category is ever renamed/removed), is EODOnly, or is WASTED while
// cfg.WastedTimeTrackingEnabled is false.
func CategoryLabelsForFaves(cfg Config) []string {
	want := make(map[string]bool, len(cfg.FavoriteCategories))
	for _, code := range cfg.FavoriteCategories {
		want[code] = true
	}
	var labels []string
	for _, c := range Categories {
		if c.EODOnly || !want[c.Code] {
			continue
		}
		if c.Code == "WASTED" && !cfg.WastedTimeTrackingEnabled {
			continue
		}
		labels = append(labels, c.Label())
	}
	return labels
}

// categoryPromptGuidance gives every report prompt the same interpretation of
// ledger categories. The wording comes from Categories so prompt semantics do
// not drift from the Daybook picker and Help legend.
func categoryPromptGuidance() string {
	var b strings.Builder
	b.WriteString("\n\nDunnit ledger semantics — treat category codes as meaningful labels, not ordinary prose:\n")
	for _, category := range Categories {
		fmt.Fprintf(&b, "- %s %s (%s): %s", category.Emoji, category.Code, category.Group, category.Help)
		if category.EODOnly {
			b.WriteString(" This is written by a dedicated end-of-day flow.")
		}
		b.WriteByte('\n')
	}
	b.WriteString("- SENTIMENT (metadata): the user's end-of-day sentiment rating.\n")
	b.WriteString("- FOCUS (metadata): the theme chosen for a period.\n")
	b.WriteString("- OBJECTIVE (OKR): a period-level objective.\n")
	b.WriteString("- KEYRESULT (OKR): a measurable result attached to the nearest matching objective.\n")
	b.WriteString("- KEYRESULT-STATUS (OKR): the latest recorded status or note for a key result.\n")
	b.WriteString("Plan entries are open or future-facing; End entries are outcomes; Hilites are notable evidence and do not by themselves resolve a Plan item. Preserve the distinction between completed work, work in progress, possibilities, blockers, meetings, and reflections. Do not count carry-forward copies or overlapping prior summaries as additional work, and do not invent facts.")
	return b.String()
}
