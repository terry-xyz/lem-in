package solver

import (
	"fmt"
	"math"
	"sort"

	"lem-in/internal/graph"
)

// Path is a sequence of original room names from start to end.
type Path struct {
	Rooms []string
}

// Length returns the number of edges in the path (rooms - 1).
func (p Path) Length() int {
	return len(p.Rooms) - 1
}

// FindPaths finds the optimal set of vertex-disjoint paths using Edmonds-Karp.
// It returns paths sorted by ascending length, or an error if no path exists.
func FindPaths(g *graph.Graph, antCount int) ([]Path, error) {
	// Phase 1: Find max flow by pushing all augmenting paths
	for {
		parent := bfs(g)
		if parent == nil {
			break
		}
		pushFlow(g, parent)
	}

	// Phase 2: Decompose flow into vertex-disjoint paths
	rawPaths := decomposePaths(g)
	if len(rawPaths) == 0 {
		return nil, fmt.Errorf("ERROR: invalid data format, no path from start to end")
	}

	// Convert to original room names, deduplicating consecutive _in/_out
	namedPaths := make([]Path, len(rawPaths))
	for i, p := range rawPaths {
		var rooms []string
		for _, nodeID := range p {
			name := graph.OriginalName(g.IDToName[nodeID])
			if len(rooms) == 0 || rooms[len(rooms)-1] != name {
				rooms = append(rooms, name)
			}
		}
		namedPaths[i] = Path{Rooms: rooms}
	}

	// Sort by ascending length
	sort.Slice(namedPaths, func(i, j int) bool {
		return namedPaths[i].Length() < namedPaths[j].Length()
	})

	// Phase 3: Find optimal subset of paths to use
	bestTurns := math.MaxInt64
	bestCount := 0
	for k := 1; k <= len(namedPaths); k++ {
		turns := computeTurns(namedPaths[:k], antCount)
		if turns < bestTurns {
			bestTurns = turns
			bestCount = k
		}
	}

	return namedPaths[:bestCount], nil
}

// bfs finds an augmenting path in the residual graph using BFS.
func bfs(g *graph.Graph) [][2]int {
	parent := make([][2]int, g.NodeCount)
	for i := range parent {
		parent[i] = [2]int{-1, -1}
	}
	parent[g.StartID] = [2]int{g.StartID, -1}

	queue := []int{g.StartID}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for idx, e := range g.Adj[curr] {
			if parent[e.To][0] == -1 && e.Cap-e.Flow > 0 {
				parent[e.To] = [2]int{curr, idx}
				if e.To == g.EndID {
					return parent
				}
				queue = append(queue, e.To)
			}
		}
	}
	return nil
}

// pushFlow pushes one unit of flow along the augmenting path found by BFS.
func pushFlow(g *graph.Graph, parent [][2]int) {
	node := g.EndID
	for node != g.StartID {
		prev := parent[node][0]
		edgeIdx := parent[node][1]
		g.Adj[prev][edgeIdx].Flow++
		revIdx := g.Adj[prev][edgeIdx].RevIdx
		g.Adj[node][revIdx].Flow--
		node = prev
	}
}

// decomposePaths extracts all flow paths from start to end by following positive flow edges.
func decomposePaths(g *graph.Graph) [][]int {
	var paths [][]int
	for {
		path := traceOnePath(g)
		if path == nil {
			break
		}
		paths = append(paths, path)
	}
	return paths
}

// traceOnePath traces a single path from start to end through edges with positive flow,
// decrementing flow as it goes.
func traceOnePath(g *graph.Graph) []int {
	path := []int{g.StartID}
	visited := make([]bool, g.NodeCount)
	visited[g.StartID] = true
	curr := g.StartID

	for curr != g.EndID {
		found := false
		for idx := range g.Adj[curr] {
			e := &g.Adj[curr][idx]
			if e.Flow > 0 && e.Cap > 0 && !visited[e.To] {
				path = append(path, e.To)
				visited[e.To] = true
				e.Flow--
				g.Adj[e.To][e.RevIdx].Flow++
				curr = e.To
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return path
}

// computeTurns calculates the minimum turns for antCount ants using given paths.
// Each ant on path i (length Li) takes Li turns to traverse. One new ant can enter
// per path per turn. So ai ants on path i finish at turn ai + Li - 1.
// Optimal: T = Lk - 1 + ceil((N - sumDiff) / k) where sumDiff = sum(Lk - Li).
func computeTurns(paths []Path, antCount int) int {
	k := len(paths)
	if k == 0 {
		return math.MaxInt64
	}

	lengths := make([]int, k)
	for i, p := range paths {
		lengths[i] = p.Length()
	}

	lk := lengths[k-1]
	sumDiff := 0
	for i := 0; i < k; i++ {
		sumDiff += lk - lengths[i]
	}

	remaining := antCount - sumDiff
	if remaining <= 0 {
		return math.MaxInt64
	}

	return lk - 1 + ceilDiv(remaining, k)
}

// ceilDiv rounds a positive division up so partial ant batches still count as a full turn.
func ceilDiv(a, b int) int {
	return (a + b - 1) / b
}
