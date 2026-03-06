package canvas

// BoxChars holds characters for drawing boxes.
type BoxChars struct {
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
	Horizontal  rune
	Vertical    rune
	TeeLeft     rune
	TeeRight    rune
	TeeUp       rune
	TeeDown     rune
	Cross       rune
}

// UnicodeBox provides Unicode box-drawing characters.
var UnicodeBox = BoxChars{
	TopLeft:     '┌',
	TopRight:    '┐',
	BottomLeft:  '└',
	BottomRight: '┘',
	Horizontal:  '─',
	Vertical:    '│',
	TeeLeft:     '┤',
	TeeRight:    '├',
	TeeUp:       '┴',
	TeeDown:     '┬',
	Cross:       '┼',
}

// ASCIIBox provides plain ASCII box-drawing characters.
var ASCIIBox = BoxChars{
	TopLeft:     '+',
	TopRight:    '+',
	BottomLeft:  '+',
	BottomRight: '+',
	Horizontal:  '-',
	Vertical:    '|',
	TeeLeft:     '+',
	TeeRight:    '+',
	TeeUp:       '+',
	TeeDown:     '+',
	Cross:       '+',
}
