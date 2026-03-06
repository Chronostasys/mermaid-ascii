// Package zenuml implements parsing and rendering of ZenUML sequence diagrams
// in Mermaid syntax.
// Package zenuml implements parsing and rendering of ZenUML sequence diagrams
// in Mermaid syntax.
package zenuml

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pgavlin/mermaid-ascii/pkg/diagram"
)

// ZenUMLKeyword is the Mermaid keyword that identifies a ZenUML diagram.
// ZenUMLKeyword is the Mermaid keyword that identifies a ZenUML diagram.
const ZenUMLKeyword = "zenuml"

var (
	// participantDeclRegex matches "Type Name" declarations like "Client client"
	participantDeclRegex = regexp.MustCompile(`^\s*(\w+)\s+(\w+)\s*$`)

	// syncMessageRegex matches "target.method(args)" without trailing brace
	syncMessageRegex = regexp.MustCompile(`^\s*(\w+)\.(\w+)\(([^)]*)\)\s*$`)

	// asyncMessageRegex matches "target.method(args) {" (async block start)
	asyncMessageRegex = regexp.MustCompile(`^\s*(\w+)\.(\w+)\(([^)]*)\)\s*\{\s*$`)

	// returnRegex matches "return value"
	returnRegex = regexp.MustCompile(`^\s*return\s+(.*?)\s*$`)

	// closeBraceRegex matches a closing brace
	closeBraceRegex = regexp.MustCompile(`^\s*\}\s*$`)

	// arrowMessageRegex matches "A->B: message text"
	arrowMessageRegex = regexp.MustCompile(`^\s*(\w+)\s*->\s*(\w+)\s*:\s*(.+?)\s*$`)

	// arrowBlockRegex matches "A->B: message {"
	arrowBlockRegex = regexp.MustCompile(`^\s*(\w+)\s*->\s*(\w+)\s*:\s*(.+?)\s*\{\s*$`)

	// singleParticipantRegex matches a standalone single word (implicit participant)
	singleParticipantRegex = regexp.MustCompile(`^\s*(\w+)\s*$`)

	// aliasRegex matches "A as Alice"
	aliasRegex = regexp.MustCompile(`^\s*(\w+)\s+as\s+(\w+)\s*$`)

	// annotatorRegex matches "@Actor Alice" or "@Database Bob"
	annotatorRegex = regexp.MustCompile(`^\s*@(\w+)\s+(\w+)\s*$`)

	// controlFlowRegex matches "while(cond) {", "if(cond) {", "opt {", etc.
	controlFlowRegex = regexp.MustCompile(`^\s*(while|if|for|forEach|loop|opt|par|try)\s*(\([^)]*\))?\s*\{\s*$`)

	// continuationRegex matches "} else {", "} catch {", "} finally {"
	continuationRegex = regexp.MustCompile(`^\s*\}\s*(else|catch|finally)\s*(\([^)]*\))?\s*\{\s*$`)

	// newRegex matches "new A1" or "new A2(args)"
	newRegex = regexp.MustCompile(`^\s*new\s+(\w+)(?:\(([^)]*)\))?\s*$`)
)

// MessageType distinguishes sync, async, and return messages.
type MessageType int

const (
	// SyncMessage represents a synchronous message call.
	SyncMessage MessageType = iota
	// AsyncMessage represents an asynchronous message call.
	AsyncMessage
	// ReturnMessage represents a return message from a call.
	ReturnMessage
)

// Participant represents a participant in the ZenUML diagram.
type Participant struct {
	TypeName string // declared type, e.g. "Client"
	ID       string // identifier, e.g. "client"
	Index    int
}

// Message represents a message/call in the ZenUML diagram.
type Message struct {
	From   *Participant
	To     *Participant
	Method string
	Args   string
	Type   MessageType
	Label  string     // used for return value text
	Nested []*Message // nested messages inside async blocks
}

// ZenUMLDiagram represents a parsed ZenUML diagram.
type ZenUMLDiagram struct {
	Participants []*Participant
	Messages     []*Message
}

// IsZenUML returns true if the input starts with the zenuml keyword.
func IsZenUML(input string) bool {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%%") {
			continue
		}
		return strings.HasPrefix(trimmed, ZenUMLKeyword)
	}
	return false
}

