package cmd

import (
	"github.com/pgavlin/mermaid-ascii/pkg/diagram"
	"github.com/pgavlin/mermaid-ascii/pkg/render"
)

// DiagramFactory detects the diagram type from input and returns the
// appropriate Diagram implementation. Delegates to pkg/render.
func DiagramFactory(input string) (diagram.Diagram, error) {
	return render.Detect(input)
}
