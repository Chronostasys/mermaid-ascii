// Package blockdiagram implements parsing and rendering of block-beta diagrams
// in Mermaid syntax.
package blockdiagram

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pgavlin/mermaid-ascii/pkg/diagram"
)

// BlockBetaKeyword is the Mermaid keyword that identifies a block-beta diagram.
const BlockBetaKeyword = "block-beta"

var (
	columnsRegex    = regexp.MustCompile(`^\s*columns\s+(\d+)\s*$`)
	blockStartRegex = regexp.MustCompile(`^\s*block\s*(?::(\S+))?\s*$`)
	blockEndRegex   = regexp.MustCompile(`^\s*end\s*$`)
	// Block name with optional label in various shape syntaxes, plus optional span
	// Supported shapes: ["text"], ("text"), (["text"]), [["text"]], [("text")], (("text"))
	blockNameRegex = regexp.MustCompile(`^\s*(\S+?)(?:\(\["([^"]+)"\]\)|\[\["([^"]+)"\]\]|\[\("([^"]+)"\)\]|\(\("([^"]+)"\)\)|\["([^"]+)"\]|\("([^"]+)"\))?\s*(?::(\d+))?\s*$`)
	// Edge pattern: --> or -- "label" -->
	edgeRegex = regexp.MustCompile(`\s+(?:-->\s+|--\s+"[^"]*"\s+-->\s+)`)
)

// Block represents a single block in the diagram.
type Block struct {
	ID       string
	Label    string
	Children []*Block
	Columns  int  // number of columns for this container block
	Span     int  // how many columns this block spans
	IsSpace  bool // true for spacer blocks that render as empty space
}

// BlockDiagram represents a parsed block diagram.
type BlockDiagram struct {
	Columns int
	Blocks  []*Block
}

// IsBlockDiagram returns true if the input starts with block-beta keyword.
func IsBlockDiagram(input string) bool {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		return strings.HasPrefix(trimmed, BlockBetaKeyword)
	}
	return false
}

// Parse parses a block diagram.
func Parse(input string) (*BlockDiagram, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	rawLines := diagram.SplitLines(input)
	lines := diagram.RemoveComments(rawLines)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no content found")
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[0]), BlockBetaKeyword) {
		return nil, fmt.Errorf("expected %q keyword", BlockBetaKeyword)
	}

	d := &BlockDiagram{Columns: 1}

	_, err := parseBlockLines(d, lines[1:], nil)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// tokenizeBlockLine splits a line into block tokens, respecting bracket syntax.
// E.g., `A["text with spaces"] B` → ["A[\"text with spaces\"]", "B"]
func tokenizeBlockLine(line string) []string {
	var tokens []string
	var current strings.Builder
	depth := 0 // track bracket/paren nesting
	inQuote := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			inQuote = !inQuote
			current.WriteByte(ch)
			continue
		}
		if inQuote {
			current.WriteByte(ch)
			continue
		}
		if ch == '[' || ch == '(' {
			depth++
			current.WriteByte(ch)
			continue
		}
		if ch == ']' || ch == ')' {
			depth--
			current.WriteByte(ch)
			continue
		}
		if ch == ' ' || ch == '\t' {
			if depth == 0 {
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				continue
			}
		}
		current.WriteByte(ch)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// parseBlockToken parses a single token into a Block.
// Returns nil if the token is not a valid block reference.
func parseBlockToken(token string) *Block {
	// Check for space keyword (with optional span)
	spaceToken := strings.TrimSpace(token)
	if spaceToken == "space" || strings.HasPrefix(spaceToken, "space:") {
		span := 1
		if idx := strings.Index(spaceToken, ":"); idx >= 0 {
			if s, err := strconv.Atoi(spaceToken[idx+1:]); err == nil && s > 0 {
				span = s
			}
		}
		return &Block{
			ID:      "space",
			Label:   "",
			Span:    span,
			IsSpace: true,
		}
	}

	if m := blockNameRegex.FindStringSubmatch(token); m != nil {
		id := m[1]
		label := id
		// Check capture groups in order: (["text"]), [["text"]], [("text")], (("text")), ["text"], ("text")
		for _, g := range []int{2, 3, 4, 5, 6, 7} {
			if m[g] != "" {
				label = m[g]
				break
			}
		}
		span := 1
		if m[8] != "" {
			span, _ = strconv.Atoi(m[8])
		}
		return &Block{
			ID:    id,
			Label: label,
			Span:  span,
		}
	}
	return nil
}

// addBlock adds a block to the parent or diagram top-level.
func addBlock(d *BlockDiagram, parent *Block, b *Block) {
	if parent != nil {
		parent.Children = append(parent.Children, b)
	} else {
		d.Blocks = append(d.Blocks, b)
	}
}

func parseBlockLines(d *BlockDiagram, lines []string, parent *Block) (int, error) {
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			i++
			continue
		}

		// columns directive
		if m := columnsRegex.FindStringSubmatch(trimmed); m != nil {
			cols, _ := strconv.Atoi(m[1])
			if parent != nil {
				parent.Columns = cols
			} else {
				d.Columns = cols
			}
			i++
			continue
		}

		// end of block
		if blockEndRegex.MatchString(trimmed) {
			return i + 1, nil
		}

		// block start
		if m := blockStartRegex.FindStringSubmatch(trimmed); m != nil {
			b := &Block{
				ID:      m[1],
				Label:   m[1],
				Columns: 1,
				Span:    1,
			}
			if b.ID == "" {
				b.ID = fmt.Sprintf("block_%d", i)
				b.Label = ""
			}
			i++
			consumed, err := parseBlockLines(d, lines[i:], b)
			if err != nil {
				return 0, err
			}
			i += consumed
			addBlock(d, parent, b)
			continue
		}

		// Edge syntax: A --> B, A -- "label" --> B, A["Start"] --> B["Stop"]
		if edgeRegex.MatchString(trimmed) {
			parts := edgeRegex.Split(trimmed, -1)
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				// Each part may itself contain multiple tokens
				tokens := tokenizeBlockLine(part)
				for _, tok := range tokens {
					if b := parseBlockToken(tok); b != nil {
						addBlock(d, parent, b)
					}
				}
			}
			i++
			continue
		}

		// Try tokenizing the line - handles both single and multi-block lines
		tokens := tokenizeBlockLine(trimmed)
		if len(tokens) > 0 {
			parsed := false
			for _, tok := range tokens {
				if b := parseBlockToken(tok); b != nil {
					addBlock(d, parent, b)
					parsed = true
				}
			}
			if parsed {
				i++
				continue
			}
		}

		i++
	}
	return i, nil
}
