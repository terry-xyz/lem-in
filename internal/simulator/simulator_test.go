package simulator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"lem-in/internal/solver"
)

func TestSimulate_SinglePath(t *testing.T) {
	paths := []solver.Path{{Rooms: []string{"start", "A", "end"}}}
	antsPerPath := []int{3}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 0},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	if len(lines) != 4 {
		t.Fatalf("expected 4 turns, got %d: %v", len(lines), lines)
	}

	// Verify all ants reach end
	validateOutput(t, lines, 3, "end", paths)
}

func TestSimulate_TwoPaths(t *testing.T) {
	paths := []solver.Path{
		{Rooms: []string{"start", "end"}},      // length 1
		{Rooms: []string{"start", "A", "end"}}, // length 2
	}
	antsPerPath := []int{2, 1}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 1},
	}

	lines := Simulate(paths, antsPerPath, assignments)
	validateOutput(t, lines, 3, "end", paths)
}

func TestSimulate_OutputFormat(t *testing.T) {
	paths := []solver.Path{{Rooms: []string{"start", "mid", "end"}}}
	antsPerPath := []int{2}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	moveRe := regexp.MustCompile(`^L\d+-\S+$`)
	for _, line := range lines {
		tokens := strings.Fields(line)
		for _, tok := range tokens {
			if !moveRe.MatchString(tok) {
				t.Errorf("invalid move format: %q", tok)
			}
		}
	}
}

func TestSimulate_AntIDOrdering(t *testing.T) {
	paths := []solver.Path{
		{Rooms: []string{"start", "A", "end"}},
		{Rooms: []string{"start", "B", "end"}},
	}
	antsPerPath := []int{2, 2}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 1},
		{AntID: 4, PathIndex: 1},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	for _, line := range lines {
		tokens := strings.Fields(line)
		lastID := 0
		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			id, _ := strconv.Atoi(tok[1 : 1+dashIdx])
			if id <= lastID {
				t.Errorf("ant IDs not ascending in line %q", line)
			}
			lastID = id
		}
	}
}

// validateOutput checks all ants reach end and no intermediate room conflicts.
func validateOutput(t *testing.T, lines []string, totalAnts int, endRoom string, paths []solver.Path) {
	t.Helper()

	// Track ant positions per turn for tunnel conflict detection
	antPositions := make(map[int]string) // antID -> current room
	arrivedAtEnd := make(map[int]bool)

	// Build set of intermediate rooms
	intermediateRooms := make(map[string]bool)
	for _, p := range paths {
		for _, r := range p.Rooms[1 : len(p.Rooms)-1] {
			intermediateRooms[r] = true
		}
	}

	// Initialize all ant positions at start room
	startRoom := paths[0].Rooms[0]
	for i := 1; i <= totalAnts; i++ {
		antPositions[i] = startRoom
	}

	for turnIdx, line := range lines {
		tokens := strings.Fields(line)
		roomOccupants := make(map[string]int) // intermediate room -> count this turn
		tunnelsUsed := make(map[string]int)   // normalized tunnel -> count this turn

		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			if dashIdx < 0 {
				t.Fatalf("invalid token: %s", tok)
			}
			antID, err := strconv.Atoi(tok[1 : 1+dashIdx])
			if err != nil {
				t.Fatalf("invalid ant ID in %s: %v", tok, err)
			}
			room := tok[2+dashIdx:]

			// Track tunnel usage (from previous position to new position)
			prevRoom := antPositions[antID]
			tunnel := normalizeTunnel(prevRoom, room)
			tunnelsUsed[tunnel]++

			antPositions[antID] = room

			if room == endRoom {
				arrivedAtEnd[antID] = true
			}

			if intermediateRooms[room] {
				roomOccupants[room]++
			}
		}

		// Check no intermediate room has more than 1 ant
		for room, count := range roomOccupants {
			if count > 1 {
				t.Errorf("turn %d: room %s has %d ants", turnIdx+1, room, count)
			}
		}

		// Check no tunnel used more than once per turn
		for tunnel, count := range tunnelsUsed {
			if count > 1 {
				t.Errorf("turn %d: tunnel %s used %d times", turnIdx+1, tunnel, count)
			}
		}
	}

	// Check all ants arrived
	for i := 1; i <= totalAnts; i++ {
		if !arrivedAtEnd[i] {
			t.Errorf("ant %d never reached %s", i, endRoom)
		}
	}

	// Verify no duplicate output line (e.g., the output is cleanly formatted)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			t.Error("empty turn line found")
		}
	}

	fmt.Printf("  Validated: %d ants, %d turns\n", totalAnts, len(lines))
}

