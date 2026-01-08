package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	diffKindInserted = "inserted"
	diffKindDeleted  = "deleted"
	diffKindReplaced = "replaced"
)

type diffSegment struct {
	text  string
	style widget.RichTextStyle
}

var (
	diffPlainStyle = widget.RichTextStyle{}
	diffStyleMap   = map[string]widget.RichTextStyle{
		diffKindInserted: {
			ColorName: theme.ColorNameSuccess,
		},
		diffKindDeleted: {
			ColorName: theme.ColorNameError,
		},
		diffKindReplaced: {
			ColorName: theme.ColorNameWarning,
		},
	}
)

func renderDiffRichText(segments []diffSegment) *widget.RichText {
	if len(segments) == 0 {
		return widget.NewRichTextFromMarkdown("_No diff available_")
	}

	rtSegments := make([]widget.RichTextSegment, 0, len(segments))
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		rtSegments = append(rtSegments, &widget.TextSegment{
			Text:  seg.text,
			Style: seg.style,
		})
	}

	if len(rtSegments) == 0 {
		return widget.NewRichTextFromMarkdown("_No diff available_")
	}

	diffText := widget.NewRichText(rtSegments...)
	diffText.Wrapping = fyne.TextWrapWord
	return diffText
}

func decodeServerDiff(diffJSON string) ([]diffSegment, error) {
	if diffJSON == "" {
		return nil, nil
	}
	var raw []struct {
		Text string `json:"text"`
		Kind string `json:"kind,omitempty"`
	}
	if err := json.Unmarshal([]byte(diffJSON), &raw); err != nil {
		return nil, err
	}
	segments := make([]diffSegment, 0, len(raw))
	for _, seg := range raw {
		style := diffPlainStyle
		switch seg.Kind {
		case diffKindInserted:
			style = diffStyleMap[diffKindInserted]
		case diffKindDeleted:
			style = diffStyleMap[diffKindDeleted]
		case diffKindReplaced:
			style = diffStyleMap[diffKindReplaced]
		}
		segments = append(segments, diffSegment{
			text:  seg.Text,
			style: style,
		})
	}
	return segments, nil
}

func buildHTMLDiffView(diffURL *string) fyne.CanvasObject {
	if diffURL == nil || *diffURL == "" {
		return widget.NewLabel("No HTML diff available")
	}
	segments, err := fetchAndDecodeDiff(*diffURL)
	if err != nil {
		label := widget.NewLabel(fmt.Sprintf("Failed to load HTML diff: %v", err))
		label.Wrapping = fyne.TextWrapWord
		return label
	}
	return renderDiffRichText(segments)
}

func fetchAndDecodeDiff(url string) ([]diffSegment, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return decodeServerDiff(string(body))
}
