package dun

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
)

func TestReportFilenameUsesCoveredPeriod(t *testing.T) {
	got := reportFilename("status", "W38-2026", "")
	if got != "status-W38-2026.md" {
		t.Fatalf("reportFilename() = %q, want status-W38-2026.md", got)
	}

	got = reportFilename("review-week", "W37-2026", ThemeStatusReport)
	if got != "review-week-W37-2026-statusreport.md" {
		t.Fatalf("reportFilename() with theme = %q", got)
	}

	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	if got, want := filepath.Base(statusReportPath(anchor, "Private")), "status-W38-2026-private.md"; got != want {
		t.Fatalf("statusReportPath() base = %q, want %q", got, want)
	}
}

func TestNormalizeReportUsesCanonicalTitle(t *testing.T) {
	got := normalizeReport("# Generated summary\n\n## Accomplishments\n- Shipped it", "Standup Update — Wed Sep 23, 2026")
	want := "# Standup Update — Wed Sep 23, 2026\n\n## Accomplishments\n- Shipped it\n"
	if got != want {
		t.Fatalf("normalizeReport() = %q, want %q", got, want)
	}
}

func TestStatusReportTitleIncludesAudienceAndWeekRange(t *testing.T) {
	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	if got, want := statusReportTitle(anchor, "Shareable"), "Shareable Status Report (W38 — Sep 14-20)"; got != want {
		t.Fatalf("statusReportTitle() = %q, want %q", got, want)
	}
}

func TestReportPathsUsePeriodDirectories(t *testing.T) {
	withTempDunnitDir(t)
	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)

	cases := []struct {
		name string
		path string
		want string
	}{
		{"standup", dailyReportPathForKind("standup", anchor), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"status", statusReportPath(anchor, "Private"), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"week summary", summaryReportPath(periodWeek, anchor), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"month summary", summaryReportPath(periodMonth, anchor), filepath.Join(DunnitDir(), "2026", "Sep")},
		{"quarter summary", summaryReportPath(periodQuarter, anchor), filepath.Join(DunnitDir(), "2026")},
		{"year summary", summaryReportPath(periodYear, anchor), filepath.Join(DunnitDir(), "2026")},
		{"day review", reviewReportPath(periodDay, anchor, ThemePersonalNotes), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"week review", reviewReportPath(periodWeek, anchor, ThemePersonalNotes), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"month review", reviewReportPath(periodMonth, anchor, ThemePersonalNotes), filepath.Join(DunnitDir(), "2026", "Sep")},
		{"quarter review", reviewReportPath(periodQuarter, anchor, ThemePersonalNotes), filepath.Join(DunnitDir(), "2026")},
		{"year review", reviewReportPath(periodYear, anchor, ThemePersonalNotes), filepath.Join(DunnitDir(), "2026")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := filepath.Dir(tc.path); got != tc.want {
				t.Fatalf("report path directory = %q, want %q (full path %q)", got, tc.want, tc.path)
			}
		})
	}
}

func TestParseReportFileNameUsesCanonicalKinds(t *testing.T) {
	cases := []struct {
		name, filename, wantKind, wantTheme string
		wantOK                              bool
	}{
		{"standup", "standup-Mon-20260914.md", "standup", "", true},
		{"status", "status-W38-2026-private.md", "status", "", true},
		{"summary", "summary-W39-2026.md", "summary", "", true},
		{"eod", "eod-Mon-20260914.md", "eod", "", true},
		{"review", "review-week-W37-2026-statusreport.md", "review-week", ThemeStatusReport, true},
		{"legacy dsu", "dsu-20260914.md", "", "", false},
		{"old summary", "summary-20260914.md", "summary", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, theme, ok := parseReportFileName(tc.filename)
			if kind != tc.wantKind || theme != tc.wantTheme || ok != tc.wantOK {
				t.Fatalf("parseReportFileName(%q) = %q, %q, %v; want %q, %q, %v", tc.filename, kind, theme, ok, tc.wantKind, tc.wantTheme, tc.wantOK)
			}
		})
	}
}

func TestSummaryReportPathsUsePeriodTokens(t *testing.T) {
	anchor := time.Date(2026, time.September, 25, 0, 0, 0, 0, time.Local)
	cases := []struct {
		period summaryPeriod
		want   string
	}{
		{periodDay, "summary-Fri-20260925.md"},
		{periodWeek, "summary-W39-2026.md"},
		{periodMonth, "summary-Sep-2026.md"},
		{periodQuarter, "summary-Q3-2026.md"},
		{periodYear, "summary-FY2026.md"},
	}
	for _, tc := range cases {
		t.Run(string(tc.period), func(t *testing.T) {
			if got := filepath.Base(summaryReportPath(tc.period, anchor)); got != tc.want {
				t.Fatalf("summaryReportPath(%s) = %q, want %q", tc.period, got, tc.want)
			}
		})
	}
}

func TestReviewReportFilenameParts(t *testing.T) {
	cases := []struct {
		name, filename, wantToken, wantTheme string
	}{
		{"canonical", "review-week-W37-2026-personalnotes.md", "W37-2026", ThemePersonalNotes},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			token, theme, ok := reviewReportFilenameParts(periodWeek, filepath.Join("reports", tc.filename))
			if !ok || token != tc.wantToken || theme != tc.wantTheme {
				t.Fatalf("reviewReportFilenameParts() = %q, %q, %v", token, theme, ok)
			}
		})
	}
}

func TestFiscalYearSettingsChangeAnnualPeriodAndToken(t *testing.T) {
	withTempDunnitDir(t)
	cfg := defaultConfig()
	cfg.YearStartMonth = 7
	cfg.YearEndMonth = 6
	if err := writeConfig(cfg); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}

	anchor := time.Date(2025, time.September, 25, 0, 0, 0, 0, time.Local)
	from, to := periodNominalRange(periodYear, anchor)
	if got, want := from.Format("2006-01-02"), "2025-07-01"; got != want {
		t.Fatalf("fiscal year start = %q, want %q", got, want)
	}
	if got, want := to.Format("2006-01-02"), "2026-06-30"; got != want {
		t.Fatalf("fiscal year end = %q, want %q", got, want)
	}
	if got, want := filepath.Base(summaryReportPath(periodYear, anchor)), "summary-FY2026.md"; got != want {
		t.Fatalf("fiscal summary filename = %q, want %q", got, want)
	}
	if got, want := filepath.Dir(yearlyReportPath(anchor, ThemeFormalReport)), filepath.Join(DunnitDir(), "2026"); got != want {
		t.Fatalf("fiscal report directory = %q, want %q", got, want)
	}
}

func TestConfigureReportLocalLinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "foo.txt")
	if err := writeReportFile(path, "notes"); err != nil {
		t.Fatalf("writeReportFile: %v", err)
	}

	link := &widget.HyperlinkSegment{
		Text: "foo",
		URL:  &url.URL{Scheme: "cc3", Opaque: "docs/foo.txt"},
	}
	richText := widget.NewRichText(link)
	configureReportLocalLinkSegments(richText.Segments, Config{
		FileAliases: map[string]string{"cc3": root},
	})

	if got, want := link.URL.String(), localFileURL(path).String(); got != want {
		t.Fatalf("report link URL = %q, want %q", got, want)
	}
	if link.OnTapped == nil {
		t.Fatal("report local link has no editor action")
	}
}
