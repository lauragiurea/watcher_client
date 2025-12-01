package ui

import (
	"html"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

func ShowChangeDetailWindow(a fyne.App, c api.ChangeEvent, m api.Monitor) {
	w := a.NewWindow("Change – " + m.Name)

	diffRich := buildDiffRichText(c.TextDiffHTML)

	screenshotContent := buildScreenshotContent(c)

	tabs := container.NewAppTabs(
		container.NewTabItem("Text diff", diffRich),
		container.NewTabItem("Screenshots", screenshotContent),
	)

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(900, 600))
	w.Show()
}

func buildDiffRichText(htmlPtr *string) *widget.RichText {
	if htmlPtr == nil || *htmlPtr == "" {
		return widget.NewRichTextFromMarkdown("_No diff available_")
	}

	s := *htmlPtr

	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")

	reIns := regexp.MustCompile(`(?is)<ins>(.*?)</ins>`)
	s = reIns.ReplaceAllString(s, `**$1**`)

	reDel := regexp.MustCompile(`(?is)<del>(.*?)</del>`)
	s = reDel.ReplaceAllString(s, `~~$1~~`)

	reTags := regexp.MustCompile(`(?is)<[^>]+>`)
	s = reTags.ReplaceAllString(s, "")

	s = html.UnescapeString(s)

	return widget.NewRichTextFromMarkdown(s)
}

func stripHTML(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "<br>", "\n")
	for {
		start := strings.Index(s, "<")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], ">")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return s
}

func buildScreenshotContent(c api.ChangeEvent) fyne.CanvasObject {
	imgs := []fyne.CanvasObject{}

	addImage := func(label string, urlPtr *string) {
		if urlPtr == nil || *urlPtr == "" {
			return
		}
		uri, err := storage.ParseURI(*urlPtr)
		if err != nil {
			return
		}
		img := canvas.NewImageFromURI(uri)
		img.FillMode = canvas.ImageFillContain
		box := container.NewBorder(widget.NewLabel(label), nil, nil, nil, img)
		imgs = append(imgs, box)
	}

	addImage("Current", c.ScreenshotURL)
	addImage("Previous", c.ScreenshotPrevURL)
	addImage("Diff", c.ScreenshotDiffURL)

	if len(imgs) == 0 {
		return widget.NewLabel("No screenshots available")
	}

	return container.NewVBox(imgs...)
}
