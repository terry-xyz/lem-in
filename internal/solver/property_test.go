package solver_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"lem-in/internal/graph"
	"lem-in/internal/parser"
	"lem-in/internal/simulator"
	"lem-in/internal/solver"
)

// TestPropertyAllAuditExamples runs property-based invariant checks on all valid audit examples.
func TestPropertyAllAuditExamples(t *testing.T) {
	examples := []struct {
		file     string
		maxTurns int
	}{
		{"../../examples/example00.txt", 6},
		{"../../examples/example01.txt", 8},
		{"../../examples/example02.txt", 11},
		{"../../examples/example03.txt", 6},
		{"../../examples/example04.txt", 6},
		{"../../examples/example05.txt", 8},
		{"../../examples/example06.txt", 0},
		{"../../examples/example07.txt", 0},
	}

	for _, ex := range examples {
		t.Run(ex.file, func(t *testing.T) {
			colony, err := parser.Parse(ex.file)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			g := graph.BuildGraph(colony)
			paths, err := solver.FindPaths(g, colony.AntCount)
			if err != nil {
				t.Fatalf("find paths error: %v", err)
			}

			// Property 1: All paths start at ##start and end at ##end
			for i, p := range paths {
				if p.Rooms[0] != colony.StartName {
					t.Errorf("path %d starts at %q, want %q", i, p.Rooms[0], colony.StartName)
				}
				if p.Rooms[len(p.Rooms)-1] != colony.EndName {
					t.Errorf("path %d ends at %q, want %q", i, p.Rooms[len(p.Rooms)-1], colony.EndName)
				}
			}

			// Property 2: Paths are vertex-disjoint (no shared intermediate rooms)
			intermediateRooms := make(map[string]int)
			for i, p := range paths {
				for _, room := range p.Rooms[1 : len(p.Rooms)-1] {
					if prev, exists := intermediateRooms[room]; exists {
						t.Errorf("room %q shared by paths %d and %d (not vertex-disjoint)", room, prev, i)
					}
					intermediateRooms[room] = i
				}
			}

			// Distribute and simulate
			antsPerPath, assignments := solver.DistributeAnts(paths, colony.AntCount)

			// Property 3: Every ant (1..N) assigned exactly once
			antSeen := make(map[int]bool)
			for _, a := range assignments {
				if antSeen[a.AntID] {
					t.Errorf("ant %d assigned more than once", a.AntID)
				}
				antSeen[a.AntID] = true
			}
			for i := 1; i <= colony.AntCount; i++ {
				if !antSeen[i] {
					t.Errorf("ant %d not assigned", i)
				}
			}

			// Property 4: No path receives negative ants
			for i, c := range antsPerPath {
				if c < 0 {
					t.Errorf("path %d has negative ant count: %d", i, c)
				}
			}

			// Simulate
			lines := simulator.Simulate(paths, antsPerPath, assignments)

			// Property 5: Turn count within audit limits (if specified)
			if ex.maxTurns > 0 && len(lines) > ex.maxTurns {
				t.Errorf("got %d turns, max allowed %d", len(lines), ex.maxTurns)
			}

			// Property 6-10: Validate simulation output invariants
			validateSimulation(t, lines, colony.AntCount, colony.StartName, colony.EndName, paths)

			t.Logf("OK: %d ants, %d paths, %d turns", colony.AntCount, len(paths), len(lines))
		})
	}
}

