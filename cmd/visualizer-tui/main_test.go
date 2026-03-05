package main

import (
	"testing"

	"lem-in/internal/format"
)

func TestScaleCoords_BasicLayout(t *testing.T) {
	rooms := []format.ParsedRoom{
		{Name: "A", X: 0, Y: 0},
		{Name: "B", X: 10, Y: 10},
		{Name: "C", X: 5, Y: 5},
	}

	positions := scaleCoords(rooms, 80, 24)

	if positions == nil {
		t.Fatal("expected non-nil positions")
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(positions))
	}

	// A should be at top-left area (near margin)
	posA := positions["A"]
	posB := positions["B"]
	posC := positions["C"]

	// A should be left/above B
	if posA.screenX >= posB.screenX {
		t.Errorf("A.x (%d) should be < B.x (%d)", posA.screenX, posB.screenX)
	}
	if posA.screenY >= posB.screenY {
		t.Errorf("A.y (%d) should be < B.y (%d)", posA.screenY, posB.screenY)
	}

	// C should be between A and B
	if posC.screenX <= posA.screenX || posC.screenX >= posB.screenX {
		t.Errorf("C.x (%d) should be between A.x (%d) and B.x (%d)", posC.screenX, posA.screenX, posB.screenX)
	}
}

func TestScaleCoords_SingleRoom(t *testing.T) {
	rooms := []format.ParsedRoom{
		{Name: "only", X: 5, Y: 5},
	}

	positions := scaleCoords(rooms, 80, 24)

	if positions == nil {
		t.Fatal("expected non-nil positions")
	}
	// Single room should be placed within bounds
	pos := positions["only"]
	if pos.screenX < 3 || pos.screenX >= 77 {
		t.Errorf("screenX %d out of bounds", pos.screenX)
	}
	if pos.screenY < 3 || pos.screenY >= 21 {
		t.Errorf("screenY %d out of bounds", pos.screenY)
	}
}

func TestScaleCoords_EmptyRooms(t *testing.T) {
	positions := scaleCoords(nil, 80, 24)
	if positions != nil {
		t.Errorf("expected nil for empty rooms, got %v", positions)
	}
}

func TestDirGrid_Horizontal(t *testing.T) {
	dg := newDirGrid(20, 10)
	cv := newCanvas(20, 10)
	dg.tracePath(2, 5, 8, 5)
	dg.applyToCanvas(cv, fgDkGreen)
	for x := 2; x <= 8; x++ {
		c := cv.get(x, 5)
		if c.ch != '─' {
			t.Errorf("at (%d,5) expected '─', got %c", x, c.ch)
		}
	}
}

func TestDirGrid_Vertical(t *testing.T) {
	dg := newDirGrid(20, 10)
	cv := newCanvas(20, 10)
	dg.tracePath(5, 2, 5, 8)
	dg.applyToCanvas(cv, fgDkGreen)
	for y := 2; y <= 8; y++ {
		c := cv.get(5, y)
		if c.ch != '│' {
			t.Errorf("at (5,%d) expected '│', got %c", y, c.ch)
		}
	}
}

func TestDirGrid_LShape(t *testing.T) {
	dg := newDirGrid(20, 10)
	cv := newCanvas(20, 10)
	dg.tracePath(2, 2, 8, 6)
	dg.applyToCanvas(cv, fgDkGreen)
	// Horizontal segment at row 2 from x=2 to x=7
	for x := 2; x <= 7; x++ {
		c := cv.get(x, 2)
		if c.ch != '─' {
			t.Errorf("at (%d,2) expected '─', got %c", x, c.ch)
		}
	}
	// Vertical segment at col 8 from y=3 to y=6
	for y := 3; y <= 6; y++ {
		c := cv.get(8, y)
		if c.ch != '│' {
			t.Errorf("at (8,%d) expected '│', got %c", y, c.ch)
		}
	}
	// Corner at (8, 2): right-then-down = ┐
	c := cv.get(8, 2)
	if c.ch != '┐' {
		t.Errorf("at (8,2) expected '┐', got %c", c.ch)
	}
}

