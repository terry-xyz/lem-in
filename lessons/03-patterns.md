# Lesson 03: Patterns

## Why Learn Patterns?

A pattern is a **proven solution to a common problem**. Like cooking techniques: you don't reinvent "sauteing" every time you cook.

Recognizing patterns helps you:
1. **Read code faster** - "Oh, this is the state machine pattern"
2. **Write better code** - Use battle-tested solutions instead of inventing fragile ones
3. **Communicate** - "Let's use the pipeline pattern" tells another developer exactly what you mean
4. **Understand WHY** - Not just WHAT the code does, but the reasoning behind the structure

This project uses 7 patterns. Each one solves a specific problem.

---

### Pattern 1: Pipeline (Assembly Line)

**The Problem (Plain English)**
You have a complex task with multiple stages. If you put everything in one giant function, it becomes impossible to understand, test, or modify. Change one thing and everything breaks.

**The Solution (The Pattern)**
Break the task into **independent stages** connected in sequence. Each stage takes input, transforms it, and passes output to the next stage. No stage knows about the internals of any other.

**The Code**

```go
// WRONG way: one massive function
func solveEverything(filename string) {
    // 200 lines of parsing
    // 150 lines of graph building
    // 200 lines of solving
    // 100 lines of simulating
    // good luck debugging line 347
}

// RIGHT way: pipeline of independent stages
func main() {
    colony, err := parser.Parse(filename)       // Stage 1: Parse
    g := graph.BuildGraph(colony)               // Stage 2: Build graph
    paths, err := solver.FindPaths(g, ...)      // Stage 3: Find paths
    _, assignments := solver.DistributeAnts(...)// Stage 4: Assign ants
    moveLines := simulator.Simulate(...)        // Stage 5: Simulate
}
```

**Real-World Analogy**
- **Bad:** One person does all the cooking, plating, and serving. They're overwhelmed and if they mess up step 3, they have to restart everything.
- **Good:** A restaurant kitchen: prep cook chops vegetables, line cook grills meat, sous chef plates it, waiter serves. Each person is an expert at their stage.

**Code Location**
`cmd/lem-in/main.go:14-50`

---

### Pattern 2: State Machine (Command Parser)

**The Problem (Plain English)**
You're reading input line by line, and the meaning of each line depends on what came BEFORE it. The line `0 0 3` could be a regular room, OR it could be the start room - depending on whether `##start` appeared on the previous line.

**The Solution (The Pattern)**
Use **boolean flags** to track the current "state." When you encounter a command like `##start`, flip the flag on. When the next room line arrives, use the flag to decide what to do, then flip it off.

**The Code**

```go
// WRONG way: try to peek ahead or backtrack
// This gets messy with comments, blank lines, etc.
for i, line := range lines {
    if line == "##start" {
        nextLine := lines[i+1]  // What if i+1 is a comment? Or out of bounds?
        // ...
    }
}

// RIGHT way: state machine with pending flags
pendingStart := false
pendingEnd := false

for i := 1; i < len(lines); i++ {
    line := lines[i]

    if line == "##start" {
        pendingStart = true      // Remember: next room is the start
        continue
    }

    room, _ := parseRoom(line)

    if pendingStart {
        colony.StartName = room.Name   // THIS room is the start
        pendingStart = false            // Reset the flag
    }
}
```

**Real-World Analogy**
- **Bad:** Reading a recipe by looking ahead: "ingredient X... wait, is this for the SAUCE section or the MAIN section? Let me scan forward..."
- **Good:** Reading with a mental bookmark: "I just saw the 'SAUCE:' header, so the next ingredients are for the sauce."

**Code Location**
`internal/parser/parser.go:54-177` (the `pendingStart`/`pendingEnd` flags)

---

### Pattern 3: Node-Splitting (Vertex Capacity Reduction)

**The Problem (Plain English)**
The max-flow algorithm only understands **edge** capacities (how much can flow through a pipe). But our constraint is on **vertices** (rooms): each room holds max 1 ant. How do you make a vertex-based constraint work with an edge-based algorithm?

**The Solution (The Pattern)**
Split each constrained vertex into two nodes (`_in` and `_out`) connected by a capacity-1 edge. Now the vertex constraint IS an edge constraint.

**The Code**

