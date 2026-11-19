package markdown

import (
	"bytes"

	latex "github.com/aziis98/goldmark-latex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// safeLatex is goldmark-latex (inline $...$ and block $$...$$ math, rendered client-side by KaTeX) with
// its inline parser guarded. That parser loops forever when a `$` opens inline math and no closing `$`
// follows before the paragraph ends: it never checks for the end of the block. One lone dollar sign in a
// post ("about $0.015 each") pinned a CPU on every render of the page. The guard looks ahead for the
// closing delimiter first and leaves a lone `$` as text; with one present, the plugin parses as before.
// The plugin's parts are registered exactly as latex.NewLatex().Extend does, with the same priorities.
type safeLatex struct{ cfg *latex.Latex }

func newSafeLatex() goldmark.Extender { return safeLatex{latex.NewLatex()} }

func (e safeLatex) Extend(m goldmark.Markdown) {
	src, out := e.cfg.Source, e.cfg.Output
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(latex.NewLatexBlockParser(src.Block), 701),
	))
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(guardedInlineLatex{
			inner: latex.NewInlineLatexParser(src.Inline),
			begin: []byte(src.Inline.Begin),
			end:   []byte(src.Inline.End),
		}, 501),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(latex.NewLatexBlockRenderer(out.Block), 501),
		util.Prioritized(latex.NewInlineLatexRenderer(out.Inline), 502),
	))
}

type guardedInlineLatex struct {
	inner      parser.InlineParser
	begin, end []byte
}

func (p guardedInlineLatex) Trigger() []byte { return p.inner.Trigger() }

func (p guardedInlineLatex) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	if !closes(block, p.begin, p.end) {
		return nil // not math: the `$` stays text
	}
	return p.inner.Parse(parent, block, pc)
}

// closes says whether the text from the reader's position starts with begin and has end after it before
// the block ends. It reads ahead and puts the reader back where it was.
func closes(block text.Reader, begin, end []byte) bool {
	if first, _ := block.PeekLine(); !bytes.HasPrefix(first, begin) {
		return false
	}
	at, seg := block.Position()
	defer block.SetPosition(at, seg)
	block.Advance(len(begin))
	for {
		rest, _ := block.PeekLine()
		if len(rest) == 0 {
			return false // the block ended (or an empty line) with the math still open
		}
		if bytes.Contains(rest, end) {
			return true
		}
		before, _ := block.Position()
		block.AdvanceLine()
		if after, _ := block.Position(); after == before {
			return false // no progress: never loop
		}
	}
}
