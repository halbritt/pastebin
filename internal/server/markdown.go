package server

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/net/html"
)

var (
	markdownParser    = goldmark.New(goldmark.WithExtensions(extension.Table))
	markdownSanitizer = bluemonday.UGCPolicy()
)

const wideTableColumnThreshold = 4

func renderMarkdown(content []byte) (template.HTML, error) {
	var rendered bytes.Buffer
	if err := markdownParser.Convert(content, &rendered); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	labeled, err := addTableCellLabels(markdownSanitizer.SanitizeBytes(rendered.Bytes()))
	if err != nil {
		return "", fmt.Errorf("label markdown tables: %w", err)
	}
	return template.HTML(labeled), nil
}

func addTableCellLabels(content []byte) ([]byte, error) {
	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	body := firstElement(doc, "body")
	if body == nil {
		return content, nil
	}
	labelTables(body)

	var output bytes.Buffer
	for child := body.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&output, child); err != nil {
			return nil, err
		}
	}
	return output.Bytes(), nil
}

func labelTables(node *html.Node) {
	if isElement(node, "table") {
		labelTableRows(node)
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		labelTables(child)
	}
}

func labelTableRows(table *html.Node) {
	headerRow := tableHeaderRow(table)
	if headerRow == nil {
		return
	}
	labels := cellLabels(headerRow)
	if len(labels) == 0 {
		return
	}
	if len(labels) > wideTableColumnThreshold {
		addClass(table, "table-wide")
	}
	for _, row := range descendantElements(table, "tr") {
		if row == headerRow || hasAncestor(row, "thead") {
			continue
		}
		cellIndex := 0
		for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
			if !isElement(cell, "td") {
				continue
			}
			if cellIndex < len(labels) && labels[cellIndex] != "" {
				setAttr(cell, "data-label", labels[cellIndex])
			}
			cellIndex++
		}
	}
}

func tableHeaderRow(table *html.Node) *html.Node {
	for child := table.FirstChild; child != nil; child = child.NextSibling {
		if isElement(child, "thead") {
			return firstElement(child, "tr")
		}
	}
	for _, row := range descendantElements(table, "tr") {
		for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
			if isElement(cell, "th") {
				return row
			}
		}
	}
	return nil
}

func cellLabels(row *html.Node) []string {
	var labels []string
	for cell := row.FirstChild; cell != nil; cell = cell.NextSibling {
		if isElement(cell, "th") || isElement(cell, "td") {
			labels = append(labels, nodeText(cell))
		}
	}
	return labels
}

func descendantElements(node *html.Node, name string) []*html.Node {
	var found []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if isElement(child, name) {
			found = append(found, child)
		}
		found = append(found, descendantElements(child, name)...)
	}
	return found
}

func firstElement(node *html.Node, name string) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if isElement(child, name) {
			return child
		}
		if found := firstElement(child, name); found != nil {
			return found
		}
	}
	return nil
}

func hasAncestor(node *html.Node, name string) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if isElement(parent, name) {
			return true
		}
	}
	return false
}

func isElement(node *html.Node, name string) bool {
	return node != nil && node.Type == html.ElementNode && node.Data == name
}

func nodeText(node *html.Node) string {
	var parts []string
	var collect func(*html.Node)
	collect = func(current *html.Node) {
		if current.Type == html.TextNode {
			parts = append(parts, current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(node)
	return strings.Join(strings.Fields(strings.Join(parts, " ")), " ")
}

func setAttr(node *html.Node, key, value string) {
	for i := range node.Attr {
		if node.Attr[i].Key == key {
			node.Attr[i].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{Key: key, Val: value})
}

func addClass(node *html.Node, className string) {
	for i := range node.Attr {
		if node.Attr[i].Key == "class" {
			classes := strings.Fields(node.Attr[i].Val)
			for _, existing := range classes {
				if existing == className {
					return
				}
			}
			classes = append(classes, className)
			node.Attr[i].Val = strings.Join(classes, " ")
			return
		}
	}
	node.Attr = append(node.Attr, html.Attribute{Key: "class", Val: className})
}