// TestSimulate_NoTrailingEmptyLines verifies output has no trailing blank lines.
func TestSimulate_NoTrailingEmptyLines(t *testing.T) {
	paths := []solver.Path{{Rooms: []string{"start", "A", "end"}}}
	antsPerPath := []int{3}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 0},
	}

	lines := Simulate(paths, antsPerPath, assignments)
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			t.Errorf("turn %d is blank", i+1)
		}
	}
}

// TestSimulate_AntFollowsPathSequence verifies each ant visits rooms in path order.
func TestSimulate_AntFollowsPathSequence(t *testing.T) {
	paths := []solver.Path{
		{Rooms: []string{"start", "A", "B", "end"}},
		{Rooms: []string{"start", "C", "end"}},
	}
	antsPerPath := []int{2, 1}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 1},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	// Track per-ant room visit sequence
	antPath := make(map[int][]string)
	for _, line := range lines {
		tokens := strings.Fields(line)
		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			antID, _ := strconv.Atoi(tok[1 : 1+dashIdx])
			room := tok[2+dashIdx:]
			antPath[antID] = append(antPath[antID], room)
		}
	}

	// Ants 1,2 should follow path 0: A, B, end
	for _, id := range []int{1, 2} {
		expected := []string{"A", "B", "end"}
		got := antPath[id]
		if len(got) != len(expected) {
			t.Errorf("ant %d: visited %v, want %v", id, got, expected)
			continue
		}
		for j, r := range expected {
			if got[j] != r {
				t.Errorf("ant %d step %d: got %s, want %s", id, j, got[j], r)
			}
		}
	}

	// Ant 3 should follow path 1: C, end
	expected := []string{"C", "end"}
	got := antPath[3]
	if len(got) != len(expected) {
		t.Errorf("ant 3: visited %v, want %v", got, expected)
	} else {
		for j, r := range expected {
			if got[j] != r {
				t.Errorf("ant 3 step %d: got %s, want %s", j, got[j], r)
			}
		}
	}
}

// TestSimulate_AntStopsAfterEnd verifies that ants that reached end don't appear again.
func TestSimulate_AntStopsAfterEnd(t *testing.T) {
	paths := []solver.Path{{Rooms: []string{"start", "A", "end"}}}
	antsPerPath := []int{3}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 0},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	arrived := make(map[int]int) // antID -> turn they arrived at end
	for turnIdx, line := range lines {
		tokens := strings.Fields(line)
		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			antID, _ := strconv.Atoi(tok[1 : 1+dashIdx])
			room := tok[2+dashIdx:]

			if prevTurn, ok := arrived[antID]; ok {
				t.Errorf("ant %d moved on turn %d after arriving at end on turn %d", antID, turnIdx+1, prevTurn)
			}
			if room == "end" {
				arrived[antID] = turnIdx + 1
			}
		}
	}
}

// TestSimulate_LowerIDArrivesFirst verifies lower-numbered ants on same path arrive before higher.
func TestSimulate_LowerIDArrivesFirst(t *testing.T) {
	paths := []solver.Path{
		{Rooms: []string{"start", "A", "B", "end"}},
		{Rooms: []string{"start", "C", "end"}},
	}
	antsPerPath := []int{3, 2}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 0},
		{AntID: 4, PathIndex: 1},
		{AntID: 5, PathIndex: 1},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	// Record arrival turns per ant
	arrivalTurn := make(map[int]int)
	for turnIdx, line := range lines {
		tokens := strings.Fields(line)
		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			antID, _ := strconv.Atoi(tok[1 : 1+dashIdx])
			room := tok[2+dashIdx:]
			if room == "end" {
				arrivalTurn[antID] = turnIdx + 1
			}
		}
	}

	// On path 0: ant 1 should arrive before ant 2, ant 2 before ant 3
	if arrivalTurn[1] >= arrivalTurn[2] {
		t.Errorf("path 0: ant 1 arrived turn %d, ant 2 arrived turn %d (want 1 < 2)", arrivalTurn[1], arrivalTurn[2])
	}
	if arrivalTurn[2] >= arrivalTurn[3] {
		t.Errorf("path 0: ant 2 arrived turn %d, ant 3 arrived turn %d (want 2 < 3)", arrivalTurn[2], arrivalTurn[3])
	}

	// On path 1: ant 4 should arrive before ant 5
	if arrivalTurn[4] >= arrivalTurn[5] {
		t.Errorf("path 1: ant 4 arrived turn %d, ant 5 arrived turn %d (want 4 < 5)", arrivalTurn[4], arrivalTurn[5])
	}
}

