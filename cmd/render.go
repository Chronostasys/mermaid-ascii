package cmd

import (
	"github.com/pgavlin/mermaid-ascii/pkg/diagram"
	"github.com/pgavlin/mermaid-ascii/pkg/render"
)

// RenderDiagram parses and renders a Mermaid diagram as ASCII/Unicode text.
// It delegates to render.Render which auto-detects the diagram type.
func RenderDiagram(input string, config *diagram.Config) (string, error) {
	return render.Render(input, config)
}
