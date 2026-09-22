package dun

import (
	"strings"
	"testing"
	"time"
)

func TestFormatCategoryHistogramCollapseCarryForwardCopies(t *testing.T) {
	first := time.Date(2026, 9, 21, 0, 0, 0, 0, time.Local)
	entries := []LedgerEntry{
		{Date: first, Category: "TODO", Text: "follow up #alpha"},
		{Date: first.AddDate(0, 0, 1), Category: "TODO", Text: "follow up #alpha s/2026-09-21"},
	}
	got := formatCategoryHistogram(entries)
	if !strings.Contains(got, "TODO") || !strings.HasSuffix(strings.TrimSpace(got), "1") {
		t.Fatalf("formatCategoryHistogram() = %q, want one TODO", got)
	}
}
