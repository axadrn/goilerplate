package markdown

import (
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// scrollTables wraps every table in a .typeset-scroll box, so a wide table
// scrolls sideways inside the text column instead of widening the page.
type scrollTables struct{}

func (scrollTables) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(east.KindTable, func(w util.BufWriter, _ []byte, _ ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			_, _ = w.WriteString("<div class=\"typeset-scroll\"><table>\n")
		} else {
			_, _ = w.WriteString("</table></div>\n")
		}
		return ast.WalkContinue, nil
	})
}
