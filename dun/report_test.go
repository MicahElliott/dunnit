package dun

import (
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2/widget"
)

func TestReportFilenameUsesCoveredPeriodAndGenerationDate(t *testing.T) {
	generated := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.Local)
	got := reportFilename("status", "w38", "", generated)
	if got != "status-w38-20260914.md" {
		t.Fatalf("reportFilename() = %q, want status-w38-20260914.md", got)
	}

	got = reportFilename("review-week", "20260907", ThemeStatusReport, generated)
	if got != "review-week-20260907-20260914-status_report.md" {
		t.Fatalf("reportFilename() with theme = %q", got)
	}

	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	if got, want := filepath.Base(statusReportPath(anchor, generated)), "status-w38-20260914.md"; got != want {
		t.Fatalf("statusReportPath() base = %q, want %q", got, want)
	}
}

func TestReportPathsUsePeriodDirectories(t *testing.T) {
	withTempDunnitDir(t)
	anchor := time.Date(2026, time.September, 16, 0, 0, 0, 0, time.Local)
	generated := time.Date(2026, time.September, 16, 12, 0, 0, 0, time.Local)

	cases := []struct {
		name string
		path string
		want string
	}{
		{"standup", weeklyReportPathForKind("standup", anchor, generated), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
		{"status", statusReportPath(anchor, generated), filepath.Join(DunnitDir(), "2026", "Sep", "w38")},
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
		{"standup", "standup-w38-20260914.md", "standup", "", true},
		{"status", "status-w38-20260914.md", "status", "", true},
		{"eod", "eod-Mon-20260914.md", "eod", "", true},
		{"review", "review-week-20260907-20260914-status_report.md", "review-week", ThemeStatusReport, true},
		{"legacy dsu", "dsu-20260914.md", "", "", false},
		{"legacy summary", "summary-20260914.md", "", "", false},
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

func TestReviewReportFilenameParts(t *testing.T) {
	path := filepath.Join("reports", "review-week-20260907-20260914-personal_notes.md")
	token, theme, ok := reviewReportFilenameParts(periodWeek, path)
	if !ok || token != "20260907" || theme != ThemePersonalNotes {
		t.Fatalf("reviewReportFilenameParts() = %q, %q, %v", token, theme, ok)
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
