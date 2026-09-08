package dun

import "testing"

func TestHasRealLedgerContent(t *testing.T) {
	cases := map[string]bool{
		"":                          false,
		"# ledger-20260903.txt\n":   false,
		"# ledger-20260903.txt\n\n": false,
		"# ledger-20260903.txt\n[09:00] DONE thing\n": true,
		"[09:00] DONE thing\n":                        true,
	}
	for in, want := range cases {
		if got := hasRealLedgerContent(in); got != want {
			t.Errorf("hasRealLedgerContent(%q) = %v, want %v", in, got, want)
		}
	}
}
