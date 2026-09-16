package dun

import (
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// parsedEntryLink is a presentation link found in entry text. The source
// text is deliberately not rewritten; Start and End identify the original
// bytes so callers can replace only the visual portion.
type parsedEntryLink struct {
	Start int
	End   int
	Text  string
	URL   *url.URL
}

var issueKeyPattern = regexp.MustCompile(`(?i)^[a-z][a-z0-9]+-\d+$`)

// parseEntryLinks finds Markdown links and bare HTTP(S) URLs in one line of
// ledger text. Markdown links win over the URL detector, so a destination is
// never rendered a second time inside its own label.
func parseEntryLinks(text string) []parsedEntryLink {
	var links []parsedEntryLink
	for i := 0; i < len(text); {
		if text[i] == '[' {
			if link, end, ok := parseMarkdownLinkAt(text, i); ok {
				links = append(links, link)
				i = end
				continue
			}
		}

		if hasHTTPPrefix(text, i) && bareURLBoundary(text, i) {
			end := i
			for end < len(text) && !unicode.IsSpace(rune(text[end])) && !strings.ContainsRune("<>[]", rune(text[end])) {
				end++
			}
			raw := trimURLPunctuation(text[i:end])
			if raw != "" {
				if target, ok := parseHTTPURL(raw); ok {
					links = append(links, parsedEntryLink{
						Start: i,
						End:   i + len(raw),
						Text:  derivedLinkLabel(target),
						URL:   target,
					})
					i += len(raw)
					continue
				}
			}
		}
		i++
	}
	return links
}

func parseMarkdownLinkAt(text string, start int) (parsedEntryLink, int, bool) {
	close := strings.IndexByte(text[start+1:], ']')
	if close < 0 {
		return parsedEntryLink{}, 0, false
	}
	close += start + 1
	if close == start+1 || close+1 >= len(text) || text[close+1] != '(' {
		return parsedEntryLink{}, 0, false
	}

	end, destination, ok := markdownDestination(text, close+2)
	if !ok {
		return parsedEntryLink{}, 0, false
	}
	target, ok := parseHTTPURL(destination)
	if !ok {
		return parsedEntryLink{}, 0, false
	}
	return parsedEntryLink{
		Start: start,
		End:   end,
		Text:  text[start+1 : close],
		URL:   target,
	}, end, true
}

// markdownDestination handles the common inline-link form and balanced
// parentheses in URLs, such as https://example.test/a_(b).
func markdownDestination(text string, start int) (int, string, bool) {
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			if depth == 0 {
				destination := strings.TrimSpace(text[start:i])
				if len(destination) >= 2 && destination[0] == '<' && destination[len(destination)-1] == '>' {
					destination = destination[1 : len(destination)-1]
				}
				return i + 1, destination, !strings.ContainsAny(destination, " \t\r\n")
			}
			depth--
		}
	}
	return 0, "", false
}

func parseHTTPURL(raw string) (*url.URL, bool) {
	target, err := url.Parse(raw)
	if err != nil || target.Host == "" {
		return nil, false
	}
	scheme := strings.ToLower(target.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, false
	}
	return target, true
}

func hasHTTPPrefix(text string, start int) bool {
	remaining := strings.ToLower(text[start:])
	return strings.HasPrefix(remaining, "http://") || strings.HasPrefix(remaining, "https://")
}

func bareURLBoundary(text string, start int) bool {
	if start == 0 {
		return true
	}
	previous := rune(text[start-1])
	return unicode.IsSpace(previous) || strings.ContainsRune("({<=:", previous)
}

func trimURLPunctuation(raw string) string {
	for len(raw) > 0 && strings.ContainsRune(".,;:!?", rune(raw[len(raw)-1])) {
		raw = raw[:len(raw)-1]
	}
	for strings.HasSuffix(raw, ")") && strings.Count(raw, ")") > strings.Count(raw, "(") {
		raw = strings.TrimSuffix(raw, ")")
	}
	return raw
}

// derivedLinkLabel keeps raw URLs readable in presentations while retaining
// a useful fallback for arbitrary web links.
func derivedLinkLabel(target *url.URL) string {
	host := strings.ToLower(target.Hostname())
	pathParts := nonEmptyPathParts(target.Path)
	for i, part := range pathParts {
		if decoded, err := url.PathUnescape(part); err == nil {
			pathParts[i] = decoded
		}
	}

	if isJiraHost(host) {
		if len(pathParts) > 0 && pathParts[0] == "wiki" {
			return "Confluence"
		}
		for _, part := range pathParts {
			if issueKeyPattern.MatchString(part) {
				return "Jira #" + strings.ToUpper(part)
			}
		}
		return "Jira"
	}

	if isGitHubHost(host) {
		if len(pathParts) >= 4 && (pathParts[2] == "issues" || pathParts[2] == "pull") {
			if pathParts[2] == "pull" {
				return "GitHub PR #" + pathParts[3]
			}
			return "GitHub #" + pathParts[3]
		}
		if len(pathParts) >= 2 {
			return "GitHub " + pathParts[0] + "/" + pathParts[1]
		}
		return "GitHub"
	}

	switch {
	case isHostOrSubdomain(host, "teams.microsoft.com"):
		return "Teams"
	case isHostOrSubdomain(host, "slack.com"):
		return "Slack"
	case host == "docs.google.com":
		if len(pathParts) > 0 {
			switch pathParts[0] {
			case "document":
				return "Google Doc"
			case "spreadsheets":
				return "Google Sheet"
			case "presentation":
				return "Google Slides"
			}
		}
		return "Google Docs"
	case host == "drive.google.com":
		return "Google Drive"
	case host == "calendar.google.com":
		return "Google Calendar"
	case host == "meet.google.com":
		return "Google Meet"
	case host == "forms.google.com":
		return "Google Form"
	case host == "mail.google.com":
		return "Gmail"
	case isHostOrSubdomain(host, "notion.so") || isHostOrSubdomain(host, "notion.site"):
		return "Notion"
	case isHostOrSubdomain(host, "linear.app"):
		for _, part := range pathParts {
			if issueKeyPattern.MatchString(part) {
				return "Linear #" + strings.ToUpper(part)
			}
		}
		return "Linear"
	case isHostOrSubdomain(host, "asana.com"):
		return "Asana"
	case isHostOrSubdomain(host, "trello.com"):
		return "Trello"
	case isHostOrSubdomain(host, "zoom.us"):
		return "Zoom"
	case isHostOrSubdomain(host, "figma.com"):
		return "Figma"
	case isHostOrSubdomain(host, "dropbox.com"):
		return "Dropbox"
	case isHostOrSubdomain(host, "sharepoint.com"):
		return "SharePoint"
	case isHostOrSubdomain(host, "office.com") || isHostOrSubdomain(host, "office365.com"):
		return "Microsoft 365"
	case host == "youtube.com" || host == "youtu.be" || isHostOrSubdomain(host, "youtube.com"):
		return "YouTube"
	}
	return "[link]"
}

func nonEmptyPathParts(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := parts[:0]
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func isJiraHost(host string) bool {
	return isHostOrSubdomain(host, "atlassian.net") || isHostOrSubdomain(host, "jira.com")
}

func isGitHubHost(host string) bool {
	return host == "github.com" ||
		(strings.HasPrefix(host, "github.") && strings.HasSuffix(host, ".com")) ||
		strings.HasSuffix(host, ".github.com")
}

func isHostOrSubdomain(host, domain string) bool {
	return host == domain || strings.HasSuffix(host, "."+domain)
}
