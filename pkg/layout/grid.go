package layout

// Coord represents a 2D coordinate.
type Coord struct {
	X, Y int
}

// Grid manages placement of items on a 2D grid.
type Grid struct {
	Occupied  map[Coord]bool
	ColWidth  map[int]int
	RowHeight map[int]int
}

// NewGrid creates a new empty grid.
func NewGrid() *Grid {
	return &Grid{
		Occupied:  make(map[Coord]bool),
		ColWidth:  make(map[int]int),
		RowHeight: make(map[int]int),
	}
}

// IsOccupied checks if a grid position is taken.
func (g *Grid) IsOccupied(c Coord) bool {
	return g.Occupied[c]
}

// Reserve marks a rectangular area as occupied.
func (g *Grid) Reserve(topLeft Coord, width, height int) {
	for dx := 0; dx < width; dx++ {
		for dy := 0; dy < height; dy++ {
			g.Occupied[Coord{X: topLeft.X + dx, Y: topLeft.Y + dy}] = true
		}
	}
}

// SetColWidth sets the minimum width for a column.
func (g *Grid) SetColWidth(col, width int) {
	if width > g.ColWidth[col] {
		g.ColWidth[col] = width
	}
}

// SetRowHeight sets the minimum height for a row.
func (g *Grid) SetRowHeight(row, height int) {
	if height > g.RowHeight[row] {
		g.RowHeight[row] = height
	}
}

// GridToPixel converts a grid coordinate to pixel position based on accumulated column/row sizes.
func (g *Grid) GridToPixel(c Coord) Coord {
	x := 0
	for col := 0; col < c.X; col++ {
		x += g.ColWidth[col]
	}
	y := 0
	for row := 0; row < c.Y; row++ {
		y += g.RowHeight[row]
	}
	return Coord{X: x + g.ColWidth[c.X]/2, Y: y + g.RowHeight[c.Y]/2}
}

// FindFreeSpot finds the next available spot starting from the given coordinate, advancing in the given direction.
func (g *Grid) FindFreeSpot(start Coord, nodeWidth, nodeHeight, step int, horizontal bool) Coord {
	c := start
	for g.IsOccupied(c) {
		if horizontal {
			c.X += step
		} else {
			c.Y += step
		}
	}
	return c
}
