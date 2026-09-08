package dun

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yuin/goldmark"
)

// periodReportPath returns DunnitDir()/<kind>-<date formatted with
// format>.md, e.g. periodReportPath("dsu", now, "20060102") or
// periodReportPath("som", from, "200601"). Unifies what were
// previously two slightly-different ad hoc conventions (dsuSavePath
// in standup.go, SOM's inline digestSavePath in som.go).
// dailysummary.go's per-ledger-directory summary-<date>.md convention
// is intentionally left alone -- it lives alongside its ledger file
// rather than at DunnitDir()'s root, a deliberately different scheme.
func periodReportPath(kind string, date time.Time, format string) string {
	return filepath.Join(DunnitDir(), kind+"-"+date.Format(format)+".md")
}

// showGeneratedReport displays a markdown report in a small
// standalone window with Copy (clipboard) and Save (writes to
// savePath) actions, plus Close -- the shared shape behind what were
// previously separate near-duplicate implementations
// (showGeneratedStandupSummary in standup.go, SOM's inline digest
// Copy/Save in som.go). title is the window title; savePath is where
// Save writes text.
func showGeneratedReport(a fyne.App, title, savePath, text string) {
	body := widget.NewRichTextFromMarkdown(text)
	body.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(body)
	scroll.SetMinSize(fyne.NewSize(0, 260))

	w := a.NewWindow(title)

	copyBtn := widget.NewButtonWithIcon("Copy", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(text)
	})
	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := os.WriteFile(savePath, []byte(text), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})

	w.SetContent(windowPad(container.NewBorder(nil,
		container.NewHBox(copyBtn, saveBtn, widget.NewButton("Close", func() { w.Close() })),
		nil, nil,
		scroll,
	)))
	w.Resize(fyne.NewSize(520, 420))
	w.Show()
}

// markdownToHTML renders md as a minimal standalone HTML document via
// goldmark (already an indirect Fyne dependency, promoted here to a
// direct import -- no new dependency actually added). Used by
// showEditableReportWindow's "Copy as HTML" action. Returns md
// unchanged (wrapped in <pre>) if conversion fails, rather than
// erroring out the whole Copy action.
func markdownToHTML(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		log.Println("Error converting markdown to HTML:", err)
		return "<pre>" + md + "</pre>"
	}
	return "<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\"></head><body>\n" +
		buf.String() + "\n</body></html>"
}

// showEditableReportWindow displays a generated report in an editable
// Markdown text window (not the read-only showGeneratedReport above --
// Reviews are meant to be tweakable before saving, per
// docs/kickoff-review-design.md's Review model) with a live Markdown
// preview below, and Save/Copy as HTML/Close actions. Save writes the
// *current edited text* (not the original draft) to savePath.
func showEditableReportWindow(a fyne.App, title, savePath, initialText string) {
	w := a.NewWindow(title)

	editor := widget.NewMultiLineEntry()
	editor.SetText(initialText)
	editor.Wrapping = fyne.TextWrapWord

	preview := widget.NewRichTextFromMarkdown(initialText)
	preview.Wrapping = fyne.TextWrapWord
	editor.OnChanged = func(text string) {
		preview.ParseMarkdown(text)
	}

	editorScroll := container.NewVScroll(editor)
	editorScroll.SetMinSize(fyne.NewSize(0, 260))
	previewScroll := container.NewVScroll(preview)
	previewScroll.SetMinSize(fyne.NewSize(0, 260))

	saveBtn := widget.NewButtonWithIcon("Save", theme.Icon(theme.IconNameDocumentSave), func() {
		if err := os.WriteFile(savePath, []byte(editor.Text), 0644); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Saved", "Saved to "+savePath, w)
	})
	copyHTMLBtn := widget.NewButtonWithIcon("Copy as HTML", theme.Icon(theme.IconNameContentCopy), func() {
		a.Clipboard().SetContent(markdownToHTML(editor.Text))
	})
	closeBtn := widget.NewButton("Close", func() { w.Close() })

	content := container.NewBorder(
		widget.NewLabelWithStyle("Edit (Markdown):", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		container.NewVBox(
			widget.NewLabelWithStyle("Preview:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			previewScroll,
			container.NewHBox(saveBtn, copyHTMLBtn, closeBtn),
		),
		nil, nil,
		editorScroll,
	)
	w.SetContent(windowPad(content))
	w.Resize(fyne.NewSize(640, 720))
	w.Show()
}
