package dun

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractPeople(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{"examples", "Followed up with @Brandon about #12345 token", []string{"@Brandon"}},
		{"multiple and duplicate case", "Helped @Surbhi and @brandon; later @Brandon", []string{"@Surbhi", "@brandon"}},
		{"hyphen and underscore", "Met @Mary-Jane and @ops_team", []string{"@Mary-Jane", "@ops_team"}},
		{"ordinary ampersand", "AT&T, email@example.com, and apples & oranges", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractPeople(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractPeople(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestContainsPersonFold(t *testing.T) {
	people := []string{"@Brandon", "@Surbhi"}
	if !containsPersonFold(people, "@brandon") {
		t.Fatal("containsPersonFold should ignore person-name case")
	}
	if containsPersonFold(people, "@Taylor") {
		t.Fatal("containsPersonFold matched an absent person")
	}
}

func TestParseLedgerEntryExtractsPeople(t *testing.T) {
	entry, ok := parseLedgerEntry(
		"[09:15:00] DONE Helped @Surbhi get past a #snap issue",
		time.Date(2026, 9, 18, 0, 0, 0, 0, time.Local), "ledger.txt", 0)
	if !ok {
		t.Fatal("parseLedgerEntry rejected a valid ledger line")
	}
	if !reflect.DeepEqual(entry.People, []string{"@Surbhi"}) {
		t.Fatalf("People = %v, want [@Surbhi]", entry.People)
	}
}

func TestLedgerQueryMatchesPeopleCaseInsensitively(t *testing.T) {
	entry := LedgerEntry{People: []string{"@Brandon", "@Surbhi"}}
	tests := []struct {
		name  string
		query LedgerQuery
		want  bool
	}{
		{"one person", LedgerQuery{People: []string{"@brandon"}}, true},
		{"all people", LedgerQuery{People: []string{"@brandon", "@SURBHI"}}, true},
		{"missing person", LedgerQuery{People: []string{"@Taylor"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.query.Matches(entry); got != tt.want {
				t.Fatalf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
