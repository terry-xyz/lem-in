package graph

import (
	"testing"

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

// TestSimple3Room verifies a linear start-A-end graph.
// A is split into A_in/A_out so node count = 2 + 2 = 4.
func TestSimple3Room(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"A", "end"}},
	)

	g := BuildGraph(colony)

	if g.NodeCount != 4 {
		t.Errorf("expected NodeCount=4, got %d", g.NodeCount)
	}

	// start and end should each have a single ID
	if _, ok := g.NameToID["start"]; !ok {
		t.Error("start node missing from NameToID")
	}
	if _, ok := g.NameToID["end"]; !ok {
		t.Error("end node missing from NameToID")
	}

	// A should be split
	if _, ok := g.NameToID["A_in"]; !ok {
		t.Error("A_in missing from NameToID")
	}
	if _, ok := g.NameToID["A_out"]; !ok {
		t.Error("A_out missing from NameToID")
	}
}

// TestExample00NodeCount builds a graph like example00 (4 rooms: start(0), 2, 3, end(1)).
// Intermediate rooms: 2 and 3 -> each split into 2 nodes = 4 nodes.
// Start + End = 2 nodes. Total = 6.
func TestExample00NodeCount(t *testing.T) {
	colony := makeColony(4,
		[]parser.Room{
			{Name: "0", X: 0, Y: 3},
			{Name: "2", X: 2, Y: 5},
			{Name: "3", X: 4, Y: 0},
			{Name: "1", X: 8, Y: 3},
		},
		"0", "1",
		[][2]string{{"0", "2"}, {"2", "3"}, {"3", "1"}},
	)

	g := BuildGraph(colony)

	if g.NodeCount != 6 {
		t.Errorf("expected NodeCount=6, got %d", g.NodeCount)
	}

	// Verify start and end IDs
	if g.IDToName[g.StartID] != "0" {
		t.Errorf("expected start name '0', got '%s'", g.IDToName[g.StartID])
	}
	if g.IDToName[g.EndID] != "1" {
		t.Errorf("expected end name '1', got '%s'", g.IDToName[g.EndID])
	}
}

// TestStartEndNotSplit verifies that start and end rooms are NOT split.
func TestStartEndNotSplit(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "end", X: 1, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "end"}},
	)

	g := BuildGraph(colony)

	// Start/end should not have _in/_out variants
	if _, ok := g.NameToID["start_in"]; ok {
		t.Error("start should not be split, but start_in exists")
	}
	if _, ok := g.NameToID["start_out"]; ok {
		t.Error("start should not be split, but start_out exists")
	}
	if _, ok := g.NameToID["end_in"]; ok {
		t.Error("end should not be split, but end_in exists")
	}
	if _, ok := g.NameToID["end_out"]; ok {
		t.Error("end should not be split, but end_out exists")
	}

	// Should be exactly 2 nodes
	if g.NodeCount != 2 {
		t.Errorf("expected NodeCount=2, got %d", g.NodeCount)
	}
}

// TestInternalEdgeCapacity verifies that split node internal edges have capacity 1.
func TestInternalEdgeCapacity(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"A", "end"}},
	)

	g := BuildGraph(colony)

	aIn := g.NameToID["A_in"]
	aOut := g.NameToID["A_out"]

	// Find the forward edge from A_in to A_out
	found := false
	for _, e := range g.Adj[aIn] {
		if e.To == aOut && e.Cap == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected internal edge A_in -> A_out with cap=1, not found")
	}
}

// TestTunnelEdgesBidirectional verifies tunnel edges create both a_out->b_in and b_out->a_in.
func TestTunnelEdgesBidirectional(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "B", X: 2, Y: 0},
			{Name: "end", X: 3, Y: 0},
		},
		"start", "end",
		[][2]string{{"A", "B"}, {"start", "A"}, {"B", "end"}},
	)

	g := BuildGraph(colony)

	aOut := g.NameToID["A_out"]
	bIn := g.NameToID["B_in"]
	bOut := g.NameToID["B_out"]
	aIn := g.NameToID["A_in"]

	// Check A_out -> B_in
	foundAtoB := false
	for _, e := range g.Adj[aOut] {
		if e.To == bIn && e.Cap == 1 {
			foundAtoB = true
			break
		}
	}
	if !foundAtoB {
		t.Error("expected tunnel edge A_out -> B_in with cap=1, not found")
	}

	// Check B_out -> A_in
	foundBtoA := false
	for _, e := range g.Adj[bOut] {
		if e.To == aIn && e.Cap == 1 {
			foundBtoA = true
			break
		}
	}
	if !foundBtoA {
		t.Error("expected tunnel edge B_out -> A_in with cap=1, not found")
	}
}

