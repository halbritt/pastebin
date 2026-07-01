package server

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var (
	markdownParser    = goldmark.New(goldmark.WithExtensions(extension.Table))
	markdownSanitizer = bluemonday.UGCPolicy()
)

func renderMarkdown(content []byte) (template.HTML, error) {
	var rendered bytes.Buffer
	if err := markdownParser.Convert(content, &rendered); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(markdownSanitizer.SanitizeBytes(rendered.Bytes())), nil
}
