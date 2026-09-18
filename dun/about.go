package dun

import (
	"fmt"
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const dunnitHomepage = "https://dunnit.today"

func showAbout(a fyne.App, parent fyne.Window) {
	meta := a.Metadata()
	version := meta.Version
	if version == "" {
		version = "development build"
	}
	versionText := "Version " + version
	if meta.Build > 0 {
		versionText = fmt.Sprintf("%s (%d)", versionText, meta.Build)
	}

	content := container.NewVBox()
	if icon := a.Icon(); icon != nil {
		logo := canvas.NewImageFromResource(icon)
		logo.FillMode = canvas.ImageFillContain
		logo.SetMinSize(fyne.NewSize(128, 128))
		content.Add(container.NewCenter(logo))
	}
	content.Add(widget.NewLabelWithStyle("dunnit", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}))
	content.Add(widget.NewLabelWithStyle("Daily activity tracker", fyne.TextAlignCenter, fyne.TextStyle{}))
	content.Add(widget.NewLabelWithStyle(versionText, fyne.TextAlignCenter, fyne.TextStyle{}))

	if homepage, err := url.Parse(dunnitHomepage); err == nil {
		link := widget.NewHyperlink("dunnit.today", homepage)
		link.Alignment = fyne.TextAlignCenter
		content.Add(link)
	}

	dialog.ShowCustom("About dunnit", "Close", windowPad(content), parent)
}