func TestDirGrid_TJunction(t *testing.T) {
	dg := newDirGrid(20, 10)
	cv := newCanvas(20, 10)
	// Horizontal line on row 5
	dg.tracePath(0, 5, 9, 5)
	// L-shape from (5,5) going up to (5,0) — corner merges with horizontal
	dg.tracePath(5, 5, 5, 0)
	dg.applyToCanvas(cv, fgDkGreen)
	// At (5,5): horizontal contributes left+right, vertical contributes up → ┴
	c := cv.get(5, 5)
	if c.ch != '┴' {
		t.Errorf("at (5,5) expected '┴' (T-junction), got %c", c.ch)
	}
}

func TestDirGrid_Crossing(t *testing.T) {
	dg := newDirGrid(20, 10)
	cv := newCanvas(20, 10)
	// Horizontal line on row 5
	dg.tracePath(0, 5, 9, 5)
	// Vertical line on col 5
	dg.tracePath(5, 0, 5, 9)
	dg.applyToCanvas(cv, fgDkGreen)
	// At (5,5): should be ┼
	c := cv.get(5, 5)
	if c.ch != '┼' {
		t.Errorf("at (5,5) expected '┼', got %c", c.ch)
	}
}

func TestPlayback_SpeedControls(t *testing.T) {
	p := newPlayback()

	// Default speed
	if p.speed != 800 {
		t.Errorf("default speed = %d, want 800", p.speed)
	}

	// Speed up
	p.faster()
	if p.speed != 700 {
		t.Errorf("after faster: speed = %d, want 700", p.speed)
	}

	// Speed up to minimum
	for i := 0; i < 20; i++ {
		p.faster()
	}
	if p.speed != 100 {
		t.Errorf("minimum speed = %d, want 100", p.speed)
	}

	// Slow down
	p.slower()
	if p.speed != 200 {
		t.Errorf("after slower: speed = %d, want 200", p.speed)
	}

	// Slow down to maximum
	for i := 0; i < 50; i++ {
		p.slower()
	}
	if p.speed != 3000 {
		t.Errorf("maximum speed = %d, want 3000", p.speed)
	}
}

func TestPlayback_DefaultMode(t *testing.T) {
	p := newPlayback()
	if p.mode != modeAutoPlay {
		t.Errorf("default mode = %d, want modeAutoPlay (%d)", p.mode, modeAutoPlay)
	}
	if p.paused {
		t.Error("should not be paused by default")
	}
	if p.turnIdx != -1 {
		t.Errorf("initial turnIdx = %d, want -1", p.turnIdx)
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		input, want int
	}{
		{5, 5},
		{-5, 5},
		{0, 0},
		{-1, 1},
	}
	for _, tc := range tests {
		got := abs(tc.input)
		if got != tc.want {
			t.Errorf("abs(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestAntState_TrackPositions(t *testing.T) {
	as := newAntState(3, "start")

	// All ants start at "start"
	ids := as.antsAtRoom("start")
	if len(ids) != 3 {
		t.Errorf("ants at start = %d, want 3", len(ids))
	}

	// Apply a turn moving ant 1 to room A
	as.applyTurn([]format.Movement{{AntID: 1, RoomName: "A"}})

	ids = as.antsAtRoom("start")
	if len(ids) != 2 {
		t.Errorf("ants at start after move = %d, want 2", len(ids))
	}
	ids = as.antsAtRoom("A")
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("ants at A = %v, want [1]", ids)
	}
}

func TestCanvas_SetAndGet(t *testing.T) {
	cv := newCanvas(10, 5)

	// Default cell should be space
	c := cv.get(0, 0)
	if c.ch != ' ' {
		t.Errorf("default cell = %c, want ' '", c.ch)
	}

	// Set and retrieve
	cv.set(3, 2, 'X', fgRed)
	c = cv.get(3, 2)
	if c.ch != 'X' {
		t.Errorf("cell = %c, want 'X'", c.ch)
	}
	if c.fg != fgRed {
		t.Errorf("cell fg = %q, want fgRed", c.fg)
	}

	// Out-of-bounds get returns space
	c = cv.get(-1, -1)
	if c.ch != ' ' {
		t.Errorf("out-of-bounds cell = %c, want ' '", c.ch)
	}
	c = cv.get(100, 100)
	if c.ch != ' ' {
		t.Errorf("out-of-bounds cell = %c, want ' '", c.ch)
	}
}
