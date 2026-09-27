package dun

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ReportFile describes one generated report file found on disk --
// the reports-corpus counterpart to LedgerEntry (ledgerentry.go),
// deliberately much lighter: reports are large markdown documents,
// not per-line structured data, so this indexes filenames/metadata
// only, not full content (that's read on demand, see ReportBody).
// See docs/navigator-design.md's "Reports-corpus indexing" section
// for the fuller design discussion.
type ReportFile struct {
	// Path is the absolute file path under DunnitDir().
	Path string
	// Kind is the report-family prefix parsed from the filename,
	// e.g. "review-week", "review-month", "standup", "status", "eod". Meant
	// for broad "what kind of thing is this" grouping/display --
	// callers wanting an exact covered date range for the "review-*"
	// family specifically already have review.go's more precise
	// reviewReportAnchorFromToken/listReviewReportsForPeriod.
	Kind string
	// Theme is the parsed theme suffix for "review-*" kind reports
	// (see review.go's reviewReportPath), "" if not applicable/not
	// present.
	Theme string
	// Audience is the intended audience when the filename records one,
	// currently Private or Shareable. Ordinary Review files are private.
	Audience string
	// Period is the period unit recovered from the covered-period token.
	// It is empty for an unrecognized legacy filename.
	Period summaryPeriod
	// Token is the filename's covered-period token, such as W39-2026.
	Token string
	// From and To are the nominal dates covered by the report. They are
	// separate from SavedAt because browsing needs the period being discussed,
	// not the last time someone edited the Markdown file.
	From, To time.Time
	// SavedAt is the file modification time.
	SavedAt time.Time
	// Date is retained as a compatibility alias for SavedAt.
	Date time.Time
}

// reportFileKinds is every known "<kind>-" filename prefix Dunnit
// currently saves reports under, longest-first so a longer, more
// specific prefix (e.g. "review-month") is matched before a shorter
// one that could otherwise falsely match part of it. Sourced from
// reviewReportKind (review.go, one per summaryPeriod) plus the other
// report kinds used by the app: standup, status, summary, and eod.
func reportFileKinds() []string {
	kinds := []string{
		reviewReportKind(periodQuarter), // "review-quarter" before "review-*" ambiguity
		reviewReportKind(periodMonth),
		reviewReportKind(periodWeek),
		reviewReportKind(periodYear),
		reviewReportKind(periodDay),
		"standup",
		"status",
		"summary",
		"eod",
	}
	return kinds
}

// parseReportFileName splits a report filename (base name, no
// directory, WITH ".md" extension) into (kind, theme, ok). Returns
// ok=false for filenames that don't start with one of
// reportFileKinds' known prefixes.
func parseReportFileName(base string) (kind, theme string, ok bool) {
	parsedKind, _, parsedTheme, _, parsedOK := parseReportFileParts(base)
	return parsedKind, parsedTheme, parsedOK
}

// parseReportFileParts decodes the filename vocabulary shared by every saved
// report. It deliberately accepts older day and summary tokens so the Saved
// Reports browser can present existing data alongside newer files.
func parseReportFileParts(base string) (kind, token, theme, audience string, ok bool) {
	if !strings.HasSuffix(base, ".md") {
		return "", "", "", "", false
	}
	nameNoExt := strings.TrimSuffix(base, ".md")
	for _, k := range reportFileKinds() {
		if nameNoExt != k && !strings.HasPrefix(nameNoExt, k+"-") {
			continue
		}
		rest := strings.TrimPrefix(nameNoExt, k+"-")
		if k == "status" {
			for _, candidate := range []string{"private", "shareable"} {
				if strings.HasSuffix(rest, "-"+candidate) {
					audience = strings.Title(candidate)
					rest = strings.TrimSuffix(rest, "-"+candidate)
					break
				}
			}
		} else if strings.HasPrefix(k, "review-") && strings.HasSuffix(rest, "-shareable") {
			audience = "Shareable"
			rest = strings.TrimSuffix(rest, "-shareable")
		}
		for _, th := range themeDisplayOrder {
			slug := themeFilenameSlug(th)
			if suffix := "-" + slug; strings.HasSuffix(rest, suffix) {
				theme = th
				rest = strings.TrimSuffix(rest, suffix)
				break
			}
		}
		return k, rest, theme, audience, true
	}
	return "", "", "", "", false
}

func reportPeriodForKind(kind string) (summaryPeriod, bool) {
	for _, period := range []summaryPeriod{periodDay, periodWeek, periodMonth, periodQuarter, periodYear} {
		if reviewReportKind(period) == kind {
			return period, true
		}
	}
	switch kind {
	case "standup", "eod":
		return periodDay, true
	case "status":
		return periodWeek, true
	case "summary":
		return "", false
	default:
		return "", false
	}
}

func reportAnchorFromToken(period summaryPeriod, token string) (time.Time, bool) {
	if anchor, ok := reviewReportAnchorFromToken(period, token); ok {
		return anchor, true
	}
	if period == periodDay {
		anchor, err := time.ParseInLocation("20060102", token, time.Local)
		return anchor, err == nil
	}
	return time.Time{}, false
}