// TestOriginalName checks that _in and _out suffixes are stripped correctly.
func TestOriginalName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"room_in", "room"},
		{"room_out", "room"},
		{"start", "start"},
		{"end", "end"},
		{"A_in", "A"},
		{"A_out", "A"},
		{"my_room_in", "my_room"},
		{"my_room_out", "my_room"},
		{"in", "in"},   // too short to strip
		{"out", "out"}, // too short to strip
	}

	for _, tt := range tests {
		result := OriginalName(tt.input)
		if result != tt.expected {
			t.Errorf("OriginalName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestResidualEdges verifies that every forward edge has a corresponding reverse edge with Cap=0.
func TestResidualEdges(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"A", "end"}},
	)

	g := BuildGraph(colony)

	for from, edges := range g.Adj {
		for i, e := range edges {
			// Check that RevIdx is valid
			if e.RevIdx < 0 || e.RevIdx >= len(g.Adj[e.To]) {
				t.Fatalf("edge %d->%d has invalid RevIdx=%d (adj[%d] has %d edges)",
					from, e.To, e.RevIdx, e.To, len(g.Adj[e.To]))
			}
			// Check mutual consistency
			rev := g.Adj[e.To][e.RevIdx]
			if rev.To != from {
				t.Errorf("edge %d->%d RevIdx=%d: reverse edge points to %d, want %d",
					from, e.To, e.RevIdx, rev.To, from)
			}
			if rev.RevIdx != i {
				t.Errorf("edge %d->%d RevIdx=%d: reverse edge's RevIdx=%d, want %d",
					from, e.To, e.RevIdx, rev.RevIdx, i)
			}
			// Forward edge Cap>0 implies reverse Cap==0 (and vice versa)
			if e.Cap > 0 && rev.Cap != 0 {
				t.Errorf("forward edge %d->%d has cap=%d but reverse has cap=%d (want 0)",
					from, e.To, e.Cap, rev.Cap)
			}
		}
	}
}

// TestDirectStartEndEdge verifies the edge exists when start and end are directly linked.
func TestDirectStartEndEdge(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "end", X: 1, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "end"}},
	)

	g := BuildGraph(colony)

	found := false
	for _, e := range g.Adj[g.StartID] {
		if e.To == g.EndID && e.Cap == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected direct edge start -> end with cap=1, not found")
	}
}

// TestFanOutFromStart verifies multiple outgoing tunnels from start.
func TestFanOutFromStart(t *testing.T) {
	colony := makeColony(3,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "B", X: 1, Y: 1},
			{Name: "C", X: 1, Y: 2},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"start", "B"}, {"start", "C"}, {"A", "end"}, {"B", "end"}, {"C", "end"}},
	)

	g := BuildGraph(colony)

	// Start should have edges to A_in, B_in, C_in (3 forward + 3 residual from reverse)
	forwardCount := 0
	for _, e := range g.Adj[g.StartID] {
		if e.Cap == 1 {
			forwardCount++
		}
	}
	if forwardCount != 3 {
		t.Errorf("start has %d forward edges, want 3", forwardCount)
	}
}

// TestDiamondTopology verifies two independent paths through a diamond graph.
func TestDiamondTopology(t *testing.T) {
	colony := makeColony(2,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 1},
			{Name: "B", X: 1, Y: 2},
			{Name: "end", X: 2, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"start", "B"}, {"A", "end"}, {"B", "end"}},
	)

	g := BuildGraph(colony)

	// 2 intermediate rooms * 2 (split) + 2 (start/end) = 6 nodes
	if g.NodeCount != 6 {
		t.Errorf("expected NodeCount=6, got %d", g.NodeCount)
	}

	// A and B should be independently split
	if g.NameToID["A_in"] == g.NameToID["B_in"] {
		t.Error("A_in and B_in should have different IDs")
	}
}

// TestIDToNameConsistency verifies IDToName and NameToID are consistent.
func TestIDToNameConsistency(t *testing.T) {
	colony := makeColony(1,
		[]parser.Room{
			{Name: "start", X: 0, Y: 0},
			{Name: "A", X: 1, Y: 0},
			{Name: "B", X: 2, Y: 0},
			{Name: "end", X: 3, Y: 0},
		},
		"start", "end",
		[][2]string{{"start", "A"}, {"A", "B"}, {"B", "end"}},
	)

	g := BuildGraph(colony)

	if len(g.IDToName) != g.NodeCount {
		t.Errorf("IDToName length %d != NodeCount %d", len(g.IDToName), g.NodeCount)
	}

	for name, id := range g.NameToID {
		if id < 0 || id >= g.NodeCount {
			t.Errorf("NameToID[%q] = %d is out of range [0, %d)", name, id, g.NodeCount)
		}
		if g.IDToName[id] != name {
			t.Errorf("IDToName[%d] = %q, but NameToID[%q] = %d", id, g.IDToName[id], name, id)
		}
	}
}
