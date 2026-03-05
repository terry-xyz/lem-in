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

// normalizeTunnel returns a canonical string for a tunnel between two rooms.
func normalizeTunnel(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}
