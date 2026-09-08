package dun

import (
	"sync"
	"time"
)

// ledgerIndex holds a scanned, fully-parsed view of every ledger
// line under DunnitDir(), refreshed at most every ledgerIndexTTL
// rather than rescanning on every call -- same caching philosophy as
// tags.go's tagCache, just holding parsed LedgerEntry structs instead
// of bare tag strings. A sync.RWMutex (rather than tagCache's plain
// Mutex) is used since this is expected to be read far more often
// (every search/tag/trend/navigator call) than written.
type ledgerIndex struct {
	mu        sync.RWMutex
	entries   []LedgerEntry
	scannedAt time.Time
}

const ledgerIndexTTL = 5 * time.Minute

var globalLedgerIndex ledgerIndex

// AllLedgerEntries returns the cached, parsed view of every ledger
// line across all history (oldest first), rescanning if the cache is
// empty or stale. This is the single shared read path search/tags/
// trend/navigator code should use instead of independently walking
// allLedgerFiles() + re-parsing lines themselves.
func AllLedgerEntries() []LedgerEntry {
	globalLedgerIndex.mu.RLock()
	if globalLedgerIndex.entries != nil && time.Since(globalLedgerIndex.scannedAt) <= ledgerIndexTTL {
		defer globalLedgerIndex.mu.RUnlock()
		return globalLedgerIndex.entries
	}
	globalLedgerIndex.mu.RUnlock()

	entries := scanAllLedgerEntries()

	globalLedgerIndex.mu.Lock()
	globalLedgerIndex.entries = entries
	globalLedgerIndex.scannedAt = time.Now()
	globalLedgerIndex.mu.Unlock()

	return entries
}

// InvalidateLedgerIndex forces the next AllLedgerEntries() call to
// rescan -- call alongside InvalidateTagCache() (see
// InvalidateLedgerCaches) anywhere a write might have changed ledger
// contents (new entry, edit, undo, category rewrite, etc).
func InvalidateLedgerIndex() {
	globalLedgerIndex.mu.Lock()
	defer globalLedgerIndex.mu.Unlock()
	globalLedgerIndex.entries = nil
}

// InvalidateLedgerCaches invalidates every cache derived from ledger
// file contents (currently: the tag cache and the ledger entry
// index) in one call, so write-path call sites don't need to
// remember to update both separately as more caches are added.
func InvalidateLedgerCaches() {
	InvalidateTagCache()
	InvalidateLedgerIndex()
}

// scanAllLedgerEntries walks every ledger-*.txt file under
// DunnitDir() (allLedgerFiles, summarize.go) and parses every line
// into a LedgerEntry, skipping lines that don't parse (same
// leniency parseLedgerLine's other callers already have). Result is
// sorted oldest-first by Date (ties broken by original file-walk/
// line order, which is already roughly chronological within a day).
func scanAllLedgerEntries() []LedgerEntry {
	var out []LedgerEntry
	for _, path := range allLedgerFiles() {
		date := ledgerFileDate(path)
		if date == nil {
			continue
		}
		for i, line := range readLedgerLinesFrom(path) {
			entry, ok := parseLedgerEntry(line, *date, path, i)
			if !ok {
				continue
			}
			out = append(out, entry)
		}
	}
	// Stable sort oldest-first by Date -- allLedgerFiles' walk order
	// already groups/roughly-orders by year/week/day directory, but
	// isn't guaranteed globally chronological (e.g. across
	// differently-named week directories), so make it explicit here
	// rather than relying on filesystem walk order.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Date.Before(out[j-1].Date); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
