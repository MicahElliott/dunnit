package dun

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
	// DONE/FAIL/WASTED, plus the internal ONGOING marker), "plan"
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
// field on Daybook's main entry row is shown -- deliberately narrowed
// (2026-09-03) to just the three "endpoint" categories (DONE/FAIL/
// WASTED, see the "Endpoints" doc comment above Categories) that a
// Plan-group item resolves into: these are the only categories where
// "how long did this take" reads as a real completed-effort duration.
// Explicitly excludes:
//   - "Plan"-group items (TODO/GOAL/etc): a mins value here would
//     read as a time *estimate*, not an actual duration -- a
//     different concept this field was never meant to capture (no
//     hours/days-scale estimation feature exists, and adding one
//     would be overkill for large accomplishments anyway).
//   - ONGOING: internal/mechanical marker (Ditto's own bookkeeping),
//     never hand-picked, so its "mins" relevance is moot.
//   - MEETING/WAITING: previously included, but these aren't
//     completed-effort entries either (MEETING is scratch agenda
//     notes; WAITING is "blocked," not "done") -- narrowed out
//     alongside the Plan-group exclusion above.
var timeTrackableCategories = map[string]bool{
	"DONE": true, "FAIL": true, "WASTED": true,
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

// GroupLabel returns a display name + description for a group, used
// as section headers in the Category Legend.
func GroupLabel(group string) string {
	switch group {
	case "end":
		return "End — day-to-day capture, the terminal states a Plan item resolves into"
	case "plan":
		return "Plan — future-facing, open items tracked toward a resolution to DONE in \"End\""
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
// "Endpoints" (2026-09-02 regroup, further narrowed later): "end" now
// holds ONLY the literal terminal states a "Plan"-group item
// (TODO/IDEA/GOAL/FIXME/etc.) resolves into -- DONE/FAIL/WASTED --
// plus the internal ONGOING marker. TIL/KUDOS/WIN moved out of "end"
// and into "hilite" alongside IMPACT/MILESTONE/CAREER/PSA: all of
// these are freestanding notable-moment callouts that don't resolve
// any specific open item, a genuinely different concept from an
// endpoint. See docs/category-taxonomy.md for the fuller design
// discussion behind this split, including the still-informal "(from
// TODO)"-style promotion-annotation convention (a plain-text marker
// written by hand when closing a Plan item into one of these
// endpoints -- not yet a structured/enforced mechanism).
var Categories = []Category{
	// end: literal endpoints only -- DONE/FAIL/WASTED, the terminal
	// states Plan-group items resolve into. ONGOING is marked EODOnly
	// (2026-09-02) -- despite the name, it's purely an internal/
	// mechanical marker written by Ditto's own rewrite logic
	// (recordExtended in ui.go), never meant to be hand-picked from
	// the live picker or explained in Help; EODOnly already means
	// exactly "written by a dedicated internal flow, excluded from
	// the picker and from Help" (see the EODOnly field doc above),
	// which fits Ditto's ONGOING rewrite just as well as it fits
	// eod.go's SUMMARY/PRODUCTIVITY/MEETING_HOURS. WASTED is further
	// gated behind Config.WastedTimeTrackingEnabled (default false,
	// see config.go) -- an opt-in feature, hidden from the live
	// picker when off, though still present here for Help/legend and
	// historical ledger entries.
	{"✔️", "DONE", "Something you completed. The most common endpoint a \"Plan\" item (TODO/IDEA/GOAL/etc.) resolves into — see docs/category-taxonomy.md.", "end", "positive", false},
	{"⏩", "ONGOING", "Still working on something (e.g. what \"Ditto\" logs) — not finished yet. Purely an internal/mechanical marker (Ditto's own bookkeeping), not part of the endpoint/promotion taxonomy.", "end", "", true},
	{"❌", "FAIL", "Something that didn't go as hoped — an endpoint a \"Plan\" item can resolve into, same as DONE, just the unsuccessful outcome.", "end", "negative", false},
	{"🗑️", "WASTED", "Unfocused, pointless work or distraction. Opt-in: hidden from the live picker unless Config.WastedTimeTrackingEnabled is set.", "end", "negative", false},

	// plan: future-facing -- includes the "open item, needs follow-up
	// or resolution" categories (WAITING/QUESTION/FIXME/RISK moved
	// here from "end", alongside TODO/GOAL, since they share the same
	// pattern: logged now, tracked as open, resolved/reviewed later
	// via SOD/SOM/Daybook's Upcoming list -- not truly "day-to-day
	// capture" like DONE/etc). IDEA also moved here (from "end")
	// -- it's future-facing/not-yet-actioned just like SOMEDAY, and
	// som.go's step 2 already treats IDEA/SOMEDAY as a matched pair
	// for triage, so grouping them together in the picker too keeps
	// that pairing consistent. TODO leads the group (2026-09-02,
	// moved ahead of IDEA per explicit request) since it's the most
	// common/actionable item in this group.
	{"📌", "TODO", "A small, tight, near-term item — actively encouraged. Roughly Jira's \"Task\": scoped and ready to act on (vs. IDEA, which is the same thing before it's scoped).", "plan", "", false},
	{"💡", "IDEA", "A new idea worth capturing, not yet scoped/ready to act on — an earlier maturity stage of TODO (loosely: an un-scoped Jira \"Story\"), not a different type of item.", "plan", "", false},
	{"🎯", "GOAL", "A bigger overarching aim, reviewed on a longer cadence (not daily). Roughly Jira's \"Epic\": a rollup that TODOs/FIXMEs work toward, not itself a single actionable item.", "plan", "", false},
	{"❓", "QUESTION", "An open question to follow up on.", "plan", "", false},
	{"⏳", "WAITING", "Blocked on someone/something else; not actionable right now.", "plan", "", false},
	{"🔧", "FIXME", "Something broken that needs fixing — roughly Jira's \"Bug\": a genuinely different *type* of item than TODO (fixing vs. building), not just a maturity stage like IDEA is.", "plan", "negative", false},
	{"⚠️", "RISK", "A risk worth flagging/tracking.", "plan", "negative", false},
	{"📅", "MEETING", "Scratch agenda-builder notes for an upcoming meeting (tag-scoped).", "plan", "", false},
	{"🕰️", "SOMEDAY", "Something you might want to do eventually, not now (also where stalled TODOs/GOALs land).", "plan", "", false},
	{"🏎️", "OPTIMIZE", "Something working but worth improving/speeding up.", "plan", "", false},

	// hilite: freestanding notable-moment callouts -- not tied to
	// resolving any specific "Plan" item (unlike DONE/FAIL/WASTED,
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
	{"🌱", "TIL", "Today I Learned — something new you picked up.", "hilite", "positive", false},
	{"🙌", "KUDOS", "Recognition given to someone else, or received from someone else.", "hilite", "positive", false},
	{"🏆", "WIN", "A distinct, successful moment or completed task — short-term momentum, not necessarily part of a bigger journey.", "hilite", "positive", false},
	{"📢", "PSA", "An announcement or heads-up the team should know about — not a personal accomplishment, just something worth broadcasting.", "hilite", "positive", false},
	{"💪", "OVERCOMING", "A time you turned a failure, roadblock, or crisis into a recovery — the comeback, not just the setback.", "hilite", "positive", false},
	{"✨", "INNOVATION", "You created a new process, tool, or idea from scratch — something that didn't exist before.", "hilite", "positive", false},
	{"👑", "LEADERSHIP", "A moment you mentored someone, led an initiative, or influenced a decision without direct authority.", "hilite", "positive", false},
	{"💥", "IMPACT", "The measurable result or value your action caused, not just what you did — why it mattered (e.g. saved time, grew revenue).", "hilite", "positive", false},
	{"🏁", "MILESTONE", "A significant checkpoint or phase transition in a longer journey, bigger in scope than a single WIN (e.g. shipping v1, a work anniversary).", "hilite", "positive", false},
	{"💼", "CAREER", "A big, resume/CV-worthy accomplishment — not a plan, a retrospective note that something huge happened.", "hilite", "positive", false},
	{"🔚", "SUMMARY", "A wrap-up/summary note (typically written via End of Day).", "hilite", "", true},
	{"📈", "PRODUCTIVITY", "A note on your own productivity/efficiency (typically written via End of Day).", "hilite", "", true},
	{"🕑", "MEETING_HOURS", "How many hours of meetings you were in today (typically written via End of Day).", "hilite", "", true},
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
// ones (SUMMARY/PRODUCTIVITY/MEETING_HOURS/ONGOING -- these are
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
