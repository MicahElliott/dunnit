package dun

import (
	"bytes"
	"html"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yuin/goldmark"
)

// reportFilename returns a descriptor-first report filename. The covered
// period comes before the generation date so filenames remain useful when
// sorted lexically while still distinguishing regenerated reports.
func reportFilename(kind, covered, theme string, generated time.Time) string {
	name := kind + "-" + covered + "-" + generated.Format("20060102")
	if theme != "" {
		name += "-" + theme
	}
	return name + ".md"
}

func writeReportFile(path, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0644)
}

// showGeneratedReport displays a markdown report in a small
// standalone window with Markdown/rich-text Copy (clipboard) and Save (writes to
// savePath) actions, plus Close — the shared shape behind what were
// previously separate near-duplicate implementations
// (showGeneratedStandupSummary in standup.go, SOM's inline digest
// Copy/Save in som.go). title is the window title; savePath is where
// Save writes text.
func showGeneratedReport(a fyne.App, title, savePath, text string) {
	heading, bodyText := reportHeadingAndBody(text)
	body := newReportRichText(bodyText)
	body.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(0, 260))

	w := a.NewWindow(title)

	copyButtons := reportCopyButtons(a, text)
	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := writeReportFile(savePath, text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})

	var headingWidget fyne.CanvasObject
	if heading != "" {
		headingText := canvas.NewText(heading, theme.Color(theme.ColorNameForeground))
		headingText.TextSize = theme.Size(theme.SizeNameHeadingText) * 1.35
		headingText.TextStyle = fyne.TextStyle{Bold: true}
		headingWidget = headingText
	}
	w.SetContent(windowPad(container.NewBorder(headingWidget,
		container.NewHBox(copyButtons.Objects[0], copyButtons.Objects[1], saveBtn, widget.NewButton("Close", func() { w.Close() })),
		nil, nil,
		scroll,
	)))
	w.Resize(fyne.NewSize(520, 420))
	w.Show()
}

func newReportRichText(markdown string) *widget.RichText {
	richText := widget.NewRichTextFromMarkdown(markdown)
	configureReportLocalLinks(richText)
	richText.Segments = addReportHeadingSpacing(richText.Segments)
	richText.Refresh()
	return richText
}

func setReportRichTextMarkdown(richText *widget.RichText, markdown string) {
	richText.Segments = widget.NewRichTextFromMarkdown(markdown).Segments
	configureReportLocalLinks(richText)
	richText.Segments = addReportHeadingSpacing(richText.Segments)
	richText.Refresh()
}

func configureReportLocalLinks(richText *widget.RichText) {
	configureReportLocalLinkSegments(richText.Segments, LoadConfig())
}

func configureReportLocalLinkSegments(segments []widget.RichTextSegment, cfg Config) {
	for _, segment := range segments {
		switch segment := segment.(type) {
		case *widget.HyperlinkSegment:
			path, ok := resolveURLToLocalPath(segment.URL, cfg)
			if !ok {
				continue
			}
			segment.URL = localFileURL(path)
			localPath := path
			segment.OnTapped = func() { openInEditor(localPath) }
		case *widget.ParagraphSegment:
			configureReportLocalLinkSegments(segment.Texts, cfg)
		case *widget.ListSegment:
			configureReportLocalLinkSegments(segment.Items, cfg)
		case *widget.TableSegment:
			configureReportLocalLinkTable(segment, cfg)
		}
	}
}

func configureReportLocalLinkTable(table *widget.TableSegment, cfg Config) {
	for _, cell := range table.Headers {
		configureReportLocalLinkSegments(cell, cfg)
	}
	for _, row := range table.Rows {
		for _, cell := range row {
			configureReportLocalLinkSegments(cell, cfg)
		}
	}
}

func addReportHeadingSpacing(segments []widget.RichTextSegment) []widget.RichTextSegment {
	spaced := make([]widget.RichTextSegment, 0, len(segments)+4)
	sectionCount := 0
	for _, segment := range segments {
		if isReportSectionHeading(segment) && sectionCount > 0 {
			spaced = append(spaced, &widget.TextSegment{
				Style: widget.RichTextStyleParagraph,
				Text:  " ",
			})
		}
		spaced = append(spaced, segment)
		if isReportSectionHeading(segment) {
			sectionCount++
		}
	}
	return spaced
}

