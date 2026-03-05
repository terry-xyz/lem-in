package solver

import (
	"testing"

	"lem-in/internal/graph"
	"lem-in/internal/parser"
)

// helper to build a Colony without reading a file.
func makeColony(antCount int, rooms []parser.Room, startName, endName string, links [][2]string) *parser.Colony {
	rm := make(map[string]int, len(rooms))
	for i, r := range rooms {
		rm[r.Name] = i
	}
	return &parser.Colony{
		AntCount:  antCount,
		Rooms:     rooms,
		RoomMap:   rm,
		Links:     links,
		StartName: startName,
		EndName:   endName,
	}
}

// ---------- FindPaths tests ----------

// TestFindPathsLinear verifies a linear graph start-A-end finds exactly 1 path.
func TestFindPathsLinear(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"A", "end"}},
	)

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	p := paths[0]
	if p.Length() != 2 {
		t.Errorf("expected path length 2, got %d", p.Length())
	}

	// Path should be [start, A, end]
	expected := []string{"start", "A", "end"}
	if len(p.Rooms) != len(expected) {
		t.Fatalf("expected rooms %v, got %v", expected, p.Rooms)
	}
	for i, r := range expected {
		if p.Rooms[i] != r {
			t.Errorf("room[%d]: expected %q, got %q", i, r, p.Rooms[i])
		}
	}
}

// TestFindPathsTwoDisjoint verifies two vertex-disjoint paths are both found.
func TestFindPathsTwoDisjoint(t *testing.T) {
	// Graph:
	//   start -> A -> end
	//   start -> B -> end
	colony := makeColony(2,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 1},
			{Name: "B", X: 1, Y: 2},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{
			{"start", "A"}, {"A", "end"},
			{"start", "B"}, {"B", "end"},
		},
	)

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}

	// Both paths should have length 2
	for i, p := range paths {
		if p.Length() != 2 {
			t.Errorf("path %d: expected length 2, got %d", i, p.Length())
		}
	}
}

// TestFindPathsNoPath verifies that an error is returned when no path exists.
func TestFindPathsNoPath(t *testing.T) {
	// start and end exist but are not connected
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}}, // no link to end
	)

	g := graph.BuildGraph(colony)
	_, err := FindPaths(g, 1)
	if err == nil {
		t.Fatal("expected error for unreachable end, got nil")
	}
}

// TestFindPathsDirectLink verifies start directly linked to end gives path of length 1.
func TestFindPathsDirectLink(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "end", X: 1, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "end"}},
	)

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	p := paths[0]
	if p.Length() != 1 {
		t.Errorf("expected path length 1, got %d", p.Length())
	}

	expected := []string{"start", "end"}
	if len(p.Rooms) != len(expected) {
		t.Fatalf("expected rooms %v, got %v", expected, p.Rooms)
	}
	for i, r := range expected {
		if p.Rooms[i] != r {
			t.Errorf("room[%d]: expected %q, got %q", i, r, p.Rooms[i])
		}
	}
}

// ---------- DistributeAnts tests ----------

// TestDistributeSinglePath verifies all ants go to the single path.
func TestDistributeSinglePath(t *testing.T) {
	paths := []Path{
		{Rooms: []string{"start", "A", "end"}}, // length 2
	}

	counts, assignments := DistributeAnts(paths, 5)

	if len(counts) != 1 {
		t.Fatalf("expected 1 count entry, got %d", len(counts))
	}
	if counts[0] != 5 {
		t.Errorf("expected 5 ants on path 0, got %d", counts[0])
	}
	if len(assignments) != 5 {
		t.Errorf("expected 5 assignments, got %d", len(assignments))
	}

	// All assignments should reference path 0
	for _, a := range assignments {
		if a.PathIndex != 0 {
			t.Errorf("ant %d: expected PathIndex=0, got %d", a.AntID, a.PathIndex)
		}
	}
}

// TestDistributeTwoEqualPaths verifies even distribution across two equal-length paths.
func TestDistributeTwoEqualPaths(t *testing.T) {
	paths := []Path{
		{Rooms: []string{"start", "A", "end"}}, // length 2
		{Rooms: []string{"start", "B", "end"}}, // length 2
	}

	counts, assignments := DistributeAnts(paths, 10)

	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 10 {
		t.Errorf("expected total ants 10, got %d", total)
	}

	if len(assignments) != 10 {
		t.Errorf("expected 10 assignments, got %d", len(assignments))
	}

	// With two equal paths and even number of ants, each should get 5
	if counts[0] != 5 || counts[1] != 5 {
		t.Errorf("expected 5/5 distribution, got %d/%d", counts[0], counts[1])
	}
}

