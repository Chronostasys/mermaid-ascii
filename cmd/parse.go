package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/elliotchance/orderedmap/v2"
	log "github.com/sirupsen/logrus"
)

type graphProperties struct {
	data             *orderedmap.OrderedMap[string, []textEdge]
	nodeInfo         map[string]textNode // maps node id to its textNode (for shape/label info)
	styleClasses     *map[string]styleClass
	graphDirection   string
	styleType        string
	paddingX         int
	paddingY         int
	subgraphs        []*textSubgraph
	useAscii         bool
	boxBorderPadding int
	showCoords       bool
}

// nodeShape represents the visual shape of a node in a graph diagram.
type nodeShape int

const (
	shapeRect      nodeShape = iota // A[text] or bare A - rectangle (default)
	shapeRounded                    // A(text) - rounded rectangle
	shapeStadium                    // A([text]) - stadium-shaped (rounded sides)
	shapeSubroutine                 // A[[text]] - subroutine (double vertical borders)
	shapeCylinder                   // A[(text)] - cylinder (curved top/bottom)
	shapeCircle                     // A((text)) - circle/double circle
	shapeDiamond                    // A{text} - diamond/rhombus
	shapeHexagon                    // A{{text}} - hexagon
	shapeFlag                       // A>text] - asymmetric/flag shape
)

type textNode struct {
	id         string    // unique identifier used as map key (e.g. "A" from "A[text]")
	name       string    // display label (e.g. "text" from "A[text]", or "A" for bare nodes)
	styleClass string
	shape      nodeShape
}

// EdgeType represents the type of edge connecting two nodes.
type EdgeType int

const (
	SolidArrow         EdgeType = iota // -->
	SolidLine                          // ---
	DottedArrow                        // -.->
	DottedLine                         // -.-
	ThickArrow                         // ==>
	ThickLine                          // ===
	BidirectionalArrow                 // <-->
	CrossEnd                           // --x
	CircleEnd                          // --o
)

type textEdge struct {
	parent   textNode
	child    textNode
	label    string
	edgeType EdgeType
}

type textSubgraph struct {
	id        string // unique identifier for edge references (e.g. "sg1" from "subgraph sg1 [Title]")
	name      string // display label (e.g. "Title" from "subgraph sg1 [Title]")
	nodes     []string
	parent    *textSubgraph
	children  []*textSubgraph
	direction string // per-subgraph direction (LR or TD), parsed but not yet applied to layout
}

// getSubgraphByID returns the subgraph with the given ID, or nil if not found.
func (gp *graphProperties) getSubgraphByID(id string) *textSubgraph {
	for _, sg := range gp.subgraphs {
		if sg.id == id {
			return sg
		}
	}
	return nil
}

// resolveSubgraphNode checks if a node ID refers to a subgraph, and if so,
// returns the first node inside that subgraph as a proxy. Returns the original
// node ID if it's not a subgraph reference.
func (gp *graphProperties) resolveSubgraphNode(nodeID string) string {
	sg := gp.getSubgraphByID(nodeID)
	if sg != nil && len(sg.nodes) > 0 {
		return sg.nodes[0]
	}
	return nodeID
}

// decodeEntityCodes replaces mermaid entity codes with their actual characters.
func decodeEntityCodes(s string) string {
	replacer := strings.NewReplacer(
		"#35;", "#",
		"#amp;", "&",
		"#lt;", "<",
		"#gt;", ">",
		"#quot;", "\"",
		"#nbsp;", " ",
	)
	return replacer.Replace(s)
}

// stripMarkdown removes basic markdown formatting from text.
func stripMarkdown(s string) string {
	// Bold: **text** -> text (must come before italic)
	bold := regexp.MustCompile(`\*\*(.+?)\*\*`)
	s = bold.ReplaceAllString(s, "$1")
	// Italic: *text* -> text
	italic := regexp.MustCompile(`\*(.+?)\*`)
	s = italic.ReplaceAllString(s, "$1")
	return s
}

