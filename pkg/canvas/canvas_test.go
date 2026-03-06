package canvas

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	c := New(5, 3)
	if c == nil {
		t.Fatal("expected non-nil canvas")
	}
	if c.Width() != 5 {
		t.Errorf("Width() = %d, want 5", c.Width())
	}
	if c.Height() != 3 {
		t.Errorf("Height() = %d, want 3", c.Height())
	}
	// All cells should be spaces.
	for x := 0; x < 5; x++ {
		for y := 0; y < 3; y++ {
			if got := c.Get(x, y); got != " " {
				t.Errorf("Get(%d,%d) = %q, want %q", x, y, got, " ")
			}
		}
	}
}

func TestNew_InvalidDimensions(t *testing.T) {
	if c := New(0, 5); c != nil {
		t.Error("expected nil for zero width")
	}
	if c := New(5, 0); c != nil {
		t.Error("expected nil for zero height")
	}
	if c := New(-1, 5); c != nil {
		t.Error("expected nil for negative width")
	}
}

func TestSetAndGet(t *testing.T) {
	c := New(3, 3)
	c.Set(1, 2, "X")
	if got := c.Get(1, 2); got != "X" {
		t.Errorf("Get(1,2) = %q, want %q", got, "X")
	}
}

func TestSetOutOfBounds(t *testing.T) {
	c := New(3, 3)
	// Should not panic.
	c.Set(10, 10, "X")
	c.Set(-1, 0, "X")
}

func TestGetOutOfBounds(t *testing.T) {
	c := New(3, 3)
	if got := c.Get(10, 10); got != " " {
		t.Errorf("out-of-bounds Get = %q, want space", got)
	}
	if got := c.Get(-1, 0); got != " " {
		t.Errorf("negative index Get = %q, want space", got)
	}
}

func TestInBounds(t *testing.T) {
	c := New(4, 5)
	tests := []struct {
		x, y int
		want bool
	}{
		{0, 0, true},
		{3, 4, true},
		{4, 0, false},
		{0, 5, false},
		{-1, 0, false},
	}
	for _, tt := range tests {
		if got := c.InBounds(tt.x, tt.y); got != tt.want {
			t.Errorf("InBounds(%d,%d) = %v, want %v", tt.x, tt.y, got, tt.want)
		}
	}
}

func TestDrawText(t *testing.T) {
	c := New(10, 1)
	c.DrawText(2, 0, "Hello")
	got := c.ToString()
	if got != "  Hello   " {
		t.Errorf("ToString() = %q, want %q", got, "  Hello   ")
	}
}

func TestDrawText_Clipping(t *testing.T) {
	c := New(5, 1)
	// Text extends beyond canvas width; excess characters are clipped.
	c.DrawText(3, 0, "ABCDE")
	if c.Get(3, 0) != "A" {
		t.Errorf("expected A at (3,0)")
	}
	if c.Get(4, 0) != "B" {
		t.Errorf("expected B at (4,0)")
	}
	// (5,0) is out of bounds, so no crash and original space remains unreachable.
}

func TestToString(t *testing.T) {
	c := New(3, 2)
	c.Set(0, 0, "A")
	c.Set(1, 0, "B")
	c.Set(2, 0, "C")
	c.Set(0, 1, "D")
	c.Set(1, 1, "E")
	c.Set(2, 1, "F")
	got := c.ToString()
	want := "ABC\nDEF"
	if got != want {
		t.Errorf("ToString() = %q, want %q", got, want)
	}
}

func TestToString_NoTrailingNewline(t *testing.T) {
	c := New(2, 2)
	s := c.ToString()
	if strings.HasSuffix(s, "\n") {
		t.Error("ToString should not end with a newline")
	}
}

func TestCopy(t *testing.T) {
	c := New(3, 3)
	c.Set(1, 1, "X")
	cp := c.Copy()
	if cp.Get(1, 1) != "X" {
		t.Error("copy should preserve content")
	}
	// Mutating the copy should not affect the original.
	cp.Set(1, 1, "Y")
	if c.Get(1, 1) != "X" {
		t.Error("mutating copy should not affect original")
	}
}

func TestCopy_Nil(t *testing.T) {
	var c *Canvas
	if c.Copy() != nil {
		t.Error("copying nil canvas should return nil")
	}
}

func TestIncreaseSize(t *testing.T) {
	c := New(3, 3)
	c.Set(0, 0, "A")
	c.Set(2, 2, "B")
	c.IncreaseSize(5, 4)
	if c.Width() != 5 || c.Height() != 4 {
		t.Errorf("after IncreaseSize: got %dx%d, want 5x4", c.Width(), c.Height())
	}
	if c.Get(0, 0) != "A" {
		t.Error("content at (0,0) should be preserved")
	}
	if c.Get(2, 2) != "B" {
		t.Error("content at (2,2) should be preserved")
	}
	// New cells should be spaces.
	if c.Get(4, 3) != " " {
		t.Error("new cells should be spaces")
	}
}

func TestIncreaseSize_NoOp(t *testing.T) {
	c := New(5, 5)
	c.Set(0, 0, "Z")
	c.IncreaseSize(3, 3)
	if c.Width() != 5 || c.Height() != 5 {
		t.Error("IncreaseSize with smaller dimensions should be a no-op")
	}
	if c.Get(0, 0) != "Z" {
		t.Error("content should be preserved on no-op IncreaseSize")
	}
}

func TestNilCanvas(t *testing.T) {
	var c *Canvas
	if c.Width() != 0 {
		t.Error("nil Width should be 0")
	}
	if c.Height() != 0 {
		t.Error("nil Height should be 0")
	}
	if c.ToString() != "" {
		t.Error("nil ToString should be empty")
	}
}

func TestBoxChars(t *testing.T) {
	// Verify that the Unicode and ASCII box char sets have the expected values.
	if UnicodeBox.TopLeft != '┌' {
		t.Errorf("UnicodeBox.TopLeft = %c, want ┌", UnicodeBox.TopLeft)
	}
	if ASCIIBox.TopLeft != '+' {
		t.Errorf("ASCIIBox.TopLeft = %c, want +", ASCIIBox.TopLeft)
	}
	if UnicodeBox.Horizontal != '─' {
		t.Errorf("UnicodeBox.Horizontal = %c, want ─", UnicodeBox.Horizontal)
	}
	if ASCIIBox.Horizontal != '-' {
		t.Errorf("ASCIIBox.Horizontal = %c, want -", ASCIIBox.Horizontal)
	}
}
