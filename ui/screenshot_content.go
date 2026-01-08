package ui

import (
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

type absoluteLayout struct {
	min fyne.Size
}

func (l *absoluteLayout) Layout(_ []fyne.CanvasObject, _ fyne.Size) {
}

func (l *absoluteLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return l.min
}

func buildScreenshotContent(c api.ChangeEvent) fyne.CanvasObject {
	if c.ScreenshotDiff == nil || *c.ScreenshotDiff == "" {
		return widget.NewLabel("No screenshot diff available")
	}

	uri, err := storage.ParseURI(*c.ScreenshotDiff)
	if err != nil {
		return widget.NewLabel("Failed to parse diff image URI")
	}

	rc, err := storage.Reader(uri)
	if err != nil {
		return widget.NewLabel("Failed to open diff image")
	}
	defer rc.Close()

	src, _, err := image.Decode(rc)
	if err != nil {
		return widget.NewLabel("Failed to decode diff image")
	}

	b := src.Bounds()
	fullSize := fyne.NewSize(float32(b.Dx()), float32(b.Dy()))

	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, src, b.Min, draw.Src)

	const tileSize = 512

	tiles := container.New(&absoluteLayout{min: fullSize})

	for y := 0; y < b.Dy(); y += tileSize {
		for x := 0; x < b.Dx(); x += tileSize {
			w := min(tileSize, b.Dx()-x)
			h := min(tileSize, b.Dy()-y)

			tile := image.NewRGBA(image.Rect(0, 0, w, h))
			srcPt := image.Point{X: b.Min.X + x, Y: b.Min.Y + y}
			draw.Draw(tile, tile.Bounds(), rgba, srcPt, draw.Src)

			im := canvas.NewImageFromImage(tile)
			im.FillMode = canvas.ImageFillOriginal
			im.Resize(fyne.NewSize(float32(w), float32(h)))
			im.Move(fyne.NewPos(float32(x), float32(y)))

			tiles.Add(im)
		}
	}

	return container.NewScroll(tiles)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
