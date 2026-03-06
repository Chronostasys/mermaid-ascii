package layout

import (
	"testing"
)

func TestNewGrid(t *testing.T) {
	g := NewGrid()
	if g == nil {
		t.Fatal("NewGrid returned nil")
	}
	if g.Occupied == nil || g.ColWidth == nil || g.RowHeight == nil {
		t.Fatal("NewGrid did not initialize maps")
	}
}

func TestIsOccupied_Empty(t *testing.T) {
	g := NewGrid()
	if g.IsOccupied(Coord{0, 0}) {
		t.Error("empty grid should not have occupied cells")
	}
}

func TestReserve_SingleCell(t *testing.T) {
	g := NewGrid()
	g.Reserve(Coord{2, 3}, 1, 1)

	if !g.IsOccupied(Coord{2, 3}) {
		t.Error("reserved cell should be occupied")
	}
	if g.IsOccupied(Coord{1, 3}) {
		t.Error("adjacent cell should not be occupied")
	}
}

func TestReserve_RectangularArea(t *testing.T) {
	g := NewGrid()
	g.Reserve(Coord{1, 1}, 3, 2)

	// All cells in the 3x2 rectangle should be occupied
	for x := 1; x <= 3; x++ {
		for y := 1; y <= 2; y++ {
			if !g.IsOccupied(Coord{x, y}) {
				t.Errorf("cell (%d,%d) should be occupied", x, y)
			}
		}
	}

	// Cells outside should not be occupied
	if g.IsOccupied(Coord{0, 0}) {
		t.Error("cell (0,0) should not be occupied")
	}
	if g.IsOccupied(Coord{4, 1}) {
		t.Error("cell (4,1) should not be occupied")
	}
	if g.IsOccupied(Coord{1, 3}) {
		t.Error("cell (1,3) should not be occupied")
	}
}

func TestSetColWidth(t *testing.T) {
	g := NewGrid()
	g.SetColWidth(0, 10)
	if g.ColWidth[0] != 10 {
		t.Errorf("expected col 0 width 10, got %d", g.ColWidth[0])
	}

	// Setting a smaller width should not decrease it
	g.SetColWidth(0, 5)
	if g.ColWidth[0] != 10 {
		t.Errorf("expected col 0 width to remain 10, got %d", g.ColWidth[0])
	}

	// Setting a larger width should increase it
	g.SetColWidth(0, 20)
	if g.ColWidth[0] != 20 {
		t.Errorf("expected col 0 width 20, got %d", g.ColWidth[0])
	}
}

func TestSetRowHeight(t *testing.T) {
	g := NewGrid()
	g.SetRowHeight(0, 8)
	if g.RowHeight[0] != 8 {
		t.Errorf("expected row 0 height 8, got %d", g.RowHeight[0])
	}

	// Setting a smaller height should not decrease it
	g.SetRowHeight(0, 3)
	if g.RowHeight[0] != 8 {
		t.Errorf("expected row 0 height to remain 8, got %d", g.RowHeight[0])
	}

	// Setting a larger height should increase it
	g.SetRowHeight(0, 12)
	if g.RowHeight[0] != 12 {
		t.Errorf("expected row 0 height 12, got %d", g.RowHeight[0])
	}
}