// Parse parses ZenUML input text into a ZenUMLDiagram.
func Parse(input string) (*ZenUMLDiagram, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	rawLines := diagram.SplitLines(input)
	lines := diagram.RemoveComments(rawLines)
	if len(lines) == 0 {
		return nil, fmt.Errorf("no content found")
	}

	if !strings.HasPrefix(strings.TrimSpace(lines[0]), ZenUMLKeyword) {
		return nil, fmt.Errorf("expected %q keyword", ZenUMLKeyword)
	}
	lines = lines[1:]

	d := &ZenUMLDiagram{
		Participants: []*Participant{},
		Messages:     []*Message{},
	}
	participantMap := make(map[string]*Participant)

	messages, _, err := parseLines(lines, d, participantMap, false)
	if err != nil {
		return nil, err
	}
	d.Messages = messages

	if len(d.Participants) == 0 {
		return nil, fmt.Errorf("no participants found")
	}

	return d, nil
}

// parseLines parses lines into messages. When inBlock is true, parsing stops at '}'.
func parseLines(lines []string, d *ZenUMLDiagram, pMap map[string]*Participant, inBlock bool) ([]*Message, int, error) {
	var messages []*Message
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])

		// 1. Empty line
		if trimmed == "" {
			i++
			continue
		}

		// 2. Block end — plain '}'
		if inBlock && closeBraceRegex.MatchString(trimmed) {
			return messages, i, nil
		}

		// 2b. Continuation — '} else {', '} catch {', '} finally {'
		// When inside a block, a continuation means the current block is done.
		if inBlock && continuationRegex.MatchString(trimmed) {
			return messages, i, nil
		}

		// 3. Return statement
		if match := returnRegex.FindStringSubmatch(trimmed); match != nil {
			var from, to *Participant
			if len(messages) > 0 {
				last := messages[len(messages)-1]
				from = last.To
				to = last.From
			} else if len(d.Participants) > 0 {
				from = d.Participants[0]
			}
			msg := &Message{
				From:  from,
				To:    to,
				Label: strings.TrimSpace(match[1]),
				Type:  ReturnMessage,
			}
			messages = append(messages, msg)
			i++
			continue
		}

		// 4. Arrow message with block: A->B: message {
		if match := arrowBlockRegex.FindStringSubmatch(trimmed); match != nil {
			fromP := getOrCreateParticipant(match[1], d, pMap)
			toP := getOrCreateParticipant(match[2], d, pMap)
			label := strings.TrimSpace(match[3])

			i++
			nested, consumed, err := parseLines(lines[i:], d, pMap, true)
			if err != nil {
				return nil, 0, err
			}
			i += consumed
			if i < len(lines) {
				i++ // consume '}'
			}

			msg := &Message{
				From:   fromP,
				To:     toP,
				Label:  label,
				Type:   AsyncMessage,
				Nested: nested,
			}
			messages = append(messages, msg)
			continue
		}

		// 5. Arrow message: A->B: message text
		if match := arrowMessageRegex.FindStringSubmatch(trimmed); match != nil {
			fromP := getOrCreateParticipant(match[1], d, pMap)
			toP := getOrCreateParticipant(match[2], d, pMap)
			label := strings.TrimSpace(match[3])

			msg := &Message{
				From:  fromP,
				To:    toP,
				Label: label,
				Type:  SyncMessage,
			}
			messages = append(messages, msg)
			i++
			continue
		}

		// 6. Async message: target.method(args) {
		if match := asyncMessageRegex.FindStringSubmatch(trimmed); match != nil {
			target := match[1]
			method := match[2]
			args := strings.TrimSpace(match[3])

			to := getOrCreateParticipant(target, d, pMap)
			from := inferCaller(d, to)

			i++
			nested, consumed, err := parseLines(lines[i:], d, pMap, true)
			if err != nil {
				return nil, 0, err
			}
			i += consumed
			if i < len(lines) {
				i++ // consume '}'
			}

			msg := &Message{
				From:   from,
				To:     to,
				Method: method,
				Args:   args,
				Type:   AsyncMessage,
				Nested: nested,
			}
			messages = append(messages, msg)
			continue
		}

		// 7. Sync message: target.method(args)
		if match := syncMessageRegex.FindStringSubmatch(trimmed); match != nil {
			target := match[1]
			method := match[2]
			args := strings.TrimSpace(match[3])

			to := getOrCreateParticipant(target, d, pMap)
			from := inferCaller(d, to)

			msg := &Message{
				From:   from,
				To:     to,
				Method: method,
				Args:   args,
				Type:   SyncMessage,
			}
			messages = append(messages, msg)
			i++
			continue
		}

		// 8. Control flow block: while(cond) {, if(cond) {, opt {, etc.
		if controlFlowRegex.MatchString(trimmed) {
			i++ // consume the opening line
			nested, consumed, err := parseLines(lines[i:], d, pMap, true)
			if err != nil {
				return nil, 0, err
			}
			i += consumed
			messages = append(messages, nested...)

			// consume the closing '}' or continuation
			if i < len(lines) {
				closeLine := strings.TrimSpace(lines[i])
				if closeBraceRegex.MatchString(closeLine) {
					i++ // consume plain '}'
				}
				// Check for continuations: } else {, } catch {, } finally {
				for i < len(lines) {
					cl := strings.TrimSpace(lines[i])
					if continuationRegex.MatchString(cl) {
						i++ // consume the continuation line
						nested2, consumed2, err := parseLines(lines[i:], d, pMap, true)
						if err != nil {
							return nil, 0, err
						}
						i += consumed2
						messages = append(messages, nested2...)
						// consume the closing '}'
						if i < len(lines) && closeBraceRegex.MatchString(strings.TrimSpace(lines[i])) {
							i++
						}
					} else {
						break
					}
				}
			}
			continue
		}

		// 9. New/create: new A1 or new A2(args)
		if match := newRegex.FindStringSubmatch(trimmed); match != nil {
			target := match[1]
			args := ""
			if len(match) > 2 {
				args = strings.TrimSpace(match[2])
			}
			to := getOrCreateParticipant(target, d, pMap)
			from := inferCaller(d, to)

			msg := &Message{
				From:   from,
				To:     to,
				Method: "new",
				Args:   args,
				Type:   SyncMessage,
			}
			messages = append(messages, msg)
			i++
			continue
		}

		// 10. Annotator: @Actor Alice, @Database Bob
		if match := annotatorRegex.FindStringSubmatch(trimmed); match != nil {
			typeName := match[1]
			id := match[2]
			if _, exists := pMap[id]; !exists {
				p := &Participant{
					TypeName: typeName,
					ID:       id,
					Index:    len(d.Participants),
				}
				d.Participants = append(d.Participants, p)
				pMap[id] = p
			}
			i++
			continue
		}

		// 11. Alias: A as Alice
		if match := aliasRegex.FindStringSubmatch(trimmed); match != nil {
			id := match[1]
			displayName := match[2]
			if _, exists := pMap[id]; !exists {
				p := &Participant{
					TypeName: displayName,
					ID:       id,
					Index:    len(d.Participants),
				}
				d.Participants = append(d.Participants, p)
				pMap[id] = p
			}
			i++
			continue
		}

		// 12. Participant declaration: Type Name (two words, first not reserved)
		if match := participantDeclRegex.FindStringSubmatch(trimmed); match != nil {
			typeName := match[1]
			id := match[2]

			if isReservedWord(typeName) {
				return nil, 0, fmt.Errorf("unexpected line: %q", trimmed)
			}

			// "as" in second position is handled by aliasRegex above
			if id == "as" {
				return nil, 0, fmt.Errorf("unexpected line: %q", trimmed)
			}

			if _, exists := pMap[id]; !exists {
				p := &Participant{
					TypeName: typeName,
					ID:       id,
					Index:    len(d.Participants),
				}
				d.Participants = append(d.Participants, p)
				pMap[id] = p
			}
			i++
			continue
		}

		// 13. Standalone participant: single word (must be last to avoid matching keywords)
		if match := singleParticipantRegex.FindStringSubmatch(trimmed); match != nil {
			id := match[1]
			if !isReservedWord(id) {
				if _, exists := pMap[id]; !exists {
					p := &Participant{
						TypeName: id,
						ID:       id,
						Index:    len(d.Participants),
					}
					d.Participants = append(d.Participants, p)
					pMap[id] = p
				}
				i++
				continue
			}
		}

		return nil, 0, fmt.Errorf("unexpected line: %q", trimmed)
	}

	return messages, i, nil
}

// inferCaller returns the first declared participant as the default caller,
// as long as it is not the same as the target. If only one participant exists,
// it returns that participant (self-call).
func inferCaller(d *ZenUMLDiagram, _ *Participant) *Participant {
	if len(d.Participants) == 0 {
		return nil
	}
	// Use first participant as default caller
	return d.Participants[0]
}

func getOrCreateParticipant(id string, d *ZenUMLDiagram, pMap map[string]*Participant) *Participant {
	if p, exists := pMap[id]; exists {
		return p
	}
	p := &Participant{
		TypeName: id,
		ID:       id,
		Index:    len(d.Participants),
	}
	d.Participants = append(d.Participants, p)
	pMap[id] = p
	return p
}

func isReservedWord(s string) bool {
	switch strings.ToLower(s) {
	case "return", "zenuml",
		"while", "if", "for", "foreach", "loop", "opt", "par", "try",
		"new", "else", "catch", "finally":
		return true
	}
	return false
}
