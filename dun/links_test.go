package dun

import "testing"

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
