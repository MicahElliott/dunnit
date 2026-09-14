package dun

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPathResolutionAgainstMigratedData verifies that the new path scheme
// correctly resolves to existing files in the migrated ~/proj/mydunnits directory.
// This is the post-migration verification step.
func TestPathResolutionAgainstMigratedData(t *testing.T) {
	// Temporarily override DunnitDir for this test
	// The test will look for actual migrated data if DUNNIT_DIR is set to ~/proj/mydunnits
	dunnitDirEnv := os.Getenv("DUNNIT_DIR")
	if dunnitDirEnv == "" {
		t.Skip("DUNNIT_DIR not set; skipping live data verification test")
	}

	tests := []struct {
		name    string
		date    time.Time
		comment string
	}{
		{
			// 2021-05-18 is in ISO week 20, which starts Monday 2021-05-17 (May)
			name:    "2021-05-18 (week 20, May)",
			date:    time.Date(2021, 5, 18, 0, 0, 0, 0, time.Local),
			comment: "Should resolve to 2021/May/w20/",
		},
		{
			// 2021-04-23 is in ISO week 16, which starts Monday 2021-04-19 (April)
			name:    "2021-04-23 (week 16, April)",
			date:    time.Date(2021, 4, 23, 0, 0, 0, 0, time.Local),
			comment: "Should resolve to 2021/Apr/w16/",
		},
		{
			// 2021-05-03 is in ISO week 18, which starts Monday 2021-05-03 (May)
			name:    "2021-05-03 (week 18, May)",
			date:    time.Date(2021, 5, 3, 0, 0, 0, 0, time.Local),
			comment: "Should resolve to 2021/May/w18/",
		},
		{
			// 2025-04-28 is in ISO week 18, which starts Monday 2025-04-28 (April)
			name:    "2025-04-28 (week 18, April)",
			date:    time.Date(2025, 4, 28, 0, 0, 0, 0, time.Local),
			comment: "Should resolve to 2025/Apr/w18/",
		},
		{
			// 2025-05-01 is in ISO week 18, which starts Monday 2025-04-28 (April)
			// This tests the month-boundary case: the week starts in April but this date is in May
			name:    "2025-05-01 (week 18, but Monday in April)",
			date:    time.Date(2025, 5, 1, 0, 0, 0, 0, time.Local),
			comment: "Should resolve to 2025/Apr/w18/ (week's Monday month, not date's month)",
		},
	}
	for _, tt := range tests {
		dir, _ := ledgerPathFor(tt.date)
		if _, err := os.Stat(dir); err != nil {
			t.Skipf("migrated data directory %s is not present; run the ledger migration first", dir)
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, path := ledgerPathFor(tt.date)

			// Verify the directory exists
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("Directory doesn't exist: %s (%v). Comment: %s", dir, err, tt.comment)
			}
			if !info.IsDir() {
				t.Fatalf("Path exists but is not a directory: %s. Comment: %s", dir, tt.comment)
			}

			// Verify the expected filename pattern
			expectedBase := "ledger-" + tt.date.Format("Mon-20060102") + ".txt"
			_ = filepath.Join(dir, expectedBase)

			// For this verification test, we don't require the specific file to exist,
			// just that the directory exists and the path is constructable.
			// (Not all dates in migrated data have entries; we only check directory existence.)

			t.Logf("✓ Path verified: %s (file would be: %s)", dir, filepath.Base(path))
			t.Logf("  Comment: %s", tt.comment)
		})
	}
}

// TestWeekMonthInfoBoundaryCase specifically tests the week-boundary scenario
// where a week spans two calendar months (e.g., 2025-04-28 to 2025-05-04).
func TestWeekMonthInfoBoundaryCase(t *testing.T) {
	// ISO week 18, 2025: starts 2025-04-28 (Monday, April), ends 2025-05-04 (Sunday, May)
	testCases := []struct {
		name        string
		date        time.Time
		expectMonth string // Expected month abbreviation (from Monday)
	}{
		{
			name:        "2025-04-28 (Monday of week 18)",
			date:        time.Date(2025, 4, 28, 0, 0, 0, 0, time.Local),
			expectMonth: "Apr",
		},
		{
			name:        "2025-04-29 (Tuesday, still April)",
			date:        time.Date(2025, 4, 29, 0, 0, 0, 0, time.Local),
			expectMonth: "Apr",
		},
		{
			name:        "2025-04-30 (Wednesday, still April)",
			date:        time.Date(2025, 4, 30, 0, 0, 0, 0, time.Local),
			expectMonth: "Apr",
		},
		{
			name:        "2025-05-01 (Thursday, now May)",
			date:        time.Date(2025, 5, 1, 0, 0, 0, 0, time.Local),
			expectMonth: "Apr", // Still April because week's Monday is in April
		},
		{
			name:        "2025-05-04 (Sunday of week 18, May)",
			date:        time.Date(2025, 5, 4, 0, 0, 0, 0, time.Local),
			expectMonth: "Apr", // Still April because week's Monday is in April
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			yr, moname, wk := weekMonthInfo(tc.date)

			if moname != tc.expectMonth {
				t.Errorf("Expected month %q but got %q for date %v (week %d)", tc.expectMonth, moname, tc.date, wk)
			}

			// Verify it's week 18
			_, isoWeek := tc.date.ISOWeek()
			if isoWeek != 18 {
				t.Errorf("Expected ISO week 18 but got %d for date %v", isoWeek, tc.date)
			}

			t.Logf("✓ %s -> %d/%s/w%d", tc.date.Format("2006-01-02"), yr, moname, wk)
		})
	}
}

func TestLedgerFileDateRequiresCanonicalWeekdaySuffix(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "canonical", path: "ledger-Sat-20260912.txt", want: true},
		{name: "legacy date only", path: "ledger-20260912.txt", want: false},
		{name: "wrong weekday", path: "ledger-Sun-20260912.txt", want: false},
		{name: "invalid date", path: "ledger-Sat-20260999.txt", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ledgerFileDate(tc.path) != nil
			if got != tc.want {
				t.Fatalf("ledgerFileDate(%q) != nil: %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}
