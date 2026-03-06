// Package packet implements parsing and rendering of packet/protocol diagrams
// in Mermaid syntax.
package packet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pgavlin/mermaid-ascii/pkg/diagram"
)

// PacketKeyword is the Mermaid keyword that identifies a packet diagram.
const PacketKeyword = "packet-beta"

var (
	fieldRegex = regexp.MustCompile(`^\s*(\d+)(?:-(\d+))?\s*:\s*"([^"]+)"\s*$`)
)

// PacketDiagram represents a parsed packet/protocol diagram.
type PacketDiagram struct {
	Fields []*Field
}

// Field represents a single field in a packet diagram, spanning one or more bits.
type Field struct {
	StartBit int
	EndBit   int
	Label    string
}

// IsPacketDiagram returns true if the input starts with the packet-beta keyword.
func IsPacketDiagram(input string) bool {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		return trimmed == PacketKeyword
	}
	return false
}

// Parse parses a packet diagram from Mermaid-style input.
func Parse(input string) (*PacketDiagram, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	rawLines := diagram.SplitLines(input)
	lines := diagram.RemoveComments(rawLines)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no content found")
	}

	if strings.TrimSpace(lines[0]) != PacketKeyword {
		return nil, fmt.Errorf("expected %q keyword", PacketKeyword)
	}
	lines = lines[1:]

	pd := &PacketDiagram{Fields: []*Field{}}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if match := fieldRegex.FindStringSubmatch(trimmed); match != nil {
			startBit, _ := strconv.Atoi(match[1])
			endBit := startBit
			if match[2] != "" {
				endBit, _ = strconv.Atoi(match[2])
			}
			pd.Fields = append(pd.Fields, &Field{
				StartBit: startBit,
				EndBit:   endBit,
				Label:    match[3],
			})
		}
	}

	if len(pd.Fields) == 0 {
		return nil, fmt.Errorf("no fields found")
	}

	return pd, nil
}
