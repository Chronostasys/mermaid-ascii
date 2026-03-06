# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

mermaid-ascii renders Mermaid diagrams as ASCII/Unicode art in the terminal. It supports 22 diagram types. Written in Go (1.21+). Usable both as a CLI tool and as a Go library.

## Commands

- **Build:** `go build` (produces `mermaid-ascii` binary)
- **Run all tests:** `go test ./... -v`
- **Run a single test:** `go test ./cmd -run TestName -v` or `go test ./pkg/sequence -run TestName -v`
- **Run benchmarks:** `go test ./cmd -bench=. -v`
- **Build via Make:** `make` (output in `build/`)

## Library Usage

```go
import (
    "github.com/pgavlin/mermaid-ascii/pkg/diagram"
    "github.com/pgavlin/mermaid-ascii/pkg/render"
)

output, err := render.Render(mermaidInput, diagram.DefaultConfig())
```

## Architecture

The pipeline is: **Parse mermaid text → Detect diagram type → Parse into diagram model → Render to string**

### Entry Points
- `main.go` → `cmd/root.go`: CLI using cobra. Reads input from file or stdin, creates a `Config`, calls `RenderDiagram`.
- `cmd/web.go`: Web interface using gin, serves on a configurable port.
- `pkg/render/render.go`: Library entry point. `Render()` auto-detects diagram type and renders. `Detect()` returns a `Diagram` for manual control.

### Diagram Abstraction (`pkg/diagram/`)
- `interface.go`: `Diagram` interface with `Parse(input)`, `Render(config)`, `Type()` methods.
- `config.go`: `Config` struct holds all rendering parameters (padding, direction, ASCII/Unicode mode, etc.). Use `DefaultConfig()`, `NewTestConfig()`, `NewCLIConfig()`, or `NewWebConfig()` constructors.

### Diagram Type Detection & Dispatch (`pkg/render/`)
- `render.go`: Registry of all diagram type detectors. `Render()` and `Detect()` auto-detect the diagram type from input text. Uses a generic `wrapper[T]` adapter for diagram types following the standard Parse/Render pattern.

### Graph Diagrams (`pkg/graph/`)
- `parse.go`: Parses mermaid graph syntax into `Properties` (nodes, edges, subgraphs, style classes). Uses `orderedmap` to preserve node definition order.
- `graph.go`: Builds a grid layout from parsed data. Nodes are placed on a grid coordinate system, then converted to drawing coordinates.
- `mapping_node.go` / `mapping_edge.go`: Map nodes and edges from grid coords to drawing coords.
- `draw.go`: Renders the final ASCII/Unicode output from the drawing model. Contains `Render()` public API.
- `direction.go`: Handles LR (left-right) vs TD (top-down) layout.
- `arrow.go`: Arrow/edge rendering characters.

### Sequence Diagrams (`pkg/sequence/`)
- `parser.go`: Parses sequence diagram syntax (participants, messages, aliases).
- `renderer.go`: Renders sequence diagrams with participant boxes and message arrows.
- `charset.go`: Unicode vs ASCII character sets for rendering.

### CLI Layer (`cmd/`)
- `diagram.go`: Thin wrapper providing `DiagramFactory()` (delegates to `pkg/render`).
- `render.go`: Thin wrapper providing `RenderDiagram()` (delegates to `pkg/render`).
- `root.go`: Cobra CLI setup with flags.
- `web.go`: Gin web server.

## Testing

Tests use a **golden file pattern** in `cmd/testdata/`:
- `cmd/testdata/ascii/` — ASCII mode graph tests
- `cmd/testdata/extended-chars/` — Unicode mode graph tests
- `cmd/testdata/sequence/` — Unicode sequence diagram tests
- `cmd/testdata/sequence-ascii/` — ASCII sequence diagram tests

Each test file has mermaid input and expected output separated by `---`. When adding new features, add corresponding test files here. The test runner is in `cmd/graph_test.go`.

Integration tests for sequence diagrams are in `cmd/integration_test.go`. Unit tests for sequence parsing/rendering are in `pkg/sequence/`.

`cmd/mermaidjs_docs_test.go` contains 73 test cases sourced from the official Mermaid.js documentation.
