package markdown

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// PlantUMLExtender adds support for PlantUML diagrams via client-side rendering
type PlantUMLExtender struct{}

// Extend extends the Goldmark parser with PlantUML support
func (e *PlantUMLExtender) Extend(md goldmark.Markdown) {
	md.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&PlantUMLRenderer{}, 100),
	))
}

// PlantUMLRenderer renders PlantUML code blocks
type PlantUMLRenderer struct{}

// RegisterFuncs registers the renderer functions
func (r *PlantUMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
}

// renderFencedCodeBlock renders fenced code blocks, handling PlantUML specially
func (r *PlantUMLRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	language := string(n.Language(source))

	// Only handle plantuml code blocks
	if language != "plantuml" {
		return ast.WalkContinue, nil
	}

	if entering {
		// Write opening tag for PlantUML diagram
		// Using a pre tag with class "plantuml" for client-side rendering
		_, _ = w.WriteString(`<pre class="plantuml">`)
		
		// Write the code content
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			_, _ = w.Write(util.EscapeHTML(line.Value(source)))
		}
		
		// Write closing tag
		_, _ = w.WriteString("</pre>\n")
	}

	return ast.WalkSkipChildren, nil
}

// NewPlantUMLExtender creates a new PlantUML extender
func NewPlantUMLExtender() *PlantUMLExtender {
	return &PlantUMLExtender{}
}
