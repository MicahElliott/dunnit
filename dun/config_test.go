package dun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartOfDayDoesNotReplaceUnreadableConfig(t *testing.T) {
	withTempDunnitDir(t)

	path := filepath.Join(DunnitDir(), "config.toml")
	original := strings.Join([]string{
		`day_start = "09:15"`,
		`day_end = [this is not valid TOML]`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	markStartOfDayRun()

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after carry-forward: %v", err)
	}
	if string(after) != original {
		t.Fatalf("unreadable config was replaced:\n got %q\nwant %q", after, original)
	}
}

func TestStartOfDayWritesLoadedConfigValues(t *testing.T) {
	withTempDunnitDir(t)

	path := filepath.Join(DunnitDir(), "config.toml")
	original := strings.Join([]string{
		`day_start = "09:15"`,
		`day_end = "18:00"`,
		`nudge_interval_minutes = 45`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	markStartOfDayRun()

	cfg := LoadConfig()
	if cfg.DayStart != "09:15" || cfg.DayEnd != "18:00" || cfg.NudgeIntervalMinutes != 45 {
		t.Fatalf("loaded config values changed: %+v", cfg)
	}
}