```go
// WRONG way: try to track room occupancy separately
// This breaks the max-flow algorithm's guarantees
occupancy := map[string]int{}
// ... manual occupancy checking during flow computation
// (doesn't work - Edmonds-Karp assumes edge-only constraints)

// RIGHT way: transform the problem so edge constraints ARE vertex constraints
for _, room := range colony.Rooms {
    if room.Name == colony.StartName || room.Name == colony.EndName {
        g.NameToID[room.Name] = nodeID     // Single node (unlimited capacity)
        nodeID++
    } else {
        g.NameToID[room.Name+"_in"] = nodeID      // Entry node
        g.NameToID[room.Name+"_out"] = nodeID + 1  // Exit node
        nodeID += 2
    }
}
// Internal edge: _in → _out with capacity 1
g.addEdge(inID, outID, 1)
```

**Real-World Analogy**
- **Bad:** Telling the highway department "each gas station can only serve 1 car at a time" - they don't have rules for stations, only roads.
- **Good:** Building each gas station with a single-lane entrance road. Now "1 car per station" = "1 car per road" - the highway department understands roads!

**Code Location**
`internal/graph/graph.go:34-64` (node assignment + internal edges)

---

### Pattern 4: Residual Graph with Reverse Edges

**The Problem (Plain English)**
When finding multiple paths through a network, early decisions might block better solutions. Once you send flow down a path, how do you "undo" it if a better arrangement exists?

**The Solution (The Pattern)**
For every forward edge `A→B`, create a reverse edge `B→A` with capacity 0. When you push flow on `A→B`, the reverse edge's effective capacity increases (because `Cap - Flow` grows as Flow goes negative). This allows BFS to find paths that "undo" previous flow.

**The Code**

```go
// Every edge is paired with its reverse. They track each other via RevIdx.
func (g *Graph) addEdge(from, to, cap int) {
    fwd := Edge{
        To: to, Cap: cap, Flow: 0,
        RevIdx: len(g.Adj[to]),         // "My reverse is at index N in to's list"
    }
    rev := Edge{
        To: from, Cap: 0, Flow: 0,
        RevIdx: len(g.Adj[from]),       // "My reverse is at index M in from's list"
    }
    g.Adj[from] = append(g.Adj[from], fwd)
    g.Adj[to] = append(g.Adj[to], rev)
}

// When pushing flow: increment forward, decrement reverse
func pushFlow(g *graph.Graph, parent [][2]int) {
    for node != g.StartID {
        g.Adj[prev][edgeIdx].Flow++             // Forward: +1 flow
        revIdx := g.Adj[prev][edgeIdx].RevIdx
        g.Adj[node][revIdx].Flow--              // Reverse: -1 flow
        // (Cap 0, Flow -1 → available = 0-(-1) = 1: reverse path now open!)
    }
}
```

**Real-World Analogy**
- **Bad:** Booking flights with no cancellation allowed. You book A→C→D and realize A→B→D was better, but you're stuck.
- **Good:** Booking with free cancellation. You can "unbook" A→C to free capacity for a better overall arrangement.

**Code Location**
- `internal/graph/graph.go:99-104` (addEdge creates the pair)
- `internal/solver/solver.go:98-108` (pushFlow updates both)

---

### Pattern 5: BFS for Shortest Augmenting Path (Edmonds-Karp)

**The Problem (Plain English)**
To find max flow, you need to repeatedly find paths with available capacity. But using DFS (depth-first search) can lead to very long paths and slow convergence. You might end up exploring deep dead-ends before finding short paths.

**The Solution (The Pattern)**
Use **BFS** (breadth-first search) to always find the **shortest** augmenting path first. This is what makes Edmonds-Karp O(V*E^2) - a guaranteed polynomial bound. BFS explores level by level, so it naturally finds the closest route first.

**The Code**

```go
// BFS: explore all neighbors at distance 1 before distance 2, etc.
func bfs(g *graph.Graph) [][2]int {
    parent := make([][2]int, g.NodeCount)
    for i := range parent {
        parent[i] = [2]int{-1, -1}     // -1 = "not visited"
    }
    parent[g.StartID] = [2]int{g.StartID, -1}  // Mark start as visited

    queue := []int{g.StartID}           // FIFO queue (first in, first out)
    for len(queue) > 0 {
        curr := queue[0]                // Take from FRONT of queue
        queue = queue[1:]               // Remove from front

        for idx, e := range g.Adj[curr] {
            // Only follow edges with remaining capacity
            if parent[e.To][0] == -1 && e.Cap-e.Flow > 0 {
                parent[e.To] = [2]int{curr, idx}   // Remember how we got here
                if e.To == g.EndID {
                    return parent       // Found end! Return the path
                }
                queue = append(queue, e.To)  // Add to BACK of queue
            }
        }
    }
    return nil  // No path found → max flow complete
}
```

