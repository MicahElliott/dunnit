package dun

import (
	"reflect"
	"testing"
	"time"
)

func TestExtractFlags(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "all flags", text: "!! clarify ?? follow up @@ continue ++", want: []string{"!!", "??", "@@", "++"}},
		{name: "flag beside person", text: "@@ ask @Surbhi", want: []string{"@@"}},
		{name: "ordinary punctuation", text: "Do the thing!!", want: nil},
		{name: "embedded token", text: "update++ later", want: nil},
		{name: "duplicate flag", text: "!! important !!", want: []string{"!!"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFlags(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractFlags(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestStripAndToggleFlags(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "leading", text: "!! renew certificate #dunnit", want: "renew certificate #dunnit"},
		{name: "middle", text: "renew !! certificate ?? #dunnit", want: "renew certificate #dunnit"},
		{name: "trailing", text: "renew certificate @@", want: "renew certificate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripFlags(tt.text); got != tt.want {
				t.Fatalf("stripFlags(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}

	text := toggleFlag("renew certificate #dunnit", "!!", true)
	if text != "!! renew certificate #dunnit" {
		t.Fatalf("toggleFlag(enable) = %q", text)
	}
	text = toggleFlag(text, "!!", false)
	if text != "renew certificate #dunnit" {
		t.Fatalf("toggleFlag(disable) = %q", text)
	}
}

func TestParseLedgerEntryExtractsFlagsWithoutChangingText(t *testing.T) {
	entry, ok := parseLedgerEntry("[09:00] TODO renew certificate !! #dunnit", time.Now(), "ledger.txt", 0)
	if !ok {
		t.Fatal("parseLedgerEntry rejected a flagged entry")
	}
	if entry.Text != "renew certificate !! #dunnit" {
		t.Fatalf("entry.Text = %q, want original text", entry.Text)
	}
	if !reflect.DeepEqual(entry.Flags, []string{"!!"}) {
		t.Fatalf("entry.Flags = %v, want [!!]", entry.Flags)
	}
}

func TestLifecycleIdentityIgnoresFlags(t *testing.T) {
	want := openItemKey("TODO", "renew certificate")
	if got := openItemKey("DOING", "!! renewing certificate ??"); got != want {
		t.Fatalf("flagged lifecycle key = %q, want %q", got, want)
	}
	if got := inflectLifecycleText("!! renew certificate @@", "DOING"); got != "!! @@ renewing certificate" {
		t.Fatalf("flagged lifecycle inflection = %q", got)
	}
}
