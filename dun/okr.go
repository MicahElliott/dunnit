package dun

import (
	"strings"
	"time"
)

// OKR ledger categories (docs/kickoff-review-design.md's OKR design,
// agreed 2026-09-02): kept as plain single-line ledger entries, same
// append-only pattern as every other category -- no new file format,
// no new ID bookkeeping.
//
//   - OBJECTIVE <text> #<periodTag> -- one per objective.
//   - KEYRESULT <text> #<periodTag> -- one per key result; associated
//     to the nearest preceding OBJECTIVE line sharing the same
//     periodTag purely by ledger order/adjacency (no explicit parent
//     ID), same "loose matching is fine, low volume, easy to eyeball"
//     tradeoff already accepted for IDEA/SOMEDAY triage.
//   - KEYRESULT-STATUS <status> <text> #<periodTag> -- appended by
//     Review to record/update a KR's status without ever mutating the
//     original KEYRESULT line (never-lose-data, append-only). The
//     latest KEYRESULT-STATUS line for a given KR text+tag wins when
//     reading current status; a KR with no status line yet is
//     "not started" and naturally carries forward to the next Review
//     with no separate carry-forward bookkeeping needed.
//
// Deliberately single-line, like every other ledger category --
// see 2026-09-02 discussion: a KR needing more than one line of text
// is a sign it should be tightened, not a case for multi-line ledger
// syntax; richer elaboration belongs in a saved Review report instead.
const (
	CategoryObjective       = "OBJECTIVE"
	CategoryKeyResult       = "KEYRESULT"
	CategoryKeyResultStatus = "KEYRESULT-STATUS"
	// CategoryFocus is a lightweight, non-scored freeform "theme for
	// this period" statement (e.g. "Consolidation quarter") --
	// distinct from OBJECTIVE/KEYRESULT, no scoring implied, just a
	// one-line framing note set at Kickoff and read back at Review.
	CategoryFocus = "FOCUS"
)

// okrStatusOptions are the fixed statuses a KEYRESULT-STATUS line can
// record, in display order for a widget.Select.
var okrStatusOptions = []string{"Not Started", "On Track", "At Risk", "Done"}

// periodTag returns the #tag token used to associate an OKR ledger
// line with the quarter or year containing anchor -- "#Q3-2026" for
// periodQuarter, "#2026" for periodYear. Other periods aren't
// meaningful for OKRs (see docs/kickoff-review-design.md) and return
// "".
func periodTag(period summaryPeriod, anchor time.Time) string {
	switch period {
	case periodQuarter:
		return "#Q" + itoa(quarterOf(anchor)) + "-" + itoa(anchor.Year())
	case periodYear:
		return "#" + itoa(anchor.Year())
	default:
		return ""
	}
}

// itoa is a tiny strconv.Itoa wrapper, avoiding a fresh "strconv"
// import purely for two call sites above -- kept local rather than
// pulling in the package for one function.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

// Objective is one OBJECTIVE line plus its associated KeyResults,
// matched purely by ledger order (KEYRESULT lines immediately
// following an OBJECTIVE line, sharing the same periodTag, belong to
// it) -- see the package-level OKR category doc comment above.
type Objective struct {
	Text       string
	KeyResults []KeyResult
}

// KeyResult is one KEYRESULT line plus its latest known status (from
// the most recent matching KEYRESULT-STATUS line, if any -- "Not
// Started" if none yet).
type KeyResult struct {
	Text   string
	Status string
	Note   string
}

// recordObjective/recordKeyResult append a new OBJECTIVE/KEYRESULT
// ledger line tagged for period's unit containing anchor.
func recordObjective(text string, period summaryPeriod, anchor time.Time) {
	recordActivity(text+" "+periodTag(period, anchor), CategoryObjective)
}

func recordKeyResult(text string, period summaryPeriod, anchor time.Time) {
	recordActivity(text+" "+periodTag(period, anchor), CategoryKeyResult)
}

// recordFocus appends a FOCUS ledger line (the "theme for this
// period" freeform note) tagged for period's unit containing anchor.
// Setting a new one doesn't erase the old line (append-only); readFocus
// returns the latest one found.
func recordFocus(text string, period summaryPeriod, anchor time.Time) {
	recordActivity(text+" "+periodTag(period, anchor), CategoryFocus)
}

// readFocus returns the most recently recorded FOCUS text for
// period's unit containing anchor, or "" if none has been set.
func readFocus(period summaryPeriod, anchor time.Time) string {
	tag := periodTag(period, anchor)
	if tag == "" {
		return ""
	}
	latest := ""
	for _, path := range allLedgerFiles() {
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || cat != CategoryFocus || !strings.Contains(text, tag) {
				continue
			}
			latest = strings.TrimSpace(strings.Replace(text, tag, "", 1))
		}
	}
	return latest
}