func isReportSectionHeading(segment widget.RichTextSegment) bool {
	text, ok := segment.(*widget.TextSegment)
	return ok && text.Style.SizeName == theme.SizeNameSubHeadingText
}

func reportHeadingAndBody(text string) (heading, body string) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		heading = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "# "))
		lines = lines[1:]
	}
	return heading, strings.TrimSpace(strings.Join(lines, "\n"))
}

// normalizeReport guarantees a canonical Markdown H1 and removes the common
// title line that LLMs add despite being told that the caller owns the title.
// Keeping this at the shared report boundary ensures copied and saved reports
// have the same meaningful title shown in their preview window.
func normalizeReport(text, title string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 {
		first := strings.TrimSpace(lines[0])
		lower := strings.ToLower(first)
		if strings.HasPrefix(first, "# ") ||
			(strings.HasPrefix(first, "**") && strings.HasSuffix(first, "**") &&
				(strings.Contains(lower, "impact report") || strings.Contains(lower, "summary") || strings.Contains(lower, "review"))) {
			lines = lines[1:]
		}
	}
	body := strings.TrimSpace(strings.Join(lines, "\n"))
	if body == "" {
		return "# " + title + "\n"
	}
	return "# " + title + "\n\n" + body + "\n"
}

// reportCopyButtons returns the two clipboard actions shared by generated
// report windows. Reports are authored as Markdown, while rich text is useful
// for pasting into formatted editors and email.
func reportCopyButtons(a fyne.App, text string) *fyne.Container {
	return container.NewHBox(
		widget.NewButtonWithIcon("Copy as Markdown", theme.Icon(theme.IconNameContentCopy), func() {
			a.Clipboard().SetContent(text)
		}),
		widget.NewButtonWithIcon("Copy as rich text", theme.Icon(theme.IconNameContentCopy), func() {
			copyRichText(a, text)
		}),
	)
}

// markdownHTMLFragment renders Markdown as an HTML fragment via goldmark.
// Clipboard consumers want the fragment itself as their text/html flavor;
// wrapping it in a complete document can make applications paste the markup
// literally or include unwanted document scaffolding.
func markdownHTMLFragment(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		log.Println("Error converting markdown to HTML:", err)
		return "<pre>" + html.EscapeString(md) + "</pre>"
	}
	return buf.String()
}

// markdownToHTML renders md as a minimal standalone HTML document. It is
// retained as a useful representation for callers that need complete HTML;
// copyRichText uses the fragment form for native rich clipboard support.
func markdownToHTML(md string) string {
	fragment := markdownHTMLFragment(md)
	return "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"></head><body>\n" +
		fragment + "\n</body></html>"
}

var (
	markdownLinkPattern    = regexp.MustCompile(`\[([^]]+)\]\([^)]*\)`)
	markdownHeadingPattern = regexp.MustCompile(`^\s{0,3}#{1,6}\s+`)
)

// markdownToPlainText is the safe fallback when the platform has no native
// HTML clipboard command. It keeps the readable report text and removes the
// Markdown delimiters, so the fallback never pastes raw HTML into Teams.
func markdownToPlainText(md string) string {
	var lines []string
	inFence := false
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			lines = append(lines, line)
			continue
		}
		line = markdownHeadingPattern.ReplaceAllString(line, "")
		line = markdownLinkPattern.ReplaceAllString(line, "$1")
		line = strings.NewReplacer("**", "", "__", "", "~~", "", "`", "", "*", "", "_", "").Replace(line)
		lines = append(lines, html.UnescapeString(line))
	}
	return strings.Join(lines, "\n")
}

// copyRichText publishes the report through the host OS's rich clipboard
// support. Fyne's fyne.Clipboard only exposes SetContent(string), so using it
// alone can never create a rich clipboard flavor. The platform commands are
// optional; when absent, readable plain text is copied.
func copyRichText(a fyne.App, markdown string) {
	if copyNativeRichClipboard(markdown) {
		return
	}
	a.Clipboard().SetContent(markdownToPlainText(markdown))
}