// TestDistributeMorePathsThanAnts verifies excess paths get 0 ants.
func TestDistributeMorePathsThanAnts(t *testing.T) {
	paths := []Path{
		{Rooms: []string{"start", "A", "end"}}, // length 2
		{Rooms: []string{"start", "B", "end"}}, // length 2
		{Rooms: []string{"start", "C", "end"}}, // length 2
	}

	counts, assignments := DistributeAnts(paths, 2)

	total := 0
	for _, c := range counts {
		total += c
	}
	if total != 2 {
		t.Errorf("expected total ants 2, got %d", total)
	}

	if len(assignments) != 2 {
		t.Errorf("expected 2 assignments, got %d", len(assignments))
	}

	// At least one path should have 0 ants
	hasZero := false
	for _, c := range counts {
		if c == 0 {
			hasZero = true
			break
		}
	}
	if !hasZero {
		t.Errorf("expected at least one path with 0 ants, got counts %v", counts)
	}
}

// ---------- Turn count verification with audit examples ----------

// turnCount computes the number of turns for the given paths and ant count.
// This matches the formula: T = pathLength[k-1] - 1 + ceil((ants - sumDiff) / k)
func turnCount(paths []Path, antCount int) int {
	_, assignments := DistributeAnts(paths, antCount)

	// Simulate to count actual turns
	type antState struct {
		pathIndex int
		position  int // index within path Rooms (0 = start)
	}

	// Group assignments by path
	antsPerPath := make([]int, len(paths))
	for _, a := range assignments {
		antsPerPath[a.PathIndex]++
	}

	// Simulate turns
	var ants []antState
	sent := make([]int, len(paths)) // how many ants have been sent on each path
	turns := 0

	for {
		turns++

		// Move existing ants forward
		for i := range ants {
			ants[i].position++
		}

		// Send new ants
		for p := 0; p < len(paths); p++ {
			if sent[p] < antsPerPath[p] {
				ants = append(ants, antState{pathIndex: p, position: 1})
				sent[p]++
			}
		}

		// Check if all ants have arrived at end
		allDone := true
		for _, a := range ants {
			if a.position < paths[a.pathIndex].Length() {
				allDone = false
				break
			}
		}

		totalSent := 0
		for _, s := range sent {
			totalSent += s
		}

		if allDone && totalSent == antCount {
			break
		}
	}

	return turns
}

// TestAuditExample00 verifies the turn count for example00 (4 ants, <= 6 turns).
func TestAuditExample00(t *testing.T) {
	colony, err := parser.Parse("../../examples/example00.txt")
	if err != nil {
		t.Fatalf("failed to parse example00: %v", err)
	}

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, colony.AntCount)
	if err != nil {
		t.Fatalf("failed to find paths: %v", err)
	}

	turns := turnCount(paths, colony.AntCount)
	if turns > 6 {
		t.Errorf("example00: expected <= 6 turns, got %d", turns)
	}
}

// TestAuditExample02 verifies the turn count for example02 (20 ants, <= 11 turns).
func TestAuditExample02(t *testing.T) {
	colony, err := parser.Parse("../../examples/example02.txt")
	if err != nil {
		t.Fatalf("failed to parse example02: %v", err)
	}

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, colony.AntCount)
	if err != nil {
		t.Fatalf("failed to find paths: %v", err)
	}

	turns := turnCount(paths, colony.AntCount)
	if turns > 11 {
		t.Errorf("example02: expected <= 11 turns, got %d", turns)
	}
}

// TestAuditExample04 verifies the turn count for example04 (9 ants, <= 6 turns).
func TestAuditExample04(t *testing.T) {
	colony, err := parser.Parse("../../examples/example04.txt")
	if err != nil {
		t.Fatalf("failed to parse example04: %v", err)
	}

	g := graph.BuildGraph(colony)
	paths, err := FindPaths(g, colony.AntCount)
	if err != nil {
		t.Fatalf("failed to find paths: %v", err)
	}

	turns := turnCount(paths, colony.AntCount)
	if turns > 6 {
		t.Errorf("example04: expected <= 6 turns, got %d", turns)
	}
}