// validateSimulation checks all simulation invariants.
func validateSimulation(t *testing.T, lines []string, antCount int, startName, endName string, paths []solver.Path) {
	t.Helper()

	moveRe := regexp.MustCompile(`^L(\d+)-(\S+)$`)
	arrivedAtEnd := make(map[int]bool)
	antPositions := make(map[int]string) // track current position for tunnel detection
	allIntermediate := make(map[string]bool)
	for _, p := range paths {
		for _, r := range p.Rooms[1 : len(p.Rooms)-1] {
			allIntermediate[r] = true
		}
	}

	// Initialize all ants at start
	for i := 1; i <= antCount; i++ {
		antPositions[i] = startName
	}

	for turnIdx, line := range lines {
		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			t.Errorf("turn %d: empty line", turnIdx+1)
			continue
		}

		roomOccupants := make(map[string]int)
		tunnelsUsed := make(map[string]int)
		lastAntID := 0

		for _, tok := range tokens {
			matches := moveRe.FindStringSubmatch(tok)
			if matches == nil {
				t.Errorf("turn %d: invalid move format %q", turnIdx+1, tok)
				continue
			}

			antID, _ := strconv.Atoi(matches[1])
			room := matches[2]

			// Ant IDs ascending within turn
			if antID <= lastAntID {
				t.Errorf("turn %d: ant ID %d not ascending (prev %d)", turnIdx+1, antID, lastAntID)
			}
			lastAntID = antID

			// Ant IDs in valid range
			if antID < 1 || antID > antCount {
				t.Errorf("turn %d: ant ID %d out of range [1, %d]", turnIdx+1, antID, antCount)
			}

			// Track tunnel usage
			prevRoom := antPositions[antID]
			tunnel := normalizeTunnel(prevRoom, room)
			tunnelsUsed[tunnel]++
			antPositions[antID] = room

			if room == endName {
				arrivedAtEnd[antID] = true
			}

			if allIntermediate[room] {
				roomOccupants[room]++
			}
		}

		// No intermediate room has > 1 ant
		for room, count := range roomOccupants {
			if count > 1 {
				t.Errorf("turn %d: intermediate room %q has %d ants", turnIdx+1, room, count)
			}
		}

		// No tunnel used more than once per turn
		for tunnel, count := range tunnelsUsed {
			if count > 1 {
				t.Errorf("turn %d: tunnel %q used %d times", turnIdx+1, tunnel, count)
			}
		}
	}

	// All ants reach ##end
	for i := 1; i <= antCount; i++ {
		if !arrivedAtEnd[i] {
			t.Errorf("ant %d never reached %s", i, endName)
		}
	}
}

// normalizeTunnel returns a canonical string for a tunnel between two rooms.
func normalizeTunnel(a, b string) string {
	if a < b {
		return a + "-" + b
	}
	return b + "-" + a
}

