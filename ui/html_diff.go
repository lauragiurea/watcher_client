package ui

import (
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const (
	diffKindInserted = "inserted"
	diffKindDeleted  = "deleted"
	diffKindReplaced = "replaced"

	diffBigDelimiter   = "\n=================================\n\n"
	diffSmallDelimiter = "\n----------- CHANGED TO -----------\n"
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
			TextStyle: fyne.TextStyle{
				Bold:   true,
				Italic: true,
			},
			ColorName: theme.ColorNameWarning,
		},
	}
)

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

func buildPlainDiffSegments(prev, curr string) []diffSegment {
	diffs := diffWordTokens(prev, curr)

	var segments []diffSegment
	seenChangeSeq := false
	inChangeSeq := false
	lastChangeType := diffmatchpatch.DiffEqual
	for idx, d := range diffs {
		if d.Type == diffmatchpatch.DiffEqual {
			inChangeSeq = false
			lastChangeType = diffmatchpatch.DiffEqual
			continue
		}
		if d.Type != diffmatchpatch.DiffInsert && d.Type != diffmatchpatch.DiffDelete {
			continue
		}

		if !inChangeSeq {
			if seenChangeSeq {
				segments = append(segments, diffSegment{
					text:  diffBigDelimiter,
					style: diffPlainStyle,
				})
			}
			inChangeSeq = true
			seenChangeSeq = true
		} else if lastChangeType != diffmatchpatch.DiffEqual && lastChangeType != d.Type {
			segments = append(segments, diffSegment{
				text:  diffSmallDelimiter,
				style: diffPlainStyle,
			})
		}

		beforeCtx := collectContextNodes(diffs, idx, -1)
		if beforeCtx != "" {
			segments = append(segments, diffSegment{
				text:  beforeCtx,
				style: diffPlainStyle,
			})
		}

		style := diffPlainStyle
		if d.Type == diffmatchpatch.DiffInsert {
			style = diffStyleMap[diffKindInserted]
		} else if d.Type == diffmatchpatch.DiffDelete {
			style = diffStyleMap[diffKindDeleted]
		}

		if d.Text != "" {
			segments = append(segments, diffSegment{
				text:  d.Text,
				style: style,
			})
		}

		afterCtx := collectContextNodes(diffs, idx, 1)
		if afterCtx != "" {
			segments = append(segments, diffSegment{
				text:  afterCtx,
				style: diffPlainStyle,
			})
		}

		lastChangeType = d.Type
	}

	return segments
}

func diffWordTokens(prev, curr string) []diffmatchpatch.Diff {
	prevTokens := tokenizeNoSpaceChunks(prev)
	currTokens := tokenizeNoSpaceChunks(curr)

	prevEncoded, currEncoded, tokenArray := tokensToChars(prevTokens, currTokens)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(prevEncoded, currEncoded, false)
	dmp.DiffCleanupEfficiency(diffs)
	diffs = dmp.DiffCharsToLines(diffs, tokenArray)
	return diffs
}

func tokenizeNoSpaceChunks(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	var quoteChar rune

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, r := range input {
		if inQuote {
			current.WriteRune(r)
			if r == quoteChar {
				inQuote = false
				flush()
			}
			continue
		}

		switch {
		case r == '"' || r == '\'':
			flush()
			inQuote = true
			quoteChar = r
			current.WriteRune(r)
		case unicode.IsSpace(r):
			flush()
			tokens = append(tokens, string(r))
		default:
			current.WriteRune(r)
		}
	}

	flush()
	return tokens
}

func tokensToChars(tokens1, tokens2 []string) (string, string, []string) {
	tokenArray := []string{}
	tokenHash := map[string]int{}

	encode := func(tokens []string) string {
		var b strings.Builder
		for _, tok := range tokens {
			idx, ok := tokenHash[tok]
			if !ok {
				tokenArray = append(tokenArray, tok)
				idx = len(tokenArray) - 1
				tokenHash[tok] = idx
			}
			b.WriteRune(rune(idx))
		}
		return b.String()
	}

	return encode(tokens1), encode(tokens2), tokenArray
}

func collectContextNodes(diffs []diffmatchpatch.Diff, idx int, direction int) string {
	var builder strings.Builder
	if direction == -1 {
		for i := idx - 1; i >= 0; i-- {
			if diffs[i].Type == diffmatchpatch.DiffEqual {
				context := extractLastChunks(diffs[i].Text, 5)
				if context != "" {
					builder.WriteString(context)
					break
				}
			}
		}
	} else if direction == 1 {
		for i := idx + 1; i < len(diffs); i++ {
			if diffs[i].Type == diffmatchpatch.DiffEqual {
				context := extractFirstChunks(diffs[i].Text, 5)
				if context != "" {
					builder.WriteString(context)
					break
				}
			}
		}
	}
	return builder.String()
}

func extractFirstChunks(content string, max int) string {
	tokens := tokenizeNoSpaceChunks(content)
	if len(tokens) == 0 {
		return ""
	}

	var collected []string
	count := 0
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		collected = append(collected, tok)
		if !isWhitespaceChunk(tok) {
			count++
			if count >= max {
				break
			}
		}
	}

	return strings.Join(collected, "")
}

func extractLastChunks(content string, max int) string {
	tokens := tokenizeNoSpaceChunks(content)
	if len(tokens) == 0 {
		return ""
	}

	var collected []string
	count := 0
	for i := len(tokens) - 1; i >= 0; i-- {
		tok := tokens[i]
		if tok == "" {
			continue
		}
		collected = append(collected, tok)
		if !isWhitespaceChunk(tok) {
			count++
			if count >= max {
				break
			}
		}
	}

	for left, right := 0, len(collected)-1; left < right; left, right = left+1, right-1 {
		collected[left], collected[right] = collected[right], collected[left]
	}

	return strings.Join(collected, "")
}

func isWhitespaceChunk(token string) bool {
	return strings.TrimSpace(token) == ""
}