// recordKeyResultStatus appends a KEYRESULT-STATUS line scoring
// keyResultText (the original KR's text, tag stripped) for period's
// unit containing anchor -- never mutates the original KEYRESULT
// line, so re-scoring across multiple Reviews just adds more status
// lines, the latest of which wins (see readObjectives).
func recordKeyResultStatus(keyResultText, status, note string, period summaryPeriod, anchor time.Time) {
	text := status + ": " + keyResultText
	if strings.TrimSpace(note) != "" {
		text += " -- " + note
	}
	recordActivity(text+" "+periodTag(period, anchor), CategoryKeyResultStatus)
}

// readObjectives scans every ledger file for OBJECTIVE/KEYRESULT/
// KEYRESULT-STATUS lines tagged for period's unit containing anchor,
// and reassembles them into Objectives with their KeyResults attached
// by ledger-order adjacency (loose matching, per the package doc
// comment -- fine for OKRs' low volume). Returns objectives in
// first-seen order; a KEYRESULT line appearing before any OBJECTIVE
// line (e.g. hand-edited ledger) is attached to an implicit
// "(ungrouped)" objective so it's never silently dropped.
func readObjectives(period summaryPeriod, anchor time.Time) []Objective {
	tag := periodTag(period, anchor)
	if tag == "" {
		return nil
	}

	// statuses maps KR text -> latest {status, note} found, in
	// first-seen-overwrites-later order (we scan files in
	// filesystem-walk order, which for ledger-YYYYMMDD.txt names is
	// effectively chronological -- later files' statuses overwrite
	// earlier ones for the same KR text, giving "latest wins").
	statuses := make(map[string]KeyResult)

	var objectives []Objective
	var current *Objective
	var ungrouped Objective
	ungrouped.Text = "(ungrouped)"

	for _, path := range allLedgerFiles() {
		for _, line := range readLedgerLinesFrom(path) {
			cat, text, ok := parseLedgerLine(line)
			if !ok || !strings.Contains(text, tag) {
				continue
			}
			bare := strings.TrimSpace(strings.Replace(text, tag, "", 1))
			switch cat {
			case CategoryObjective:
				objectives = append(objectives, Objective{Text: bare})
				current = &objectives[len(objectives)-1]
			case CategoryKeyResult:
				kr := KeyResult{Text: bare, Status: "Not Started"}
				if current != nil {
					current.KeyResults = append(current.KeyResults, kr)
				} else {
					ungrouped.KeyResults = append(ungrouped.KeyResults, kr)
				}
			case CategoryKeyResultStatus:
				// bare is "STATUS: original text" or
				// "STATUS: original text -- note".
				parts := strings.SplitN(bare, ": ", 2)
				if len(parts) != 2 {
					continue
				}
				status := parts[0]
				rest := parts[1]
				note := ""
				if idx := strings.Index(rest, " -- "); idx >= 0 {
					note = rest[idx+4:]
					rest = rest[:idx]
				}
				statuses[rest] = KeyResult{Text: rest, Status: status, Note: note}
			}
		}
	}

	if len(ungrouped.KeyResults) > 0 {
		objectives = append(objectives, ungrouped)
	}

	// Apply latest-status overlay.
	for oi := range objectives {
		for ki, kr := range objectives[oi].KeyResults {
			if latest, ok := statuses[kr.Text]; ok {
				objectives[oi].KeyResults[ki].Status = latest.Status
				objectives[oi].KeyResults[ki].Note = latest.Note
			}
		}
	}

	return objectives
}

// okrSummaryText renders period's Objectives/KeyResults (if any, and
// only if cfg.EnableOKRs) as plain text suitable for feeding into a
// Review's AI prompt as structured input alongside the raw-ledger/
// sub-report rollup material -- see generateThemedReview (period.go).
// Returns "" if OKRs aren't enabled or there are none set for this
// period.
func okrSummaryText(cfg Config, period summaryPeriod, anchor time.Time) string {
	if !cfg.EnableOKRs {
		return ""
	}
	objectives := readObjectives(period, anchor)
	if len(objectives) == 0 {
		return ""
	}
	var b strings.Builder
	if focus := readFocus(period, anchor); focus != "" {
		b.WriteString("Theme for this period: " + focus + "\n\n")
	}
	for _, o := range objectives {
		b.WriteString("Objective: " + o.Text + "\n")
		for _, kr := range o.KeyResults {
			line := "  - [" + kr.Status + "] " + kr.Text
			if kr.Note != "" {
				line += " (" + kr.Note + ")"
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}
