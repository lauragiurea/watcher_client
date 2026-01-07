package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

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