// TestSimulate_SingleSpaceSeparator verifies moves are separated by exactly one space.
func TestSimulate_SingleSpaceSeparator(t *testing.T) {
	paths := []solver.Path{
		{Rooms: []string{"start", "A", "end"}},
		{Rooms: []string{"start", "B", "end"}},
	}
	antsPerPath := []int{2, 2}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
		{AntID: 3, PathIndex: 1},
		{AntID: 4, PathIndex: 1},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	for _, line := range lines {
		if strings.Contains(line, "  ") {
			t.Errorf("double space found in line: %q", line)
		}
		if strings.HasPrefix(line, " ") || strings.HasSuffix(line, " ") {
			t.Errorf("leading/trailing space in line: %q", line)
		}
	}
}

// TestSimulate_OnlyMovingAntsAppear verifies stationary ants don't appear in output.
func TestSimulate_OnlyMovingAntsAppear(t *testing.T) {
	paths := []solver.Path{{Rooms: []string{"start", "A", "end"}}}
	antsPerPath := []int{2}
	assignments := []solver.AntAssignment{
		{AntID: 1, PathIndex: 0},
		{AntID: 2, PathIndex: 0},
	}

	lines := Simulate(paths, antsPerPath, assignments)

	// Each ant should appear in exactly 2 lines (moving to A, then to end)
	antAppearances := make(map[int]int)
	for _, line := range lines {
		tokens := strings.Fields(line)
		for _, tok := range tokens {
			dashIdx := strings.Index(tok[1:], "-")
			antID, _ := strconv.Atoi(tok[1 : 1+dashIdx])
			antAppearances[antID]++
		}
	}

	for antID, count := range antAppearances {
		// Path has 2 steps (start->A->end), so each ant should appear exactly 2 times
		if count != 2 {
			t.Errorf("ant %d appeared %d times, want 2", antID, count)
		}
	}
}

// TestSimulate_TurnCountMatchesFormula verifies turn count matches T = Lk - 1 + ceil((N - sumDiff) / k).
func TestSimulate_TurnCountMatchesFormula(t *testing.T) {
	tests := []struct {
		name          string
		paths         []solver.Path
		antsPerPath   []int
		assignments   []solver.AntAssignment
		expectedTurns int
	}{
		{
			name:        "single path length 2",
			paths:       []solver.Path{{Rooms: []string{"start", "A", "end"}}},
			antsPerPath: []int{3},
			assignments: []solver.AntAssignment{
				{AntID: 1, PathIndex: 0},
				{AntID: 2, PathIndex: 0},
				{AntID: 3, PathIndex: 0},
			},
			expectedTurns: 4, // L=2, T = 2 - 1 + ceil(3/1) = 1 + 3 = 4
		},
		{
			name: "two equal paths length 1",
			paths: []solver.Path{
				{Rooms: []string{"start", "end"}},
				{Rooms: []string{"start", "end"}},
			},
			antsPerPath: []int{2, 2},
			assignments: []solver.AntAssignment{
				{AntID: 1, PathIndex: 0},
				{AntID: 2, PathIndex: 0},
				{AntID: 3, PathIndex: 1},
				{AntID: 4, PathIndex: 1},
			},
			expectedTurns: 2, // L=1, T = 1 - 1 + ceil(4/2) = 0 + 2 = 2
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines := Simulate(tc.paths, tc.antsPerPath, tc.assignments)
			if len(lines) != tc.expectedTurns {
				t.Errorf("got %d turns, want %d", len(lines), tc.expectedTurns)
			}
		})
	}
}

// TestSimulate_EmptyAssignments verifies nil return for empty input.
func TestSimulate_EmptyAssignments(t *testing.T) {
	lines := Simulate(nil, nil, nil)
	if lines != nil {
		t.Errorf("expected nil, got %v", lines)
	}
}

// normalizeTunnel returns a canonical string for a tunnel between two rooms.
func normalizeTunnel(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}
