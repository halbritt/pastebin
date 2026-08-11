package server

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

type highlightNode struct {
	ast.BaseInline
}

var kindHighlight = ast.NewNodeKind("Highlight")

func (n *highlightNode) Kind() ast.NodeKind {
	return kindHighlight
}

func (n *highlightNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type highlightDelimiterProcessor struct{}

func (p *highlightDelimiterProcessor) IsDelimiter(b byte) bool {
	return b == '='
}

func (p *highlightDelimiterProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	return opener.Char == closer.Char
}

func (p *highlightDelimiterProcessor) OnMatch(consumes int) ast.Node {
	return &highlightNode{}
}

var defaultHighlightDelimiterProcessor = &highlightDelimiterProcessor{}

type highlightParser struct{}

func (p *highlightParser) Trigger() []byte {
	return []byte{'='}
}

func (p *highlightParser) Parse(parent ast.Node, block text.Reader, context parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	delimiter := parser.ScanDelimiter(line, before, 2, defaultHighlightDelimiterProcessor)
	if delimiter == nil || delimiter.OriginalLength != 2 || before == '=' {
		return nil
	}

	delimiter.Segment = segment.WithStop(segment.Start + delimiter.OriginalLength)
	block.Advance(delimiter.OriginalLength)
	context.PushDelimiter(delimiter)
	return delimiter
}

type highlightHTMLRenderer struct{}

func (r *highlightHTMLRenderer) RegisterFuncs(registry renderer.NodeRendererFuncRegisterer) {
	registry.Register(kindHighlight, r.renderHighlight)
}

func (r *highlightHTMLRenderer) renderHighlight(
	writer util.BufWriter, source []byte, node ast.Node, entering bool,
) (ast.WalkStatus, error) {
	if entering {
		_, err := writer.WriteString("<mark>")
		return ast.WalkContinue, err
	}
	_, err := writer.WriteString("</mark>")
	return ast.WalkContinue, err
}

type highlightExtension struct{}

func (e *highlightExtension) Extend(markdown goldmark.Markdown) {
	markdown.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&highlightParser{}, 500),
	))
	markdown.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&highlightHTMLRenderer{}, 500),
	))
}
