package dun

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeTagDefinitionName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "bare", in: "foo", want: "foo", ok: true},
		{name: "hash-prefixed", in: "#foo", want: "foo", ok: true},
		{name: "ticket", in: "#SCRUM-12345", want: "SCRUM-12345", ok: true},
		{name: "colon", in: "pts:3", want: "pts:3", ok: true},
		{name: "empty", in: " ", ok: false},
		{name: "spaces", in: "foo bar", ok: false},
		{name: "double-hash", in: "##foo", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTagDefinitionName(tt.in)
			if (err == nil) != tt.ok {
				t.Fatalf("normalizeTagDefinitionName(%q) error = %v, want success %v", tt.in, err, tt.ok)
			}
			if tt.ok && got != tt.want {
				t.Fatalf("normalizeTagDefinitionName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTagDefinitionRoundTripAndAliasLookup(t *testing.T) {
	withTempDunnitDir(t)
	want := TagDefinition{
		Name:        "foo",
		Title:       "Foo",
		Summary:     "A short Foo summary.",
		Description: "## Foo\n\nA longer description.",
		URL:         "https://example.com/foo",
		Kind:        "project",
		Status:      "active",
		Aliases:     []string{"oldfoo", "#legacy"},
		Parent:      "company",
	}
	if err := SaveTagDefinition(want); err != nil {
		t.Fatalf("SaveTagDefinition: %v", err)
	}

	got, found, err := LoadTagDefinition("#legacy")
	if err != nil {
		t.Fatalf("LoadTagDefinition: %v", err)
	}
	if !found {
		t.Fatal("LoadTagDefinition did not find alias")
	}
	want.Name = "foo"
	want.Aliases = []string{"oldfoo", "legacy"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LoadTagDefinition = %#v, want %#v", got, want)
	}

	data, err := os.ReadFile(filepath.Join(DunnitDir(), "tags.toml"))
	if err != nil {
		t.Fatalf("read tags.toml: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, `[tags.foo]`) || strings.Contains(text, "name =") {
		t.Fatalf("tags.toml has unexpected shape:\n%s", text)
	}
}

func TestSaveTagDefinitionPreservesOtherDefinitions(t *testing.T) {
	withTempDunnitDir(t)
	if err := SaveTagDefinition(TagDefinition{Name: "first", Summary: "one"}); err != nil {
		t.Fatalf("save first definition: %v", err)
	}
	if err := SaveTagDefinition(TagDefinition{Name: "second", Summary: "two"}); err != nil {
		t.Fatalf("save second definition: %v", err)
	}

	first, found, err := LoadTagDefinition("first")
	if err != nil || !found {
		t.Fatalf("load first definition = %#v, %v, found %v", first, err, found)
	}
	if first.Summary != "one" {
		t.Fatalf("first definition summary = %q, want %q", first.Summary, "one")
	}
}
