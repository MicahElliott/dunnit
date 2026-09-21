package dun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEntryLinks(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		wantText []string
		wantURLs []string
	}{
		{
			name:     "markdown label",
			text:     "fixed [Jira #74750](https://acme.atlassian.net/browse/ABC-74750)",
			wantText: []string{"Jira #74750"},
			wantURLs: []string{"https://acme.atlassian.net/browse/ABC-74750"},
		},
		{
			name:     "bare URL with punctuation",
			text:     "see https://github.com/acme/dunnit/pull/12.",
			wantText: []string{"GitHub PR #12"},
			wantURLs: []string{"https://github.com/acme/dunnit/pull/12"},
		},
		{
			name:     "multiple links",
			text:     "[chat](https://teams.microsoft.com/l/meetup-join/x) and https://docs.google.com/document/d/abc",
			wantText: []string{"chat", "Google Doc"},
			wantURLs: []string{"https://teams.microsoft.com/l/meetup-join/x", "https://docs.google.com/document/d/abc"},
		},
		{
			name:     "unknown URL",
			text:     "https://example.com/path",
			wantText: []string{"[link]"},
			wantURLs: []string{"https://example.com/path"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseEntryLinks(tc.text)
			if len(got) != len(tc.wantText) {
				t.Fatalf("found %d links, want %d: %#v", len(got), len(tc.wantText), got)
			}
			for i, link := range got {
				if link.Text != tc.wantText[i] || link.URL.String() != tc.wantURLs[i] {
					t.Errorf("link %d = (%q, %q), want (%q, %q)", i, link.Text, link.URL, tc.wantText[i], tc.wantURLs[i])
				}
			}
		})
	}
}

func TestParseEntryLinksDoesNotTreatMalformedLinksAsURLs(t *testing.T) {
	text := "[label](javascript:alert(1)) and [https://example.com]"
	if got := parseEntryLinks(text); got != nil {
		t.Fatalf("parseEntryLinks(%q) = %#v, want no links", text, got)
	}
}

func TestParseEntryLinksResolvesLocalReferences(t *testing.T) {
	dunnitRoot := t.TempDir()
	aliasRoot := t.TempDir()
	searchRoot := t.TempDir()
	t.Setenv("DUNNIT_DIR", dunnitRoot)

	if err := os.MkdirAll(filepath.Join(aliasRoot, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(aliasRoot, "docs", "foo.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(searchRoot, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(searchRoot, "docs", "bar.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		FileAliases:    map[string]string{"cc3": aliasRoot},
		FileSearchPath: []string{searchRoot},
	}
	links := parseEntryLinksWithConfig(
		"[foo](cc3:docs/foo.txt) [bar](docs/bar.txt) [day](dunnit:notes/day.md)",
		cfg,
	)
	if len(links) != 3 {
		t.Fatalf("found %d links, want 3: %#v", len(links), links)
	}

	wantPaths := []string{
		filepath.Join(aliasRoot, "docs", "foo.txt"),
		filepath.Join(searchRoot, "docs", "bar.txt"),
		filepath.Join(dunnitRoot, "notes", "day.md"),
	}
	for i, want := range wantPaths {
		if links[i].LocalPath != want {
			t.Errorf("link %d local path = %q, want %q", i, links[i].LocalPath, want)
		}
		if links[i].URL.String() != localFileURL(want).String() {
			t.Errorf("link %d URL = %q, want %q", i, links[i].URL, localFileURL(want))
		}
	}
}

func TestParseEntryLinksRejectsUnsafeOrUnconfiguredLocalReferences(t *testing.T) {
	text := "[script](javascript:alert(1)) [unknown](other:thing) [escape](dunnit:../secret)"
	if got := parseEntryLinksWithConfig(text, Config{}); got != nil {
		t.Fatalf("parseEntryLinksWithConfig(%q) = %#v, want no links", text, got)
	}
}

func TestParseEntryLinksAcceptsAbsoluteFileURI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "my session.md")
	text := "[session](" + localFileURL(path).String() + ")"
	links := parseEntryLinksWithConfig(text, Config{})
	if len(links) != 1 {
		t.Fatalf("found %d links, want 1: %#v", len(links), links)
	}
	if links[0].LocalPath != path {
		t.Fatalf("local path = %q, want %q", links[0].LocalPath, path)
	}
}

func TestParseEntryLinksInfersAliasFromSearchRootName(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "cc3")
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "docs", "foo.txt")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	links := parseEntryLinksWithConfig("[foo](cc3:docs/foo.txt)", Config{
		FileSearchPath: []string{root},
	})
	if len(links) != 1 || links[0].LocalPath != path {
		t.Fatalf("inferred alias links = %#v, want path %q", links, path)
	}
}

func TestDerivedLinkLabel(t *testing.T) {
	cases := map[string]string{
		"https://foo.atlassian.net/browse/PROJ-42":     "Jira #PROJ-42",
		"https://github.acme.com/acme/dunnit/issues/9": "GitHub #9",
		"https://app.slack.com/client/T123/C456":       "Slack",
		"https://drive.google.com/file/d/abc":          "Google Drive",
		"https://notion.site/example/page":             "Notion",
	}
	for raw, want := range cases {
		target, ok := parseHTTPURL(raw)
		if !ok {
			t.Fatalf("parseHTTPURL(%q) failed", raw)
		}
		if got := derivedLinkLabel(target); got != want {
			t.Errorf("derivedLinkLabel(%q) = %q, want %q", raw, got, want)
		}
	}
}