// extractNodeShape parses a node identifier and optional shape/label syntax.
// Examples:
//   - "A"         -> id="A", label="A", shape=shapeRect
//   - "A[text]"   -> id="A", label="text", shape=shapeRect
//   - "A(text)"   -> id="A", label="text", shape=shapeRounded
//   - "A([text])" -> id="A", label="text", shape=shapeStadium
//   - "A[[text]]" -> id="A", label="text", shape=shapeSubroutine
//   - "A[(text)]" -> id="A", label="text", shape=shapeCylinder
//   - "A((text))" -> id="A", label="text", shape=shapeCircle
//   - "A{text}"   -> id="A", label="text", shape=shapeDiamond
//   - "A{{text}}" -> id="A", label="text", shape=shapeHexagon
//   - "A>text]"   -> id="A", label="text", shape=shapeFlag
func extractNodeShape(raw string) (id string, label string, shape nodeShape) {
	// Try each shape pattern from most specific to least specific.
	// Order matters: double-delimiter patterns before single-delimiter patterns.
	patterns := []struct {
		re    *regexp.Regexp
		shape nodeShape
	}{
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\["([^"]+)"\]$`), shapeRect},      // A["quoted text"]
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\(\[(.+)\]\)$`), shapeStadium},    // A([text])
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[\[(.+)\]\]$`), shapeSubroutine}, // A[[text]]
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[\((.+)\)\]$`), shapeCylinder},   // A[(text)]
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\(\((.+)\)\)$`), shapeCircle},     // A((text))
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\{\{(.+)\}\}$`), shapeHexagon},    // A{{text}}
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\[(.+)\]$`), shapeRect},           // A[text]
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\((.+)\)$`), shapeRounded},        // A(text)
		{regexp.MustCompile(`^([A-Za-z0-9_]+)\{(.+)\}$`), shapeDiamond},        // A{text}
		{regexp.MustCompile(`^([A-Za-z0-9_]+)>(.+)\]$`), shapeFlag},            // A>text]
	}

	for _, p := range patterns {
		if m := p.re.FindStringSubmatch(raw); m != nil {
			return m[1], strings.TrimSpace(m[2]), p.shape
		}
	}

	// No shape syntax found - bare node name
	return raw, raw, shapeRect
}

func parseNode(line string) textNode {
	// Trim any whitespace from the line that might be left after comment removal
	trimmedLine := strings.TrimSpace(line)

	nodeWithClass, _ := regexp.Compile(`^(.+):::(.+)$`)

	var nodeBody, class string
	if match := nodeWithClass.FindStringSubmatch(trimmedLine); match != nil {
		nodeBody = strings.TrimSpace(match[1])
		class = strings.TrimSpace(match[2])
	} else {
		nodeBody = trimmedLine
		class = ""
	}

	id, label, shape := extractNodeShape(nodeBody)
	// Post-process label: decode entity codes and strip markdown
	label = decodeEntityCodes(label)
	label = stripMarkdown(label)
	_ = id // id is used for the data map key but label is the display name
	return textNode{name: label, styleClass: class, shape: shape, id: id}
}

func parseStyleClass(matchedLine []string) styleClass {
	className := matchedLine[0]
	styles := matchedLine[1]
	// Styles are comma separated and key-values are separated by colon
	// Example: fill:#f9f,stroke:#333,stroke-width:4px
	styleMap := make(map[string]string)
	for _, style := range strings.Split(styles, ",") {
		kv := strings.Split(style, ":")
		styleMap[kv[0]] = kv[1]
	}
	return styleClass{className, styleMap}
}

func setArrowWithLabelAndType(lhs, rhs []textNode, label string, edgeType EdgeType, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) []textNode {
	log.Debug("Setting arrow from ", lhs, " to ", rhs, " with label ", label, " type ", edgeType)
	for _, l := range lhs {
		for _, r := range rhs {
			setData(l, textEdge{l, r, label, edgeType}, data, nodeInfo)
		}
	}
	return rhs
}

func setArrowWithLabel(lhs, rhs []textNode, label string, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) []textNode {
	return setArrowWithLabelAndType(lhs, rhs, label, SolidArrow, data, nodeInfo)
}

func setArrow(lhs, rhs []textNode, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) []textNode {
	return setArrowWithLabelAndType(lhs, rhs, "", SolidArrow, data, nodeInfo)
}

func setArrowOfType(lhs, rhs []textNode, edgeType EdgeType, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) []textNode {
	return setArrowWithLabelAndType(lhs, rhs, "", edgeType, data, nodeInfo)
}

func setArrowWithLabelOfType(lhs, rhs []textNode, label string, edgeType EdgeType, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) []textNode {
	return setArrowWithLabelAndType(lhs, rhs, label, edgeType, data, nodeInfo)
}

func addNode(node textNode, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) {
	if _, ok := data.Get(node.id); !ok {
		data.Set(node.id, []textEdge{})
	}
	// Always store/update node info (later definitions can override labels)
	if _, exists := nodeInfo[node.id]; !exists {
		nodeInfo[node.id] = node
	}
}

func setData(parent textNode, edge textEdge, data *orderedmap.OrderedMap[string, []textEdge], nodeInfo map[string]textNode) {
	// Check if the parent is in the map
	if children, ok := data.Get(parent.id); ok {
		// If it is, append the child to the list of children
		data.Set(parent.id, append(children, edge))
	} else {
		// If it isn't, add it to the map
		data.Set(parent.id, []textEdge{edge})
	}
	// Store node info for parent and child
	if _, exists := nodeInfo[parent.id]; !exists {
		nodeInfo[parent.id] = parent
	}
	// Check if the child is in the map
	if _, ok := data.Get(edge.child.id); ok {
		// If it is, do nothing
	} else {
		// If it isn't, add it to the map
		data.Set(edge.child.id, []textEdge{})
	}
	if _, exists := nodeInfo[edge.child.id]; !exists {
		nodeInfo[edge.child.id] = edge.child
	}
}

func (gp *graphProperties) parseString(line string) ([]textNode, error) {
	log.Debugf("Parsing line: %v", line)
	var lhs, rhs []textNode
	var err error
	// Patterns are matched in order
	patterns := []struct {
		regex   *regexp.Regexp
		handler func([]string) ([]textNode, error)
	}{
		{
			regex: regexp.MustCompile(`^\s*$`),
			handler: func(match []string) ([]textNode, error) {
				// Ignore empty lines
				return []textNode{}, nil
			},
		},
		// --- Edge patterns with labels (must come before patterns without labels) ---
		// Dotted arrow with label: -.->|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+-\.->\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], DottedArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Thick arrow with label: ==>|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+==>\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], ThickArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Bidirectional arrow with label: <-->|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+<-->\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], BidirectionalArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Cross end with label: --x|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+--x\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], CrossEnd, gp.data, gp.nodeInfo), nil
			},
		},
		// Circle end with label: --o|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+--o\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], CircleEnd, gp.data, gp.nodeInfo), nil
			},
		},
		// Solid arrow with label: -->|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+-->\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], SolidArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Solid line with label: ---|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+---\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], SolidLine, gp.data, gp.nodeInfo), nil
			},
		},
		// Dotted line with label: -.-|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+-\.-\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], DottedLine, gp.data, gp.nodeInfo), nil
			},
		},
		// Thick line with label: ===|text|
		{
			regex: regexp.MustCompile(`^(.+)\s+===\|(.+)\|\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[2]); err != nil {
					rhs = []textNode{parseNode(match[2])}
				}
				return setArrowWithLabelOfType(lhs, rhs, match[1], ThickLine, gp.data, gp.nodeInfo), nil
			},
		},
		// --- Edge patterns without labels ---
		// Dotted arrow: -.->
		{
			regex: regexp.MustCompile(`^(.+)\s+-\.->\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, DottedArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Thick arrow: ==>
		{
			regex: regexp.MustCompile(`^(.+)\s+==>\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, ThickArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Bidirectional arrow: <-->
		{
			regex: regexp.MustCompile(`^(.+)\s+<-->\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, BidirectionalArrow, gp.data, gp.nodeInfo), nil
			},
		},
		// Cross end: --x
		{
			regex: regexp.MustCompile(`^(.+)\s+--x\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, CrossEnd, gp.data, gp.nodeInfo), nil
			},
		},
		// Circle end: --o
		{
			regex: regexp.MustCompile(`^(.+)\s+--o\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, CircleEnd, gp.data, gp.nodeInfo), nil
			},
		},
		// Solid arrow: -->
		{
			regex: regexp.MustCompile(`^(.+)\s+-->\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrow(lhs, rhs, gp.data, gp.nodeInfo), nil
			},
		},
		// Solid line (no arrow): ---
		{
			regex: regexp.MustCompile(`^(.+)\s+---\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, SolidLine, gp.data, gp.nodeInfo), nil
			},
		},
		// Dotted line (no arrow): -.-
		{
			regex: regexp.MustCompile(`^(.+)\s+-\.-\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, DottedLine, gp.data, gp.nodeInfo), nil
			},
		},
		// Thick line (no arrow): ===
		{
			regex: regexp.MustCompile(`^(.+)\s+===\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				if lhs, err = gp.parseString(match[0]); err != nil {
					lhs = []textNode{parseNode(match[0])}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					rhs = []textNode{parseNode(match[1])}
				}
				return setArrowOfType(lhs, rhs, ThickLine, gp.data, gp.nodeInfo), nil
			},
		},
		{
			regex: regexp.MustCompile(`^classDef\s+(.+)\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				s := parseStyleClass(match)
				(*gp.styleClasses)[s.name] = s
				return []textNode{}, nil
			},
		},
		// Inline style directive: style nodeId fill:#f9f,...
		{
			regex: regexp.MustCompile(`^\s*style\s+(\S+)\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				nodeID := match[0]
				styles := match[1]
				// Create a unique anonymous style class for this node
				anonClassName := "_style_" + nodeID
				styleMap := make(map[string]string)
				for _, style := range strings.Split(styles, ",") {
					kv := strings.Split(style, ":")
					if len(kv) == 2 {
						styleMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
					}
				}
				(*gp.styleClasses)[anonClassName] = styleClass{anonClassName, styleMap}
				// Apply the style class to the node
				if info, exists := gp.nodeInfo[nodeID]; exists {
					info.styleClass = anonClassName
					gp.nodeInfo[nodeID] = info
				} else {
					// Node not yet defined; create it
					node := textNode{id: nodeID, name: nodeID, styleClass: anonClassName, shape: shapeRect}
					addNode(node, gp.data, gp.nodeInfo)
				}
				return []textNode{}, nil
			},
		},
		// linkStyle directive: linkStyle N stroke:color,...
		{
			regex: regexp.MustCompile(`^\s*linkStyle\s+(\d+)\s+(.+)$`),
			handler: func(match []string) ([]textNode, error) {
				// Parse without error but don't apply visually (edge coloring is limited in ASCII)
				log.Debugf("linkStyle directive parsed: index=%s styles=%s", match[0], match[1])
				return []textNode{}, nil
			},
		},
		{
			regex: regexp.MustCompile(`^(.+) & (.+)$`),
			handler: func(match []string) ([]textNode, error) {
				log.Debugf("Found & pattern node %v to %v", match[0], match[1])
				var node textNode
				if lhs, err = gp.parseString(match[0]); err != nil {
					node = parseNode(match[0])
					lhs = []textNode{node}
				}
				if rhs, err = gp.parseString(match[1]); err != nil {
					node = parseNode(match[1])
					rhs = []textNode{node}
				}
				return append(lhs, rhs...), nil
			},
		},
	}
	for _, pattern := range patterns {
		if match := pattern.regex.FindStringSubmatch(line); match != nil {
			nodes, err := pattern.handler(match[1:])
			if err == nil {
				return nodes, nil
			}
		}
	}
	return []textNode{}, errors.New("Could not parse line: " + line)
}

func mermaidFileToMap(mermaid, styleType string) (*graphProperties, error) {
	// Allow split on both \n and the actual string "\n" for curl compatibility
	newlinePattern := regexp.MustCompile(`\n|\\n`)
	rawLines := newlinePattern.Split(string(mermaid), -1)

	// Process lines to remove comments
	lines := []string{}
	for _, line := range rawLines {
		// Stop processing at "---" separator (used in test files)
		if line == "---" {
			break
		}

		// Skip lines that start with %% (comment lines)
		if strings.HasPrefix(strings.TrimSpace(line), "%%") {
			continue
		}

		// Remove inline comments (anything after %%) and trim resulting whitespace
		if idx := strings.Index(line, "%%"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Skip empty lines after comment removal
		if len(strings.TrimSpace(line)) > 0 {
			lines = append(lines, line)
		}
	}

	data := orderedmap.NewOrderedMap[string, []textEdge]()
	styleClasses := make(map[string]styleClass)
	nodeInfo := make(map[string]textNode)
	properties := graphProperties{
		data:             data,
		nodeInfo:         nodeInfo,
		styleClasses:     &styleClasses,
		graphDirection:   "",
		styleType:        styleType,
		paddingX:         5,
		paddingY:         5,
		subgraphs:        []*textSubgraph{},
		boxBorderPadding: 1,
	}

	// Pick up optional padding directives before the graph definition
	paddingRegex := regexp.MustCompile(`^(?i)padding([xy])\s*=\s*(\d+)$`)
	for len(lines) > 0 {
		trimmed := strings.TrimSpace(lines[0])
		if trimmed == "" {
			lines = lines[1:]
			continue
		}
		if match := paddingRegex.FindStringSubmatch(trimmed); match != nil {
			paddingValue, err := strconv.Atoi(match[2])
			if err != nil {
				return &properties, err
			}
			if strings.EqualFold(match[1], "x") {
				properties.paddingX = paddingValue
			} else {
				properties.paddingY = paddingValue
			}
			lines = lines[1:]
			continue
		}
		break
	}

	if len(lines) == 0 {
		return &properties, errors.New("missing graph definition")
	}

	// First line should either say "graph TD" or "graph LR"
	switch lines[0] {
	case "graph LR", "flowchart LR":
		properties.graphDirection = "LR"
	case "graph TD", "flowchart TD", "graph TB", "flowchart TB":
		properties.graphDirection = "TD"
	case "graph BT", "flowchart BT":
		properties.graphDirection = "BT"
	case "graph RL", "flowchart RL":
		properties.graphDirection = "RL"
	default:
		return &properties, fmt.Errorf("unsupported graph type '%s'. Supported types: graph TD, graph TB, graph LR, flowchart TD, flowchart TB, flowchart LR, graph BT, flowchart BT, graph RL, flowchart RL", lines[0])
	}
	lines = lines[1:]

	// Track subgraph context using a stack
	subgraphStack := []*textSubgraph{}
	// Supports: "subgraph id [title]" or "subgraph title"
	subgraphRegex := regexp.MustCompile(`^\s*subgraph\s+(\S+)(?:\s+\[(.+)\])?\s*$`)
	endRegex := regexp.MustCompile(`^\s*end\s*$`)
	directionRegex := regexp.MustCompile(`^\s*direction\s+(LR|TD|TB|BT|RL)\s*$`)

	// Iterate over the lines
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for subgraph start
		if match := subgraphRegex.FindStringSubmatch(trimmedLine); match != nil {
			var subgraphID, subgraphName string
			if match[2] != "" {
				// "subgraph id [title]" form
				subgraphID = strings.TrimSpace(match[1])
				subgraphName = strings.TrimSpace(match[2])
			} else {
				// "subgraph title" form - title is both id and name
				subgraphID = strings.TrimSpace(match[1])
				subgraphName = subgraphID
			}
			newSubgraph := &textSubgraph{
				id:       subgraphID,
				name:     subgraphName,
				nodes:    []string{},
				children: []*textSubgraph{},
			}

			// Set parent relationship if we're nested
			if len(subgraphStack) > 0 {
				parent := subgraphStack[len(subgraphStack)-1]
				newSubgraph.parent = parent
				parent.children = append(parent.children, newSubgraph)
			}

			subgraphStack = append(subgraphStack, newSubgraph)
			properties.subgraphs = append(properties.subgraphs, newSubgraph)
			log.Debugf("Started subgraph id=%s name=%s", subgraphID, subgraphName)
			continue
		}

		// Check for subgraph end
		if endRegex.MatchString(trimmedLine) {
			if len(subgraphStack) > 0 {
				closedSubgraph := subgraphStack[len(subgraphStack)-1]
				subgraphStack = subgraphStack[:len(subgraphStack)-1]
				log.Debugf("Ended subgraph %s", closedSubgraph.name)
			}
			continue
		}

		// Check for direction directive inside a subgraph
		if match := directionRegex.FindStringSubmatch(trimmedLine); match != nil {
			if len(subgraphStack) > 0 {
				currentSubgraph := subgraphStack[len(subgraphStack)-1]
				currentSubgraph.direction = match[1]
				log.Debugf("Set direction %s for subgraph %s", match[1], currentSubgraph.name)
			}
			continue
		}

		// Remember nodes before parsing this line
		existingNodes := make(map[string]bool)
		for el := data.Front(); el != nil; el = el.Next() {
			existingNodes[el.Key] = true
		}

		// Parse nodes and edges normally
		nodes, err := properties.parseString(line)
		if err != nil {
			log.Debugf("Parsing remaining text to node %v", line)
			node := parseNode(line)
			addNode(node, properties.data, properties.nodeInfo)
		} else {
			// Ensure all returned nodes are in the map
			for _, node := range nodes {
				addNode(node, properties.data, properties.nodeInfo)
			}
		}

		// Add all new nodes to current subgraph(s)
		if len(subgraphStack) > 0 {
			for el := data.Front(); el != nil; el = el.Next() {
				nodeName := el.Key
				// If this is a new node (wasn't in existingNodes), add it to subgraph
				if !existingNodes[nodeName] {
					for _, sg := range subgraphStack {
						// Check if node is not already in the subgraph
						found := false
						for _, n := range sg.nodes {
							if n == nodeName {
								found = true
								break
							}
						}
						if !found {
							sg.nodes = append(sg.nodes, nodeName)
							log.Debugf("Added node %s to subgraph %s", nodeName, sg.name)
						}
					}
				}
			}
		}
	}
	// Resolve edges that reference subgraph IDs: route them to the first node
	// in the referenced subgraph.
	properties.resolveSubgraphEdges()

	// Apply "classDef default" to all nodes that don't have a class already
	if _, hasDefault := styleClasses["default"]; hasDefault {
		for nodeID, info := range properties.nodeInfo {
			if info.styleClass == "" {
				info.styleClass = "default"
				properties.nodeInfo[nodeID] = info
			}
		}
	}

	return &properties, nil
}

// resolveSubgraphEdges rewrites edges that reference subgraph IDs so they
// point to the first node inside that subgraph instead.
func (gp *graphProperties) resolveSubgraphEdges() {
	if len(gp.subgraphs) == 0 {
		return
	}

	// Build a set of subgraph IDs for quick lookup
	sgIDs := make(map[string]bool)
	for _, sg := range gp.subgraphs {
		sgIDs[sg.id] = true
	}

	// For each key in the data map that is a subgraph ID, move its edges
	// to the resolved node.
	keysToResolve := []string{}
	for el := gp.data.Front(); el != nil; el = el.Next() {
		if sgIDs[el.Key] {
			keysToResolve = append(keysToResolve, el.Key)
		}
	}
	for _, key := range keysToResolve {
		resolved := gp.resolveSubgraphNode(key)
		if resolved != key {
			edges, _ := gp.data.Get(key)
			gp.data.Delete(key)
			if existingEdges, ok := gp.data.Get(resolved); ok {
				gp.data.Set(resolved, append(existingEdges, edges...))
			} else {
				gp.data.Set(resolved, edges)
			}
			// Update nodeInfo: ensure resolved node exists
			if _, exists := gp.nodeInfo[resolved]; !exists {
				gp.nodeInfo[resolved] = textNode{id: resolved, name: resolved, shape: shapeRect}
			}
			delete(gp.nodeInfo, key)
			log.Debugf("Resolved subgraph edge source %s -> node %s", key, resolved)
		}
	}

	// Also resolve child references within edges
	for el := gp.data.Front(); el != nil; el = el.Next() {
		edges := el.Value
		for i, edge := range edges {
			resolvedChild := gp.resolveSubgraphNode(edge.child.id)
			if resolvedChild != edge.child.id {
				if info, exists := gp.nodeInfo[resolvedChild]; exists {
					edges[i].child = info
				} else {
					edges[i].child = textNode{id: resolvedChild, name: resolvedChild, shape: shapeRect}
					gp.nodeInfo[resolvedChild] = edges[i].child
				}
				// Ensure resolved child is in data map
				if _, ok := gp.data.Get(resolvedChild); !ok {
					gp.data.Set(resolvedChild, []textEdge{})
				}
				log.Debugf("Resolved subgraph edge target %s -> node %s", edge.child.id, resolvedChild)
			}
		}
		gp.data.Set(el.Key, edges)
	}
}