// TestPropertyRandomGraphs tests solver invariants on programmatically generated graphs.
func TestPropertyRandomGraphs(t *testing.T) {
	tests := []struct {
		name     string
		antCount int
		rooms    []parser.Room
		start    string
		end      string
		links    [][2]string
	}{
		{
			name:     "single path 3 rooms",
			antCount: 5,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "m", X: 1, Y: 0}, {Name: "e", X: 2, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "m"}, {"m", "e"}},
		},
		{
			name:     "two parallel paths",
			antCount: 10,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "a", X: 1, Y: 1}, {Name: "b", X: 1, Y: 2}, {Name: "e", X: 2, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "a"}, {"a", "e"}, {"s", "b"}, {"b", "e"}},
		},
		{
			name:     "three parallel paths different lengths",
			antCount: 15,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0},
				{Name: "a1", X: 1, Y: 0}, {Name: "a2", X: 2, Y: 0},
				{Name: "b1", X: 1, Y: 1},
				{Name: "c1", X: 1, Y: 2}, {Name: "c2", X: 2, Y: 2}, {Name: "c3", X: 3, Y: 2},
				{Name: "e", X: 4, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{
				{"s", "a1"}, {"a1", "a2"}, {"a2", "e"},
				{"s", "b1"}, {"b1", "e"},
				{"s", "c1"}, {"c1", "c2"}, {"c2", "c3"}, {"c3", "e"},
			},
		},
		{
			name:     "direct link",
			antCount: 3,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "e", X: 1, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "e"}},
		},
		{
			name:     "diamond graph",
			antCount: 20,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "a", X: 1, Y: 1}, {Name: "b", X: 1, Y: 2},
				{Name: "c", X: 2, Y: 1}, {Name: "d", X: 2, Y: 2}, {Name: "e", X: 3, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{
				{"s", "a"}, {"s", "b"}, {"a", "c"}, {"b", "d"}, {"c", "e"}, {"d", "e"},
			},
		},
		{
			name:     "single ant",
			antCount: 1,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "a", X: 1, Y: 0}, {Name: "b", X: 1, Y: 1}, {Name: "e", X: 2, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "a"}, {"a", "e"}, {"s", "b"}, {"b", "e"}},
		},
		{
			name:     "large ant count",
			antCount: 500,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "a", X: 1, Y: 0}, {Name: "b", X: 1, Y: 1}, {Name: "e", X: 2, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "a"}, {"a", "e"}, {"s", "b"}, {"b", "e"}},
		},
		{
			name:     "graph with cycle",
			antCount: 5,
			rooms: []parser.Room{
				{Name: "s", X: 0, Y: 0}, {Name: "a", X: 1, Y: 0}, {Name: "b", X: 2, Y: 0},
				{Name: "c", X: 1, Y: 1}, {Name: "e", X: 3, Y: 0},
			},
			start: "s", end: "e",
			links: [][2]string{{"s", "a"}, {"a", "b"}, {"b", "e"}, {"a", "c"}, {"c", "b"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			colony := &parser.Colony{
				AntCount:  tc.antCount,
				Rooms:     tc.rooms,
				RoomMap:   make(map[string]int),
				Links:     tc.links,
				StartName: tc.start,
				EndName:   tc.end,
			}
			for i, r := range tc.rooms {
				colony.RoomMap[r.Name] = i
			}

			g := graph.BuildGraph(colony)
			paths, err := solver.FindPaths(g, tc.antCount)
			if err != nil {
				t.Fatalf("FindPaths error: %v", err)
			}

			// Verify vertex-disjoint
			seen := make(map[string]bool)
			for _, p := range paths {
				if p.Rooms[0] != tc.start {
					t.Errorf("path starts at %q, want %q", p.Rooms[0], tc.start)
				}
				if p.Rooms[len(p.Rooms)-1] != tc.end {
					t.Errorf("path ends at %q, want %q", p.Rooms[len(p.Rooms)-1], tc.end)
				}
				for _, r := range p.Rooms[1 : len(p.Rooms)-1] {
					if seen[r] {
						t.Errorf("intermediate room %q used by multiple paths", r)
					}
					seen[r] = true
				}
			}

			// Distribute and simulate
			antsPerPath, assignments := solver.DistributeAnts(paths, tc.antCount)
			lines := simulator.Simulate(paths, antsPerPath, assignments)

			// Validate all simulation invariants
			validateSimulation(t, lines, tc.antCount, tc.start, tc.end, paths)

			t.Logf("OK: %d ants, %d paths, %d turns", tc.antCount, len(paths), len(lines))
		})
	}
}

// TestPropertyDeterminism runs the same example 5 times and verifies consistent results.
func TestPropertyDeterminism(t *testing.T) {
	examples := []string{
		"../../examples/example00.txt",
		"../../examples/example01.txt",
		"../../examples/example04.txt",
	}

	for _, file := range examples {
		t.Run(file, func(t *testing.T) {
			var firstOutput string

			for run := 0; run < 5; run++ {
				colony, err := parser.Parse(file)
				if err != nil {
					t.Fatalf("run %d: parse error: %v", run, err)
				}

				g := graph.BuildGraph(colony)
				paths, err := solver.FindPaths(g, colony.AntCount)
				if err != nil {
					t.Fatalf("run %d: find paths error: %v", run, err)
				}

				antsPerPath, assignments := solver.DistributeAnts(paths, colony.AntCount)
				lines := simulator.Simulate(paths, antsPerPath, assignments)
				output := fmt.Sprintf("%v", lines)

				if run == 0 {
					firstOutput = output
				} else if output != firstOutput {
					t.Errorf("run %d produced different output than run 0", run)
				}
			}
		})
	}
}
