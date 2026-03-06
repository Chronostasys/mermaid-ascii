# Mermaid ASCII

Render [Mermaid](https://mermaid.js.org/) diagrams as ASCII/Unicode art in your terminal. Supports 22 diagram types.

## Installation

Build from source (requires Go 1.21+):

```bash
git clone https://github.com/pgavlin/mermaid-ascii.git
cd mermaid-ascii
go build
./mermaid-ascii --help
```

Or using Nix:

```bash
nix build
./result/bin/mermaid-ascii --help
```

## CLI Usage

```bash
# From a file
mermaid-ascii -f diagram.mermaid

# From stdin
echo 'graph LR
A --> B --> C' | mermaid-ascii

# ASCII-only mode (no Unicode box-drawing characters)
mermaid-ascii -f diagram.mermaid --ascii

# Adjust spacing
mermaid-ascii -f diagram.mermaid -x 8 -y 3 -p 2
```

### Flags

```
  -a, --ascii               Don't use extended character set
  -p, --borderPadding int   Padding between text and border (default 1)
  -c, --coords              Show coordinates
  -f, --file string         Mermaid file to parse (use '-' for stdin)
  -x, --paddingX int        Horizontal space between nodes (default 5)
  -y, --paddingY int        Vertical space between nodes (default 5)
  -v, --verbose             Verbose output
```

### Web Interface

```bash
mermaid-ascii web --port 3001
# Then visit http://localhost:3001
```

## Library Usage

mermaid-ascii is also usable as a Go library:

```go
import (
    "fmt"

    "github.com/pgavlin/mermaid-ascii/pkg/diagram"
    "github.com/pgavlin/mermaid-ascii/pkg/render"
)

func main() {
    input := `graph LR
    A --> B --> C`

    output, err := render.Render(input, diagram.DefaultConfig())
    if err != nil {
        panic(err)
    }
    fmt.Println(output)
}
```

For more control, use `render.Detect()` to get a `Diagram` instance:

```go
diag, err := render.Detect(input)
if err != nil { ... }

if err := diag.Parse(input); err != nil { ... }

output, err := diag.Render(config)
```

## Examples

### Graph / Flowchart

```
$ echo 'graph LR
A --> B & C
B --> C & D
D --> C' | mermaid-ascii
┌───┐     ┌───┐     ┌───┐
│   │     │   │     │   │
│ A ├────►│ B ├────►│ D │
│   │     │   │     │   │
└─┬─┘     └─┬─┘     └─┬─┘
  │         │         │
  │         │         │
  │         │         │
  │         │         │
  │         ▼         │
  │       ┌───┐       │
  │       │   │       │
  └──────►│ C │◄──────┘
          │   │
          └───┘
```

Supports LR, TD/TB, BT, and RL directions, subgraphs, node shapes (round, stadium, subroutine, cylinder, circle, diamond, hexagon, flag), edge types (solid, dotted, thick, bidirectional, cross, circle), edge labels, `classDef` styling, and colored output.

### Sequence Diagram

```
$ echo 'sequenceDiagram
Alice->>Bob: Hello Bob!
Bob-->>Alice: Hi Alice!' | mermaid-ascii
┌───────┐     ┌─────┐
│ Alice │     │ Bob │
└───┬───┘     └──┬──┘
    │            │
    │ Hello Bob! │
    ├───────────►│
    │            │
    │ Hi Alice!  │
    │◄┈┈┈┈┈┈┈┈┈┈┈┤
    │            │
```

Supports all arrow types (->>、-->>、->、-->、-x、--x、-)、--)), activation boxes, notes, interaction blocks (loop, alt, opt, par, critical, break, rect), actors, participant grouping, create/destroy, and aliases.

### Class Diagram

```
$ echo 'classDiagram
class Animal {
  +String name
  +makeSound()
}
class Dog {
  +fetch()
}
Animal <|-- Dog' | mermaid-ascii
```

### State Diagram

```
$ echo 'stateDiagram-v2
[*] --> Active
Active --> Inactive
Inactive --> [*]' | mermaid-ascii
```

### Entity Relationship Diagram

```
$ echo 'erDiagram
CUSTOMER ||--o{ ORDER : places
ORDER ||--|{ LINE-ITEM : contains' | mermaid-ascii
```

### Gantt Chart

```
$ echo 'gantt
title Project Schedule
section Design
  Task A :a1, 2024-01-01, 5d
section Development
  Task B :b1, after a1, 10d' | mermaid-ascii
```

### And More

All 22 Mermaid diagram types are supported — see the full list below.

## Supported Diagram Types

| Diagram Type | Keyword |
|---|---|
| Graph / Flowchart | `graph`, `flowchart` |
| Sequence | `sequenceDiagram` |
| Class | `classDiagram` |
| State | `stateDiagram-v2` |
| Entity Relationship | `erDiagram` |
| Gantt | `gantt` |
| Pie Chart | `pie` |
| Mindmap | `mindmap` |
| Timeline | `timeline` |
| Git Graph | `gitGraph` |
| User Journey | `journey` |
| Quadrant Chart | `quadrantChart` |
| XY Chart | `xychart-beta` |
| C4 Diagram | `C4Context` |
| Requirement | `requirementDiagram` |
| Block Diagram | `block-beta` |
| Sankey | `sankey-beta` |
| Packet | `packet-beta` |
| Kanban | `kanban` |
| Architecture | `architecture-beta` |
| ZenUML | `zenuml` |

## Docker

```bash
docker build -t mermaid-ascii .

echo 'graph LR
A-->B-->C' | docker run -i mermaid-ascii -f -

# Web interface
docker run -p 3001:3001 mermaid-ascii web --port 3001
```

## How It Works

The rendering pipeline:

1. **Detect** — Auto-detect diagram type from the input text
2. **Parse** — Parse Mermaid syntax into a diagram-specific model
3. **Layout** — Compute positions (grid placement + A* pathfinding for graphs)
4. **Render** — Draw to a 2D character canvas, output as string

For graph/flowchart diagrams, nodes are placed on a grid coordinate system. Each node occupies a 3x3 area of grid points, with pathfinding used to route edges between nodes without overlapping.
