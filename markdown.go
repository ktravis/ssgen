package main

import (
	"bytes"
	"strings"

	figure "github.com/mangoumbrella/goldmark-figure"
	fences "github.com/stefanfritsch/goldmark-fences"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.abhg.dev/goldmark/anchor"
	"go.abhg.dev/goldmark/frontmatter"
)

func MarkdownToHtml(ctx parser.Context, text string) (string, error) {
	text = strings.TrimSpace(text)

	if text == "" {
		return "", nil
	}

	var buf bytes.Buffer

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Strikethrough,
			extension.TaskList,
			extension.Footnote,
			&fences.Extender{},
			&anchor.Extender{
				Texter: anchor.Text("#"),
			},
			figure.Figure.WithImageLink(),
			&frontmatter.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)
	if err := md.Convert([]byte(text), &buf, parser.WithContext(ctx)); err != nil {
		return "", err
	}

	return buf.String(), nil
}
