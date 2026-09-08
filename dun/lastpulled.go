package dun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// lastPulledPath is where per-tag "last agenda pulled" timestamps are
// stored (FR-12) -- a small JSON file under DunnitDir(), not mixed
// into config.toml since it's transient/derived state rather than
// user preference.
func lastPulledPath() string {
	return filepath.Join(DunnitDir(), "last_pulled.json")
}

// loadLastPulled reads the tag -> last-pulled-time map, returning an
// empty map if the file doesn't exist yet or fails to parse (treated
// as "never pulled before" rather than a hard error, since this is
// convenience state, not the ledger itself).
func loadLastPulled() map[string]time.Time {
	out := map[string]time.Time{}
	data, err := os.ReadFile(lastPulledPath())
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

// saveLastPulled writes m to lastPulledPath(), best-effort (errors are
// swallowed -- a failure to persist this marker shouldn't block using
// Meeting Prep, it just means the next pull won't narrow by it).
func saveLastPulled(m map[string]time.Time) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(DunnitDir(), 0755)
	_ = os.WriteFile(lastPulledPath(), data, 0644)
}

// markPulled records now as the last-pulled time for tag, persisting
// immediately.
func markPulled(tag string) {
	m := loadLastPulled()
	m[tag] = time.Now()
	saveLastPulled(m)
}
