package server

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

var (
	markdownParser    = goldmark.New()
	markdownSanitizer = bluemonday.UGCPolicy()
)

func renderMarkdown(content []byte) (template.HTML, error) {
	var rendered bytes.Buffer
	if err := markdownParser.Convert(content, &rendered); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(markdownSanitizer.SanitizeBytes(rendered.Bytes())), nil
}
