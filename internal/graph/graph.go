package graph

import "lem-in/internal/parser"

// Edge represents a directed edge in the flow network.
type Edge struct {
	To     int // target node ID
	Cap    int // capacity
	Flow   int // current flow
	RevIdx int // index of reverse edge in adj[To]
}

// Graph is an indexed adjacency list for the flow network.
type Graph struct {
	Adj       [][]Edge // adjacency list indexed by node ID
	NodeCount int
	NameToID  map[string]int
	IDToName  []string
	StartID   int
	EndID     int
}

// BuildGraph creates a flow network from a parsed Colony using node-splitting.
// Intermediate rooms are split into room_in and room_out with a capacity-1 edge.
// Start and end rooms are NOT split (unlimited capacity).
func BuildGraph(colony *parser.Colony) *Graph {
	g := &Graph{
		NameToID: make(map[string]int),
	}

	nodeID := 0

	// Assign IDs to all rooms
	for _, room := range colony.Rooms {
		if room.Name == colony.StartName || room.Name == colony.EndName {
			// Start and end: single node (no split)
			g.NameToID[room.Name] = nodeID
			g.IDToName = append(g.IDToName, room.Name)
			nodeID++
		} else {
			// Intermediate: split into _in and _out
			inID := nodeID
			outID := nodeID + 1
			g.NameToID[room.Name+"_in"] = inID
			g.NameToID[room.Name+"_out"] = outID
			g.IDToName = append(g.IDToName, room.Name+"_in", room.Name+"_out")
			nodeID += 2
		}
	}

	g.NodeCount = nodeID
	g.Adj = make([][]Edge, nodeID)
	g.StartID = g.NameToID[colony.StartName]
	g.EndID = g.NameToID[colony.EndName]

	// Add internal edges for split nodes (capacity 1)
	for _, room := range colony.Rooms {
		if room.Name == colony.StartName || room.Name == colony.EndName {
			continue
		}
		inID := g.NameToID[room.Name+"_in"]
		outID := g.NameToID[room.Name+"_out"]
		g.addEdge(inID, outID, 1)
	}

	// Add tunnel edges
	for _, link := range colony.Links {
		a, b := link[0], link[1]
		aOut := g.outNode(a, colony)
		bIn := g.inNode(b, colony)
		bOut := g.outNode(b, colony)
		aIn := g.inNode(a, colony)

		// Bidirectional: a_out -> b_in and b_out -> a_in
		g.addEdge(aOut, bIn, 1)
		g.addEdge(bOut, aIn, 1)
	}

	return g
}

// outNode returns the "out" node ID for a room (or the single node for start/end).
func (g *Graph) outNode(name string, colony *parser.Colony) int {
	if name == colony.StartName || name == colony.EndName {
		return g.NameToID[name]
	}
	return g.NameToID[name+"_out"]
}

// inNode returns the "in" node ID for a room (or the single node for start/end).
func (g *Graph) inNode(name string, colony *parser.Colony) int {
	if name == colony.StartName || name == colony.EndName {
		return g.NameToID[name]
	}
	return g.NameToID[name+"_in"]
}

// addEdge adds a directed edge with the given capacity and its reverse edge (capacity 0).
func (g *Graph) addEdge(from, to, cap int) {
	fwd := Edge{To: to, Cap: cap, Flow: 0, RevIdx: len(g.Adj[to])}
	rev := Edge{To: from, Cap: 0, Flow: 0, RevIdx: len(g.Adj[from])}
	g.Adj[from] = append(g.Adj[from], fwd)
	g.Adj[to] = append(g.Adj[to], rev)
}

// OriginalName extracts the original room name from an internal node name.
func OriginalName(internalName string) string {
	if len(internalName) > 3 && internalName[len(internalName)-3:] == "_in" {
		return internalName[:len(internalName)-3]
	}
	if len(internalName) > 4 && internalName[len(internalName)-4:] == "_out" {
		return internalName[:len(internalName)-4]
	}
	return internalName
}