func TestGridToPixel(t *testing.T) {
	g := NewGrid()
	g.SetColWidth(0, 10)
	g.SetColWidth(1, 20)
	g.SetColWidth(2, 30)
	g.SetRowHeight(0, 4)
	g.SetRowHeight(1, 6)
	g.SetRowHeight(2, 8)

	tests := []struct {
		name     string
		input    Coord
		expected Coord
	}{
		{
			name:     "origin",
			input:    Coord{0, 0},
			expected: Coord{5, 2}, // 10/2, 4/2
		},
		{
			name:     "col 1 row 0",
			input:    Coord{1, 0},
			expected: Coord{20, 2}, // 10 + 20/2, 4/2
		},
		{
			name:     "col 0 row 1",
			input:    Coord{0, 1},
			expected: Coord{5, 7}, // 10/2, 4 + 6/2
		},
		{
			name:     "col 2 row 2",
			input:    Coord{2, 2},
			expected: Coord{45, 14}, // 10+20+30/2, 4+6+8/2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.GridToPixel(tt.input)
			if result != tt.expected {
				t.Errorf("GridToPixel(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFindFreeSpot_Horizontal(t *testing.T) {
	g := NewGrid()
	g.Reserve(Coord{0, 0}, 1, 1)
	g.Reserve(Coord{1, 0}, 1, 1)

	spot := g.FindFreeSpot(Coord{0, 0}, 1, 1, 1, true)
	if spot != (Coord{2, 0}) {
		t.Errorf("expected (2,0), got %v", spot)
	}
}

func TestFindFreeSpot_Vertical(t *testing.T) {
	g := NewGrid()
	g.Reserve(Coord{0, 0}, 1, 1)
	g.Reserve(Coord{0, 1}, 1, 1)

	spot := g.FindFreeSpot(Coord{0, 0}, 1, 1, 1, false)
	if spot != (Coord{0, 2}) {
		t.Errorf("expected (0,2), got %v", spot)
	}
}

func TestFindFreeSpot_AlreadyFree(t *testing.T) {
	g := NewGrid()

	spot := g.FindFreeSpot(Coord{0, 0}, 1, 1, 1, true)
	if spot != (Coord{0, 0}) {
		t.Errorf("expected (0,0), got %v", spot)
	}
}

func TestFindPath_StraightHorizontal(t *testing.T) {
	g := NewGrid()
	path, err := FindPath(g, Coord{0, 0}, Coord{3, 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) != 4 {
		t.Fatalf("expected path length 4, got %d: %v", len(path), path)
	}
	if path[0] != (Coord{0, 0}) || path[len(path)-1] != (Coord{3, 0}) {
		t.Errorf("path should start at (0,0) and end at (3,0), got %v to %v", path[0], path[len(path)-1])
	}
}

func TestFindPath_StraightVertical(t *testing.T) {
	g := NewGrid()
	path, err := FindPath(g, Coord{0, 0}, Coord{0, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) != 4 {
		t.Fatalf("expected path length 4, got %d: %v", len(path), path)
	}
	if path[0] != (Coord{0, 0}) || path[len(path)-1] != (Coord{0, 3}) {
		t.Errorf("path should start at (0,0) and end at (0,3), got %v to %v", path[0], path[len(path)-1])
	}
}

func TestFindPath_AroundObstacle(t *testing.T) {
	g := NewGrid()
	// Place an obstacle blocking direct horizontal path
	g.Reserve(Coord{1, 0}, 1, 1)

	path, err := FindPath(g, Coord{0, 0}, Coord{2, 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path[0] != (Coord{0, 0}) || path[len(path)-1] != (Coord{2, 0}) {
		t.Errorf("path should start at (0,0) and end at (2,0), got %v to %v", path[0], path[len(path)-1])
	}
	// The path should not go through the obstacle
	for _, c := range path {
		if c == (Coord{1, 0}) {
			t.Error("path should not go through obstacle at (1,0)")
		}
	}
}

func TestFindPath_NoPath(t *testing.T) {
	g := NewGrid()
	// Surround the start point so no path exists
	g.Reserve(Coord{1, 0}, 1, 1)
	g.Reserve(Coord{0, 1}, 1, 1)

	_, err := FindPath(g, Coord{0, 0}, Coord{5, 5})
	if err == nil {
		t.Error("expected error for blocked path, got nil")
	}
}

func TestFindPath_SamePoint(t *testing.T) {
	g := NewGrid()
	path, err := FindPath(g, Coord{2, 2}, Coord{2, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(path) != 1 {
		t.Fatalf("expected path length 1, got %d: %v", len(path), path)
	}
	if path[0] != (Coord{2, 2}) {
		t.Errorf("expected single point (2,2), got %v", path[0])
	}
}

func TestFindPath_DestinationOccupied(t *testing.T) {
	g := NewGrid()
	// Destination is occupied but FindPath should still reach it
	g.Reserve(Coord{2, 0}, 1, 1)

	path, err := FindPath(g, Coord{0, 0}, Coord{2, 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path[len(path)-1] != (Coord{2, 0}) {
		t.Errorf("path should end at occupied destination (2,0), got %v", path[len(path)-1])
	}
}