**Real-World Analogy**
- **Bad (DFS):** Looking for someone in a building by going as deep as possible first - down a hallway, into a room, into a closet, back out, into the next closet... You might search the entire basement before checking the lobby.
- **Good (BFS):** Checking all rooms on floor 1 first, then all rooms on floor 2, etc. If the person is on floor 2, you find them without searching floors 3-10.

**Code Location**
`internal/solver/solver.go:72-95`

---

### Pattern 6: Greedy Optimization with Exhaustive Search

**The Problem (Plain English)**
You found N parallel paths, but using all N might not be the fastest option. How do you pick the best number of paths to use?

**The Solution (The Pattern)**
Try all possibilities (k=1 through k=N), compute the turn count for each, and pick the best. Since paths are sorted by length and the formula is O(k) per evaluation, this exhaustive search is cheap.

**The Code**

```go
// WRONG way: assume more paths = faster
return namedPaths  // Always use all paths? No!

// RIGHT way: try every subset size, pick the best
bestTurns := math.MaxInt64
bestCount := 0
for k := 1; k <= len(namedPaths); k++ {
    turns := computeTurns(namedPaths[:k], antCount)
    if turns < bestTurns {
        bestTurns = turns
        bestCount = k
    }
}
return namedPaths[:bestCount]  // Only use the best k paths
```

**Real-World Analogy**
- **Bad:** Hiring every available truck to deliver 5 packages. One truck goes 1 mile, another goes 100 miles. You're waiting for the slow truck.
- **Good:** Calculate delivery time for 1 truck, 2 trucks, 3 trucks... and pick the number that minimizes total wait time.

**Code Location**
`internal/solver/solver.go:57-68`

---

### Pattern 7: Consumer-Only Decomposition (Path Extraction)

**The Problem (Plain English)**
After max-flow, the graph has flow values on edges but no explicit "list of paths." You need to extract the actual paths from the flow data.

**The Solution (The Pattern)**
Repeatedly trace a path from start to end following edges with positive flow, **consuming** (decrementing) the flow as you go. Each trace extracts one complete path and removes it from the flow, until no flow remains.

**The Code**

```go
// Extract one path at a time, consuming flow as we go
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
                e.Flow--                           // Consume the flow
                g.Adj[e.To][e.RevIdx].Flow++       // Update reverse
                curr = e.To
                found = true
                break
            }
        }
        if !found { return nil }  // Dead end (shouldn't happen with valid flow)
    }
    return path
}

// Keep extracting until no more paths exist
func decomposePaths(g *graph.Graph) [][]int {
    var paths [][]int
    for {
        path := traceOnePath(g)
        if path == nil { break }
        paths = append(paths, path)
    }
    return paths
}
```

**Real-World Analogy**
- **Bad:** Trying to figure out all the delivery routes by staring at a map of "how many trucks used each road."
- **Good:** Following one truck's route from warehouse to destination, crossing off each road as you go. Repeat for the next truck.

**Code Location**
`internal/solver/solver.go:111-150`

---

## Pattern Summary Table

| Pattern | Problem | Solution | Where |
|---------|---------|----------|-------|
| Pipeline | Monolithic complexity | Chain of independent stages | `cmd/lem-in/main.go` |
| State Machine | Context-dependent input | Boolean flags track state | `parser.go:54-177` |
| Node-Splitting | Vertex capacity → edge capacity | Split node into in/out pair | `graph.go:34-64` |
| Residual Graph | Undo bad flow decisions | Reverse edges with Cap=0 | `graph.go:99-104` |
| BFS Shortest Path | Efficient augmenting paths | Breadth-first search (FIFO) | `solver.go:72-95` |
| Exhaustive Optimization | Pick best subset size | Try all k, keep best | `solver.go:57-68` |
| Consumer Decomposition | Extract paths from flow | Trace + decrement flow | `solver.go:111-150` |

---

## Next: [04 - Line by Line](04-line-by-line.md) - Deep walkthrough of every important function
