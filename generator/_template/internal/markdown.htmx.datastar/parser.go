package markdown

import (
	"bytes"
	"io"

	"{{.ModulePath}}/internal/model"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"
)

type Parser struct {
	md goldmark.Markdown
}

func NewParser() *Parser {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			&frontmatter.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			goldmarkhtml.WithHardWraps(),
			goldmarkhtml.WithXHTML(),
		),
	)

	return &Parser{
		md: md,
	}
}

func (p *Parser) Parse(source []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := p.md.Convert(source, &buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Parser) ParseWithFrontmatter(source []byte) (content []byte, meta map[string]any, err error) {
	context := parser.NewContext()
	var buf bytes.Buffer

	err = p.md.Convert(source, &buf, parser.WithContext(context))
	if err != nil {
		return nil, nil, err
	}

	data := frontmatter.Get(context)
	if data == nil {
		meta = make(map[string]any)
	} else {
		err = data.Decode(&meta)
		if err != nil {
			meta = make(map[string]any)
		}
	}

	return buf.Bytes(), meta, nil
}

func (p *Parser) ConvertReader(r io.Reader, w io.Writer) error {
	data, err := readAll(r)
	if err != nil {
		return err
	}
	return p.md.Convert(data, w)
}

func readAll(r io.Reader) ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Parser) ExtractFrontmatter(source []byte) map[string]any {
	context := parser.NewContext()
	p.md.Parser().Parse(text.NewReader(source), parser.WithContext(context))

	data := frontmatter.Get(context)
	if data == nil {
		return make(map[string]any)
	}

	var meta map[string]any
	err := data.Decode(&meta)
	if err != nil {
		return make(map[string]any)
	}
	return meta
}

func (p *Parser) Renderer() renderer.Renderer {
	return p.md.Renderer()
}

func (p *Parser) ExtractHeadings(source []byte) []model.DocHeading {
	doc := p.md.Parser().Parse(text.NewReader(source))
	headings := make([]model.DocHeading, 0)

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok || heading.Level < 2 || heading.Level > 3 {
			return ast.WalkContinue, nil
		}

		idAttr, ok := heading.Attribute([]byte("id"))
		if !ok {
			return ast.WalkContinue, nil
		}

		text := extractNodeText(heading, source)
		if text == "" {
			return ast.WalkContinue, nil
		}

		id, ok := idAttr.([]byte)
		if !ok || len(id) == 0 {
			return ast.WalkContinue, nil
		}

		headings = append(headings, model.DocHeading{
			ID:    string(id),
			Text:  text,
			Level: heading.Level,
		})

		return ast.WalkContinue, nil
	})

	return headings
}

func extractNodeText(node ast.Node, source []byte) string {
	var textBuf bytes.Buffer

	var walk func(ast.Node)
	walk = func(current ast.Node) {
		for child := current.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				textBuf.Write(textNode.Segment.Value(source))
				continue
			}
			walk(child)
		}
	}

	walk(node)

	return textBuf.String()
}
