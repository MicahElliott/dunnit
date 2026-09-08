package dun

import (
	"os"
	"path/filepath"
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
	// e.g. "review-week", "review-month", "dsu", "summary". Meant
	// for broad "what kind of thing is this" grouping/display --
	// callers wanting an exact covered date range for the "review-*"
	// family specifically already have review.go's more precise
	// reviewReportAnchorFromToken/listReviewReportsForPeriod.
	Kind string
	// Theme is the parsed theme suffix for "review-*" kind reports
	// (see review.go's reviewReportPath), "" if not applicable/not
	// present.
	Theme string
	// Date is the file's modification time -- used as a stand-in for
	// "when was this generated/last saved", since each report kind
	// encodes its own covered period differently in its filename
	// (reviewReportDateToken vs periodReportPath's plain
	// time.Format-based token) and ReportFile only needs a
	// reasonably-ordered "when" for browsing/sorting, not the exact
	// covered range.
	Date time.Time
}

// reportFileKinds is every known "<kind>-" filename prefix Dunnit
// currently saves reports under, longest-first so a longer, more
// specific prefix (e.g. "review-month") is matched before a shorter
// one that could otherwise falsely match part of it. Sourced from
// reviewReportKind (review.go, one per summaryPeriod) plus the other
// ad hoc kinds seen in periodReportPath call sites (standup.go's
// "dsu", som.go's "som") and dailysummary.go's "summary".
func reportFileKinds() []string {
	kinds := []string{
		reviewReportKind(periodQuarter), // "review-quarter" before "review-*" ambiguity
		reviewReportKind(periodMonth),
		reviewReportKind(periodWeek),
		reviewReportKind(periodYear),
		reviewReportKind(periodDay),
		"dsu",
		"som",
		"summary",
	}
	return kinds
}

// parseReportFileName splits a report filename (base name, no
// directory, WITH ".md" extension) into (kind, theme, ok). Returns
// ok=false for filenames that don't start with one of
// reportFileKinds' known prefixes.
func parseReportFileName(base string) (kind, theme string, ok bool) {
	if !strings.HasSuffix(base, ".md") {
		return "", "", false
	}
	nameNoExt := strings.TrimSuffix(base, ".md")
	for _, k := range reportFileKinds() {
		if nameNoExt != k && !strings.HasPrefix(nameNoExt, k+"-") {
			continue
		}
		rest := strings.TrimPrefix(nameNoExt, k+"-")
		theme = ""
		for _, th := range themeDisplayOrder {
			if suffix := "-" + th; strings.HasSuffix(rest, suffix) || rest == th {
				theme = th
				break
			}
		}
		return k, theme, true
	}
	return "", "", false
}

// AllReportFiles walks DunnitDir() (root-level "<kind>-<token>[-<theme>].md"
// files, e.g. review-*/dsu-*/som-*) plus every ledger-adjacent
// "summary-*.md" daily summary doc (dailysummary.go's per-day-
// directory convention), returning a ReportFile per match. No
// caching yet (unlike AllLedgerEntries) -- report file counts are
// expected to be orders of magnitude smaller than ledger line counts
// (one file per generated report vs one line per logged activity),
// so a fresh directory walk per call is cheap enough; revisit if
// this proves otherwise in practice.
func AllReportFiles() []ReportFile {
	var out []ReportFile

	root := DunnitDir()
	rootEntries, err := os.ReadDir(root)
	if err == nil {
		for _, entry := range rootEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			kind, theme, ok := parseReportFileName(entry.Name())
			if !ok {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			out = append(out, ReportFile{
				Path:  filepath.Join(root, entry.Name()),
				Kind:  kind,
				Theme: theme,
				Date:  info.ModTime(),
			})
		}
	}

	// Daily summary docs live alongside their ledger file (one level
	// down, in a year/w<week>-<month> directory), not at DunnitDir()'s
	// root -- walk those separately.
	filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil
		}
		name := info.Name()
		if !strings.HasPrefix(name, "summary-") || !strings.HasSuffix(name, ".md") {
			return nil
		}
		if filepath.Dir(path) == root {
			return nil // already picked up by the root-level pass above
		}
		out = append(out, ReportFile{
			Path: path,
			Kind: "summary",
			Date: info.ModTime(),
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
