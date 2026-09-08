package dun

import "strings"

// ReportSearchResult is one report file whose body matched a search
// query, with a short surrounding excerpt for context (avoids
// dumping the whole report body into a results list).
type ReportSearchResult struct {
	Report  ReportFile
	Excerpt string
}

// reportExcerptRadius is how many characters of context to include
// on each side of a match in ReportSearchResult.Excerpt.
const reportExcerptRadius = 80

// SearchReports scans every report file's body (via AllReportFiles +
// ReportBody) for query as a case-insensitive substring, returning
// one ReportSearchResult per matching file (first match's surrounding
// text only, even if a report matches multiple times -- this is a
// "which reports mention X" browse, not an every-occurrence
// full-text index). Empty query matches nothing, same convention as
// searchLedgers.
func SearchReports(query string) []ReportSearchResult {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	lowerQuery := strings.ToLower(query)
	var out []ReportSearchResult
	for _, r := range AllReportFiles() {
		body, err := ReportBody(r)
		if err != nil {
			continue
		}
		lowerBody := strings.ToLower(body)
		idx := strings.Index(lowerBody, lowerQuery)
		if idx < 0 {
			continue
		}
		out = append(out, ReportSearchResult{Report: r, Excerpt: excerptAround(body, idx, len(query))})
	}
	return out
}

// excerptAround returns a short snippet of text centered on the
// match at byte offset idx (length matchLen), padded by
// reportExcerptRadius characters on each side, with "…" markers if
// the excerpt was truncated from the full text on either end.
func excerptAround(text string, idx, matchLen int) string {
	start := idx - reportExcerptRadius
	prefix := "\u2026"
	if start <= 0 {
		start = 0
		prefix = ""
	}
	end := idx + matchLen + reportExcerptRadius
	suffix := "\u2026"
	if end >= len(text) {
		end = len(text)
		suffix = ""
	}
	excerpt := text[start:end]
	// Collapse newlines to spaces so each result renders as one line.
	excerpt = strings.ReplaceAll(excerpt, "\n", " ")
	return prefix + excerpt + suffix
}
