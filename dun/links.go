package dun

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// parsedEntryLink is a presentation link found in entry text. The source
// text is deliberately not rewritten; Start and End identify the original
// bytes so callers can replace only the visual portion.
type parsedEntryLink struct {
	Start     int
	End       int
	Text      string
	URL       *url.URL
	LocalPath string
}

var issueKeyPattern = regexp.MustCompile(`(?i)^[a-z][a-z0-9]+-\d+$`)

// parseEntryLinks finds Markdown links and bare HTTP(S) URLs in one line of
// ledger text. Markdown links win over the URL detector, so a destination is
// never rendered a second time inside its own label. Markdown destinations
// may also name local files resolved through Dunnit's configured roots.
func parseEntryLinks(text string) []parsedEntryLink {
	return parseEntryLinksWithConfig(text, LoadConfig())
}

func parseEntryLinksWithConfig(text string, cfg Config) []parsedEntryLink {
	var links []parsedEntryLink
	for i := 0; i < len(text); {
		if text[i] == '[' {
			if link, end, ok := parseMarkdownLinkAt(text, i, cfg); ok {
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

func parseMarkdownLinkAt(text string, start int, cfg Config) (parsedEntryLink, int, bool) {
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
	target, localPath, ok := parseEntryLinkTarget(destination, cfg)
	if !ok {
		return parsedEntryLink{}, 0, false
	}
	return parsedEntryLink{
		Start:     start,
		End:       end,
		Text:      text[start+1 : close],
		URL:       target,
		LocalPath: localPath,
	}, end, true
}

func parseEntryLinkTarget(destination string, cfg Config) (*url.URL, string, bool) {
	if target, ok := parseHTTPURL(destination); ok {
		return target, "", true
	}
	path, ok := resolveLocalFileReference(destination, cfg)
	if !ok {
		return nil, "", false
	}
	return localFileURL(path), path, true
}

// resolveLocalFileReference turns a Markdown destination into an absolute
// local path. dunnit: is rooted at DunnitDir(); a configured alias such as
// cc3:docs/foo.txt is rooted at its alias directory; ordinary relative paths
// search FileSearchPath in order before falling back to DunnitDir().
func resolveLocalFileReference(destination string, cfg Config) (string, bool) {
	destination = strings.TrimSpace(destination)
	if destination == "" || strings.HasPrefix(destination, "#") {
		return "", false
	}

	if strings.HasPrefix(strings.ToLower(destination), "file:") {
		return resolveFileURI(destination, cfg)
	}

	if strings.HasPrefix(strings.ToLower(destination), "dunnit:") {
		relative := strings.TrimLeft(destination[len("dunnit:"):], "/\\")
		return joinLinkRoot(DunnitDir(), relative)
	}

	if alias, relative, ok := splitFileAlias(destination); ok {
		if root, found := fileAliasRoot(alias, cfg); found {
			return joinLinkRoot(root, relative)
		}
		return "", false
	}

	if filepath.IsAbs(destination) {
		return filepath.Clean(destination), true
	}

	// A destination with an unrecognized URI scheme is not a local path. In
	// particular, keep javascript: links out of the custom file resolver.
	if parsed, err := url.Parse(destination); err == nil && parsed.Scheme != "" {
		return "", false
	}

	relative := filepath.FromSlash(destination)
	roots := cfg.FileSearchPath
	if len(roots) == 0 {
		roots = []string{DunnitDir()}
	}
	for _, root := range roots {
		candidate, ok := joinLinkRoot(root, relative)
		if !ok {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	if len(cfg.FileSearchPath) > 0 {
		if candidate, ok := joinLinkRoot(DunnitDir(), relative); ok {
			return candidate, true
		}
	}
	return joinLinkRoot(roots[0], relative)
}

func fileAliasRoot(alias string, cfg Config) (string, bool) {
	if root, found := cfg.FileAliases[alias]; found {
		return root, true
	}
	for _, root := range cfg.FileSearchPath {
		if filepath.Base(expandLinkPath(root)) == alias {
			return root, true
		}
	}
	return "", false
}

func resolveFileURI(raw string, cfg Config) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Scheme, "file") {
		return "", false
	}
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
		return "", false
	}
	path := parsed.Path
	if path == "" {
		path = parsed.Opaque
	}
	if path == "" {
		return "", false
	}
	path, err = url.PathUnescape(path)
	if err != nil {
		return "", false
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), true
	}
	return resolveLocalFileReference(path, cfg)
}

func splitFileAlias(destination string) (alias, relative string, ok bool) {
	colon := strings.IndexByte(destination, ':')
	if colon <= 0 || strings.ContainsAny(destination[:colon], "/\\") {
		return "", "", false
	}
	return destination[:colon], destination[colon+1:], true
}

func joinLinkRoot(root, relative string) (string, bool) {
	root = expandLinkPath(root)
	if root == "" {
		return "", false
	}
	relative = filepath.FromSlash(relative)
	if filepath.IsAbs(relative) {
		return "", false
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	path, err := filepath.Abs(filepath.Join(root, relative))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Clean(path), true
}

func expandLinkPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Clean(path)
}

func localFileURL(path string) *url.URL {
	return &url.URL{Scheme: "file", Path: filepath.ToSlash(path)}
}

func resolveURLToLocalPath(target *url.URL, cfg Config) (string, bool) {
	if target == nil {
		return "", false
	}
	return resolveLocalFileReference(target.String(), cfg)
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
