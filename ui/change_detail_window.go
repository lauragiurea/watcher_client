package ui

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sergi/go-diff/diffmatchpatch"

	"watcher-client/api"
)

type diffSegment struct {
	text  string
	style widget.RichTextStyle
}

const (
	diffKindInserted = "inserted"
	diffKindDeleted  = "deleted"
	diffKindReplaced = "replaced"
	plainDiffContext = 80
)

var (
	diffPlainStyle = widget.RichTextStyle{}
	diffStyleMap   = map[string]widget.RichTextStyle{
		diffKindInserted: {
			TextStyle: fyne.TextStyle{Bold: true},
			ColorName: theme.ColorNameSuccess,
		},
		diffKindDeleted: {
			TextStyle: fyne.TextStyle{Italic: true},
			ColorName: theme.ColorNameError,
		},
		diffKindReplaced: {
			TextStyle: fyne.TextStyle{
				Bold:   true,
				Italic: true,
			},
			ColorName: theme.ColorNameWarning,
		},
	}
)

func ShowChangeDetailWindow(a fyne.App, c api.ChangeEvent, m api.Monitor) {
	w := a.NewWindow("Change – " + m.Name)

	detailSize := fyne.NewSize(900, 600)
	contentSize := fyne.NewSize(detailSize.Width-40, detailSize.Height-80)

	diffContentHolder := container.NewStack(widget.NewLabel("Loading diff…"))
	diffScroll := container.NewScroll(diffContentHolder)
	diffScroll.SetMinSize(contentSize)

	screenshotScroll := container.NewScroll(buildScreenshotContent(c))
	screenshotScroll.SetMinSize(contentSize)

	loadAndShowDiff := func(prevURL, currURL *string) {
		prevHTML, errPrev := loadHTMLFromURL(prevURL)
		currHTML, errCurr := loadHTMLFromURL(currURL)

		updateContentWith := func(build func() fyne.CanvasObject) {
			apply := func() {
				obj := build()
				diffContentHolder.Objects = []fyne.CanvasObject{obj}
				diffContentHolder.Refresh()
			}
			apply()
		}

		if errPrev != nil || errCurr != nil {
			msg := "Failed to load HTML diff."
			if errPrev != nil {
				msg += fmt.Sprintf("\nPrev: %v", errPrev)
			}
			if errCurr != nil {
				msg += fmt.Sprintf("\nCurr: %v", errCurr)
			}
			updateContentWith(func() fyne.CanvasObject {
				label := widget.NewLabel(msg)
				label.Wrapping = fyne.TextWrapWord
				return label
			})
			return
		}

		var prevPtr *string
		if prevURL != nil && *prevURL != "" {
			prevCopy := prevHTML
			prevPtr = &prevCopy
		}
		var currPtr *string
		if currURL != nil && *currURL != "" {
			currCopy := currHTML
			currPtr = &currCopy
		}

		segments, err := buildDiffSegments(prevPtr, currPtr)
		if err != nil {
			msg := fmt.Sprintf("Failed to build HTML diff: %v", err)
			updateContentWith(func() fyne.CanvasObject {
				label := widget.NewLabel(msg)
				label.Wrapping = fyne.TextWrapWord
				return label
			})
			return
		}
		fmt.Printf("diff: built %d render segments\n", len(segments))
		updateContentWith(func() fyne.CanvasObject {
			return renderDiffRichText(segments)
		})
	}

	loadAndShowDiff(c.HTMLPrev, c.HTMLCurr)

	tabs := container.NewAppTabs(
		container.NewTabItem("Text diff", diffScroll),
		container.NewTabItem("Screenshots", screenshotScroll),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.Resize(detailSize)
	w.Show()
}

func buildDiffSegments(prevHTML, currHTML *string) ([]diffSegment, error) {
	if (prevHTML == nil || *prevHTML == "") && (currHTML == nil || *currHTML == "") {
		return nil, nil
	}

	prev := ""
	if prevHTML != nil {
		prev = *prevHTML
	}
	curr := ""
	if currHTML != nil {
		curr = *currHTML
	}

	segments := buildPlainDiffSegments(prev, curr)
	return segments, nil
}

func buildPlainDiffSegments(prev, curr string) []diffSegment {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(prev, curr, false)
	dmp.DiffCleanupSemantic(diffs)

	var segments []diffSegment
	for i, d := range diffs {
		switch d.Type {
		case diffmatchpatch.DiffEqual:
			prevChange := i > 0 && diffs[i-1].Type != diffmatchpatch.DiffEqual
			nextChange := i < len(diffs)-1 && diffs[i+1].Type != diffmatchpatch.DiffEqual
			if !(prevChange || nextChange) {
				continue
			}
			text := d.Text
			if len(text) > plainDiffContext*2 {
				head := text[:plainDiffContext]
				tail := text[len(text)-plainDiffContext:]
				text = head + "\n... unchanged ...\n" + tail
			}
			segments = append(segments, diffSegment{
				text:  text,
				style: diffPlainStyle,
			})
		case diffmatchpatch.DiffInsert:
			segments = append(segments, diffSegment{
				text:  d.Text,
				style: diffStyleMap[diffKindInserted],
			})
		case diffmatchpatch.DiffDelete:
			segments = append(segments, diffSegment{
				text:  d.Text,
				style: diffStyleMap[diffKindDeleted],
			})
		}
	}

	return segments
}

func renderDiffRichText(segments []diffSegment) *widget.RichText {
	if len(segments) == 0 {
		return widget.NewRichTextFromMarkdown("_No diff available_")
	}

	rtSegments := make([]widget.RichTextSegment, 0, len(segments))
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		txt := seg.text
		style := seg.style
		rtSegments = append(rtSegments, &widget.TextSegment{
			Text:  txt,
			Style: style,
		})
	}

	if len(rtSegments) == 0 {
		return widget.NewRichTextFromMarkdown("_No diff available_")
	}

	fmt.Printf("diff: creating RichText with %d segments\n", len(rtSegments))
	diffText := widget.NewRichText(rtSegments...)
	diffText.Wrapping = fyne.TextWrapWord
	return diffText
}

func loadHTMLFromURL(urlPtr *string) (string, error) {
	if urlPtr == nil || *urlPtr == "" {
		return "", nil
	}
	url := *urlPtr
	fmt.Printf("diff: downloading HTML from %s\n", url)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("diff: failed to GET %s: %v\n", url, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Printf("diff: GET %s returned HTTP %d\n", url, resp.StatusCode)
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("diff: failed to read body from %s: %v\n", url, err)
		return "", err
	}
	fmt.Printf("diff: download succeeded from %s (%d bytes)\n", url, len(body))
	return string(body), nil
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