// summaryPeriodAndAnchorFromToken recognizes the shared report token format
// without relying on the report's directory. This makes fiscal-year and
// cross-month reports filterable from one browser.
func summaryPeriodAndAnchorFromToken(token string) (summaryPeriod, time.Time, bool) {
	for _, period := range []summaryPeriod{periodDay, periodWeek, periodMonth, periodQuarter, periodYear} {
		if anchor, ok := reportAnchorFromToken(period, token); ok {
			return period, anchor, true
		}
	}
	return "", time.Time{}, false
}

func reportFileCoverage(kind, token string) (summaryPeriod, time.Time, time.Time, bool) {
	period, ok := reportPeriodForKind(kind)
	if !ok {
		period, anchor, parsed := summaryPeriodAndAnchorFromToken(token)
		if !parsed {
			return "", time.Time{}, time.Time{}, false
		}
		from, to := periodNominalRange(period, anchor)
		return period, from, to, true
	}
	anchor, ok := reportAnchorFromToken(period, token)
	if !ok {
		return "", time.Time{}, time.Time{}, false
	}
	from, to := periodNominalRange(period, anchor)
	return period, from, to, true
}

func reportStyle(r ReportFile) string {
	if r.Theme != "" {
		return themeDisplayNames[r.Theme]
	}
	if r.Kind == "status" {
		return "Legacy Status Report"
	}
	return reportKindLabel(r.Kind)
}

func reportCoverageLabel(r ReportFile) string {
	if r.From.IsZero() || r.To.IsZero() {
		return "Period unavailable"
	}
	if r.Period == periodDay {
		return r.From.Format("Mon Jan 2, 2006")
	}
	return periodLabel(LoadConfig(), r.Period, r.From) + " (" + shortDateRange(r.From, r.To) + ")"
}

func reportGroupKey(r ReportFile) string {
	from := r.From.Format("20060102")
	to := r.To.Format("20060102")
	if from == "00010101" || to == "00010101" {
		from = r.SavedAt.Format("20060102")
		to = from
	}
	return fmt.Sprintf("%s|%s|%s", logicalReportKind(r), from, to)
}

// logicalReportKind folds legacy standalone weekly status and period summary
// files into the corresponding Review family for browsing. The files remain
// separate variants and are never deleted; the grouping simply stops the
// browser from presenting two rows for one covered period.
func logicalReportKind(r ReportFile) string {
	if r.Kind == "status" && r.Period == periodWeek {
		return reviewReportKind(periodWeek)
	}
	if r.Kind == "summary" && r.Period != "" {
		return reviewReportKind(r.Period)
	}
	return r.Kind
}

// ReportGroup is one logical report period with its saved style/audience
// variants. Grouping keeps the browser deduplicated without deleting or
// hiding deliberately different versions.
type ReportGroup struct {
	Kind    string
	From    time.Time
	To      time.Time
	Reports []ReportFile
}

func groupReportFiles(files []ReportFile) []ReportGroup {
	groupsByKey := make(map[string]*ReportGroup)
	var groups []*ReportGroup
	for _, report := range files {
		key := reportGroupKey(report)
		group := groupsByKey[key]
		if group == nil {
			group = &ReportGroup{Kind: logicalReportKind(report), From: report.From, To: report.To}
			groupsByKey[key] = group
			groups = append(groups, group)
		}
		group.Reports = append(group.Reports, report)
	}
	for _, group := range groups {
		sort.SliceStable(group.Reports, func(i, j int) bool {
			return group.Reports[i].SavedAt.After(group.Reports[j].SavedAt)
		})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].From.IsZero() || groups[j].From.IsZero() {
			return groups[i].Reports[0].SavedAt.After(groups[j].Reports[0].SavedAt)
		}
		return groups[i].From.After(groups[j].From)
	})
	out := make([]ReportGroup, len(groups))
	for i, group := range groups {
		out[i] = *group
	}
	return out
}

// AllReportFiles walks the entire DunnitDir() tree for canonical report
// filenames, returning a ReportFile per match. No
// caching yet (unlike AllLedgerEntries) -- report file counts are
// expected to be orders of magnitude smaller than ledger line counts
// (one file per generated report vs one line per logged activity),
// so a fresh directory walk per call is cheap enough; revisit if
// this proves otherwise in practice.
func AllReportFiles() []ReportFile {
	var out []ReportFile

	root := DunnitDir()
	filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		kind, token, theme, audience, ok := parseReportFileParts(info.Name())
		if !ok {
			return nil
		}
		period, from, to, _ := reportFileCoverage(kind, token)
		out = append(out, ReportFile{
			Path:     path,
			Kind:     kind,
			Theme:    theme,
			Audience: audience,
			Period:   period,
			Token:    token,
			From:     from,
			To:       to,
			SavedAt:  info.ModTime(),
			Date:     info.ModTime(),
		})
		return nil
	})

	return out
}

// ReportBody reads a ReportFile's full markdown content from disk.
func ReportBody(r ReportFile) (string, error) {
	data, err := os.ReadFile(r.Path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
