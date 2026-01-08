package ui

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"watcher-client/api"
)

func ShowChangeDetailWindow(a fyne.App, c api.ChangeEvent, m api.Monitor) {
	w := a.NewWindow("Change – " + m.Name)

	detailSize := fyne.NewSize(900, 600)
	contentSize := fyne.NewSize(detailSize.Width-40, detailSize.Height-80)

	diffContentHolder := container.NewStack(widget.NewLabel("Loading diff…"))
	diffScroll := container.NewScroll(diffContentHolder)
	diffScroll.SetMinSize(contentSize)

	downloadsContentHolder := container.NewStack(widget.NewLabel("Loading downloads…"))
	downloadsScroll := container.NewScroll(downloadsContentHolder)
	downloadsScroll.SetMinSize(contentSize)

	screenshotContent := buildScreenshotContent(c)

	loadAndShowDiff := func(prevURL, currURL *string, diffOverride func(prevHTML, currHTML string) fyne.CanvasObject) {
		prevHTML, errPrev := loadHTMLFromURL(prevURL)
		currHTML, errCurr := loadHTMLFromURL(currURL)

		updateDiffContentWith := func(build func() fyne.CanvasObject) {
			obj := build()
			diffContentHolder.Objects = []fyne.CanvasObject{obj}
			diffContentHolder.Refresh()
		}
		updateDownloadsContentWith := func(build func() fyne.CanvasObject) {
			obj := build()
			downloadsContentHolder.Objects = []fyne.CanvasObject{obj}
			downloadsContentHolder.Refresh()
		}

		if errPrev != nil || errCurr != nil {
			msg := "Failed to load HTML diff."
			if errPrev != nil {
				msg += fmt.Sprintf("\nPrev: %v", errPrev)
			}
			if errCurr != nil {
				msg += fmt.Sprintf("\nCurr: %v", errCurr)
			}
			updateDiffContentWith(func() fyne.CanvasObject {
				label := widget.NewLabel(msg)
				label.Wrapping = fyne.TextWrapWord
				return label
			})
			updateDownloadsContentWith(func() fyne.CanvasObject {
				return buildDownloadsTab(w, "", "", c)
			})
			return
		}

		var diffObj fyne.CanvasObject
		if diffOverride != nil {
			diffObj = diffOverride(prevHTML, currHTML)
		} else {
			diffObj = buildHTMLDiffView(c.HTMLDiff)
		}
		updateDiffContentWith(func() fyne.CanvasObject {
			return diffObj
		})
		updateDownloadsContentWith(func() fyne.CanvasObject {
			return buildDownloadsTab(w, prevHTML, currHTML, c)
		})
	}

	var statusDiffOverride func(prevHTML, currHTML string) fyne.CanvasObject
	if c.HTTPStatusPrev != nil && c.HTTPStatusCurr != nil && *c.HTTPStatusPrev != *c.HTTPStatusCurr {
		prevStatus := *c.HTTPStatusPrev
		currStatus := *c.HTTPStatusCurr
		statusDiffOverride = func(_, _ string) fyne.CanvasObject {
			label := widget.NewLabel(fmt.Sprintf("HTTP status changed from %d to %d.", prevStatus, currStatus))
			label.Wrapping = fyne.TextWrapWord
			return label
		}
	}

	loadAndShowDiff(c.HTMLPrev, c.HTMLCurr, statusDiffOverride)

	tabs := container.NewAppTabs(
		container.NewTabItem("Text diff", diffScroll),
		container.NewTabItem("Screenshots", screenshotContent),
		container.NewTabItem("Downloads", downloadsScroll),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	w.SetContent(tabs)
	w.Resize(detailSize)
	w.Show()
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

func buildDownloadsTab(w fyne.Window, prevHTML, currHTML string, c api.ChangeEvent) fyne.CanvasObject {
	rows := []fyne.CanvasObject{
		buildDownloadRow(w, "Previous HTML", "previous.html", prevHTML),
		buildDownloadRow(w, "Current HTML", "current.html", currHTML),
		buildRemoteDownloadRow(w, "Current screenshot", "current.png", c.ScreenshotCurr),
		buildRemoteDownloadRow(w, "Previous screenshot", "previous.png", c.ScreenshotPrev),
	}
	return container.NewVBox(rows...)
}

func buildDownloadRow(w fyne.Window, label, defaultFile string, content string) fyne.CanvasObject {
	text := widget.NewLabel(label)
	text.Wrapping = fyne.TextWrapWord

	var action fyne.CanvasObject
	if content == "" {
		action = widget.NewLabel("Unavailable")
	} else {
		btn := widget.NewButton("Download", func() {
			autoSaveToDownloads(w, defaultFile, content)
		})
		action = btn
	}

	return container.NewBorder(nil, nil, nil, action, text)
}

func buildRemoteDownloadRow(w fyne.Window, label, defaultFile string, urlPtr *string) fyne.CanvasObject {
	text := widget.NewLabel(label)
	text.Wrapping = fyne.TextWrapWord

	var action fyne.CanvasObject
	if urlPtr == nil || *urlPtr == "" {
		action = widget.NewLabel("Unavailable")
	} else {
		urlCopy := *urlPtr
		action = widget.NewButton("Download", func() {
			downloadAndSaveRemoteFile(w, defaultFile, urlCopy)
		})
	}

	return container.NewBorder(nil, nil, nil, action, text)
}

func autoSaveToDownloads(win fyne.Window, defaultName string, content string) {
	if content == "" {
		dialog.ShowInformation("Download HTML", "No HTML content available.", win)
		return
	}

	saveBytesToDownloads(win, defaultName, []byte(content))
}

func downloadAndSaveRemoteFile(win fyne.Window, defaultName string, url string) {
	if url == "" {
		dialog.ShowInformation("Download asset", "No asset available for download.", win)
		return
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to download asset: %w", err), win)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		dialog.ShowError(fmt.Errorf("asset download returned HTTP %d", resp.StatusCode), win)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to read asset response: %w", err), win)
		return
	}

	saveBytesToDownloads(win, defaultName, data)
}

func saveBytesToDownloads(win fyne.Window, defaultName string, data []byte) {
	downloadsPath, err := os.UserHomeDir()
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to locate home directory: %w", err), win)
		return
	}
	downloadsPath = filepath.Join(downloadsPath, "Downloads")
	if err := os.MkdirAll(downloadsPath, 0o755); err != nil {
		dialog.ShowError(fmt.Errorf("failed to create downloads directory: %w", err), win)
		return
	}

	destPath := filepath.Join(downloadsPath, defaultName)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		dialog.ShowError(fmt.Errorf("failed to save file: %w", err), win)
		return
	}

	dialog.ShowInformation("Download complete", fmt.Sprintf("Saved to %s", destPath), win)
}