func copyNativeRichClipboard(markdown string) bool {
	htmlFragment := markdownHTMLFragment(markdown)
	switch runtime.GOOS {
	case "darwin":
		return copyMacOSRichClipboard(markdownToHTML(markdown))
	case "linux":
		if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("WAYLAND_SOCKET") != "" {
			if _, err := exec.LookPath("wl-copy"); err == nil {
				if startClipboardCommand("wl-copy", []string{"--type", "text/html"}, htmlFragment) {
					return true
				}
			}
		}
		if os.Getenv("DISPLAY") != "" {
			if _, err := exec.LookPath("xclip"); err == nil {
				return startClipboardCommand("xclip", []string{
					"-selection", "clipboard", "-t", "text/html",
					"-alt-text", markdownToPlainText(markdown), "-i",
				}, htmlFragment)
			}
		}
	}
	return false
}

// copyMacOSRichClipboard converts HTML to RTF because pbcopy recognises RTF
// input as rich text. Its -Prefer option belongs to pbpaste and cannot publish
// an HTML pasteboard flavor.
func copyMacOSRichClipboard(htmlDocument string) bool {
	if _, err := exec.LookPath("textutil"); err != nil {
		return false
	}
	convert := exec.Command("textutil", "-stdin", "-format", "html", "-convert", "rtf", "-stdout")
	convert.Stdin = strings.NewReader(htmlDocument)
	rtf, err := convert.Output()
	if err != nil {
		return false
	}
	if len(rtf) == 0 {
		return false
	}
	return runClipboardCommand("pbcopy", nil, string(rtf))
}

func runClipboardCommand(name string, args []string, content string) bool {
	if _, err := exec.LookPath(name); err != nil {
		return false
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run() == nil
}

func startClipboardCommand(name string, args []string, content string) bool {
	cmd := exec.Command(name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		return false
	}
	if _, err := io.WriteString(stdin, content); err != nil {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		return false
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Process.Kill()
		return false
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(100 * time.Millisecond):
		// wl-copy stays alive to serve the selection. xclip may either
		// stay alive or exit after handing the selection to X, so either
		// a clean exit or a still-running process is success.
		return true
	}
}

// showEditableReportWindow displays a generated report in an editable
// Markdown text window (not the read-only showGeneratedReport above --
// Reviews are meant to be tweakable before saving, per
// docs/kickoff-review-design.md's Review model) with a live Markdown
// preview below, and Save/Copy as rich text/Close actions. Save writes the
// *current edited text* (not the original draft) to savePath.
func showEditableReportWindow(a fyne.App, title, savePath, initialText string) {
	w := a.NewWindow(title)

	editor := widget.NewMultiLineEntry()
	editor.SetText(initialText)
	editor.Wrapping = fyne.TextWrapWord

	preview := newReportRichText(initialText)
	preview.Wrapping = fyne.TextWrapWord
	editor.OnChanged = func(text string) {
		setReportRichTextMarkdown(preview, text)
	}

	editorScroll := container.NewVScroll(editor)
	editorScroll.SetMinSize(fyne.NewSize(0, 260))
	previewScroll := container.NewVScroll(preview)
	previewScroll.SetMinSize(fyne.NewSize(0, 260))

	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := writeReportFile(savePath, editor.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})
	copyMarkdownBtn := widget.NewButtonWithIcon("Copy as Markdown", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(editor.Text)
	})
	copyRichTextBtn := widget.NewButtonWithIcon("Copy as rich text", theme.Icon(theme.IconNameContentCopy), func() {
		copyRichText(a, editor.Text)
	})
	closeBtn := widget.NewButton("Close", func() { w.Close() })

	content := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("This is the AI-generated report. Click to edit it before saving.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			widget.NewLabelWithStyle("Closing without Save discards your edits.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		),
		container.NewVBox(
			widget.NewLabelWithStyle("Preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			previewScroll,
			container.NewHBox(saveBtn, copyMarkdownBtn, copyRichTextBtn, closeBtn),
		),
		nil, nil,
		editorScroll,
	)
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(640, 720))
	w.Show()
}
