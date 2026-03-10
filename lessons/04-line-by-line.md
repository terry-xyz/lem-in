# Lesson 04: Line by Line

## Reading Go Code: Quick Syntax Guide

```go
func name(param Type) ReturnType {  // Function: takes param, returns ReturnType
    // param is input, ReturnType is output
}

if condition {     // If statement: runs block when condition is true
    // do this
}

for i := 0; i < 10; i++ {   // Loop: i goes 0,1,2,...,9
    // i++ means "add 1 to i"
}

for _, item := range slice {  // Range loop: iterate every item in a slice
    // _ means "ignore the index"
}

&variable          // Get the memory address of variable (a "pointer")
*pointer           // Get the value that a pointer points to
slice = append(slice, item)  // Add item to end of slice
map[key]           // Look up value by key in a map
make(map[K]V)      // Create a new empty map
make([]T, n)       // Create a new slice of length n
len(slice)         // Number of items in a slice
nil                // "nothing" / empty / not-yet-created
```

---

## Section 1: The Entry Point

**Big Picture:** The program starts here. It reads a filename from the command line, runs the four pipeline stages, and prints the result. Think of it as the factory manager who calls each department.

**File:** `cmd/lem-in/main.go:14-50`

```go
func main() {
    // STEP 1: Check that the user gave us a filename
    if len(os.Args) < 2 {
        // os.Args is the list of command-line arguments
        // os.Args[0] is the program name, [1] would be the file
        fmt.Println("USAGE: go run . <your-file.txt>")
        return  // Exit early - nothing to do
    }

    filename := os.Args[1]  // The file path the user typed

    // STEP 2: Parse the input file into a Colony struct
    colony, err := parser.Parse(filename)
    if err != nil {
        fmt.Println(err)  // Print the error message (e.g., "ERROR: invalid data format, ...")
        os.Exit(1)        // Exit with error code 1 (non-zero = failure)
    }

    // STEP 3: Build the flow network (with node-splitting)
    g := graph.BuildGraph(colony)

    // STEP 4: Find optimal paths using Edmonds-Karp
    paths, err := solver.FindPaths(g, colony.AntCount)
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    // STEP 5: Assign ants to paths
    // _ discards the first return value (antsPerPath) - we only need assignments
    _, assignments := solver.DistributeAnts(paths, colony.AntCount)

    // STEP 6: Generate turn-by-turn output
    moveLines := simulator.Simulate(paths, assignments)

    // STEP 7: Print the result
    fmt.Println(strings.Join(colony.Lines, "\n"))  // Echo original input
    fmt.Println()                                   // Blank separator line
    for _, line := range moveLines {
        fmt.Println(line)                           // Print each turn's moves
    }
}
```

**Key Insight:** The `_` in `_, assignments := ...` is Go's way of saying "I don't need this value." The function returns two things, but we only care about the second one here. This is idiomatic Go - you'll see it everywhere.

---

## Section 2: The Parser

### 2a: File Reading and Normalization

**Big Picture:** The parser reads a file from disk, normalizes line endings (Windows uses `\r\n`, Unix uses `\n`), and splits the text into individual lines. This ensures the rest of the parser works the same on any operating system.

**File:** `internal/parser/parser.go:13-30`

```go
// INPUT:  filename (path to the input file)
// OUTPUT: *Colony (structured data) or error

func Parse(filename string) (*Colony, error) {
    // STEP 1: Read the entire file into memory as raw bytes
    data, err := os.ReadFile(filename)
    if err != nil {
        // File doesn't exist, no permission, etc.
        return nil, fmt.Errorf("ERROR: invalid data format, cannot read file")
    }

    // STEP 2: Convert bytes to text and normalize line endings
    content := string(data)                            // bytes → string
    content = strings.ReplaceAll(content, "\r\n", "\n") // Windows → Unix
    content = strings.TrimRight(content, "\n")          // Remove trailing newlines

    // STEP 3: Handle empty file
    if content == "" {
        return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
    }

    // STEP 4: Split into lines and delegate to the line parser
    lines := strings.Split(content, "\n")
    return parseLines(lines)  // This does all the actual work
}
```

**Key Insight:** `\r\n` normalization is critical. Without it, room names would have an invisible `\r` at the end on Windows, causing silent lookup failures.

### 2b: The State Machine

**Big Picture:** This is the heart of the parser. It reads lines one at a time, using a state machine (boolean flags) to track whether the next room should be marked as start or end. Lines can be: ant count, comments, commands, rooms, links, or blank lines.

**File:** `internal/parser/parser.go:32-197`

```go
// INPUT:  lines (all lines from the file, split by newline)
// OUTPUT: *Colony or error

func parseLines(lines []string) (*Colony, error) {
    // STEP 1: Initialize the Colony with empty containers
    c := &Colony{
        RoomMap: make(map[string]int),  // Empty lookup table
        Lines:   lines,                 // Save original text for echoing
    }

    // STEP 2: Parse ant count (always the first line)
    antStr := strings.TrimSpace(lines[0])    // Remove leading/trailing whitespace
    antCount, err := strconv.Atoi(antStr)    // "42" → 42
    if err != nil || antCount <= 0 {
        return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
    }
    if antCount > maxAnts {  // maxAnts = 10,000,000
        return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
    }
    c.AntCount = antCount

    // STEP 3: State machine for rooms and links
    pendingStart := false   // "The next room is the start room"
    pendingEnd := false     // "The next room is the end room"
    startFound := false     // "We've already seen a start room"
    endFound := false       // "We've already seen an end room"
    inLinks := false        // "We've transitioned to the links section"
    linkSet := make(map[string]bool)  // For duplicate link detection

    for i := 1; i < len(lines); i++ {
        line := lines[i]

        // --- Handle ## commands ---
        if strings.HasPrefix(line, "##") {
            cmd := line[2:]  // Strip the "##" prefix
            switch cmd {
            case "start":
                if startFound {
                    return nil, fmt.Errorf("ERROR: ... duplicate start command")
                }
                if pendingStart || pendingEnd {
                    // Can't have ##start right after ##start or ##end
                    return nil, fmt.Errorf("ERROR: ... invalid command placement")
                }
                pendingStart = true  // FLAG: next room is start
            case "end":
                // (mirror logic for end)
                pendingEnd = true    // FLAG: next room is end
            default:
                // Unknown ## commands: silently ignored (spec says so)
            }
            continue  // Skip to next line
        }

        // --- Handle # comments ---
        if strings.HasPrefix(line, "#") {
            continue  // Comments are preserved in Lines but don't affect data
        }

        // --- Skip blank lines ---
        if strings.TrimSpace(line) == "" {
            continue
        }

        // --- Try parsing as a link (contains dash, no spaces) ---
        if isLink(line) {
            if pendingStart || pendingEnd {
                // A command was expecting a room, but got a link instead
                return nil, fmt.Errorf("ERROR: ... invalid command placement")
            }
            inLinks = true  // We're now in the links section

            parts := strings.SplitN(line, "-", 2)  // "A-B" → ["A", "B"]
            name1, name2 := parts[0], parts[1]

            // Validate: both rooms must exist, no self-links
            // Normalize for duplicate detection: always store "a-b" where a < b
            a, b := name1, name2
            if a > b { a, b = b, a }
            key := a + "-" + b
            if linkSet[key] {
                return nil, fmt.Errorf("ERROR: ... duplicate link")
            }
            linkSet[key] = true

            c.Links = append(c.Links, [2]string{name1, name2})
            continue
        }

        // --- If we're past links, reject non-link data ---
        if inLinks {
            return nil, fmt.Errorf("ERROR: ... invalid data")
        }

        // --- Parse as a room: "name X Y" ---
        room, err := parseRoom(line)  // Splits by spaces, validates coords
        if err != nil { return nil, err }

        // Validate room name (no "L" prefix, no "#" prefix, no dashes)
        // Check for duplicates via RoomMap lookup

        idx := len(c.Rooms)
        c.Rooms = append(c.Rooms, room)
        c.RoomMap[room.Name] = idx  // "roomA" → index 3

        // --- Apply pending commands ---
        if pendingStart {
            c.StartName = room.Name  // THIS room is start
            startFound = true
            pendingStart = false     // Reset the flag
        }
        if pendingEnd {
            c.EndName = room.Name    // THIS room is end
            endFound = true
            pendingEnd = false       // Reset the flag
        }
    }

    // STEP 4: Final validation
    if pendingStart || pendingEnd {
        // A ##start or ##end was never followed by a room
        return nil, fmt.Errorf("ERROR: ... invalid command placement")
    }
    if !startFound { return nil, fmt.Errorf("ERROR: ... no start room found") }
    if !endFound   { return nil, fmt.Errorf("ERROR: ... no end room found") }
    if c.StartName == c.EndName {
        return nil, fmt.Errorf("ERROR: ... start and end are the same room")
    }

    return c, nil
}
```

**Key Insight:** Notice the `if pendingStart { ... } if pendingEnd { ... }` pattern (lines 168-177). These are two separate `if` statements, NOT `if/else if`. This means a room could theoretically be BOTH start AND end if `##start` and `##end` appear on consecutive lines before the same room. The final `c.StartName == c.EndName` check catches this.

---

## Section 3: The Graph Builder

**Big Picture:** This function transforms a human-friendly Colony (room names, links) into a math-friendly flow network (numbered nodes, directed edges with capacities). The key transformation is node-splitting for intermediate rooms.

**File:** `internal/graph/graph.go:26-80`

```go
// INPUT:  colony (parsed rooms and links)
// OUTPUT: *Graph (flow network ready for Edmonds-Karp)

func BuildGraph(colony *parser.Colony) *Graph {
    g := &Graph{
        NameToID: make(map[string]int),  // Will map "room_in" → node 3, etc.
    }

    nodeID := 0

    // STEP 1: Assign numeric IDs to all rooms
    for _, room := range colony.Rooms {
        if room.Name == colony.StartName || room.Name == colony.EndName {
            // Start and end: ONE node (unlimited capacity)
            g.NameToID[room.Name] = nodeID
            g.IDToName = append(g.IDToName, room.Name)
            nodeID++
        } else {
            // Intermediate rooms: TWO nodes (_in and _out)
            g.NameToID[room.Name+"_in"] = nodeID      // Entry point
            g.NameToID[room.Name+"_out"] = nodeID + 1  // Exit point
            g.IDToName = append(g.IDToName, room.Name+"_in", room.Name+"_out")
            nodeID += 2
        }
    }

    g.NodeCount = nodeID
    g.Adj = make([][]Edge, nodeID)  // Pre-allocate adjacency list
    g.StartID = g.NameToID[colony.StartName]
    g.EndID = g.NameToID[colony.EndName]

    // STEP 2: Add internal edges for split nodes (capacity 1)
    // This is where the vertex-capacity constraint becomes an edge-capacity constraint
    for _, room := range colony.Rooms {
        if room.Name == colony.StartName || room.Name == colony.EndName {
            continue  // Start/end are NOT split
        }
        inID := g.NameToID[room.Name+"_in"]
        outID := g.NameToID[room.Name+"_out"]
        g.addEdge(inID, outID, 1)  // room_in → room_out, cap 1
        // Also creates reverse edge: room_out → room_in, cap 0
    }

    // STEP 3: Add tunnel edges (bidirectional)
    for _, link := range colony.Links {
        a, b := link[0], link[1]
        // outNode("A") = "A_out" (or "A" if start/end)
        // inNode("B") = "B_in" (or "B" if start/end)
        aOut := g.outNode(a, colony)
        bIn := g.inNode(b, colony)
        bOut := g.outNode(b, colony)
        aIn := g.inNode(a, colony)

        // A→B: a_out → b_in (you exit A, then enter B)
        g.addEdge(aOut, bIn, 1)
        // B→A: b_out → a_in (reverse direction, since tunnels are bidirectional)
        g.addEdge(bOut, aIn, 1)
    }

    return g
}
```

**Key Insight:** Why `aOut → bIn` and not `aIn → bIn`? Because in the split model, traffic flows: enter room A (`a_in`) → pass through A (`a_in → a_out`, cap 1) → exit A to enter B (`a_out → b_in`) → pass through B (`b_in → b_out`, cap 1). The `_out → _in` edges represent tunnels; the `_in → _out` edges represent room capacity.

---

## Section 4: The Solver (Edmonds-Karp)

### 4a: Main Algorithm

**Big Picture:** This orchestrates the three phases: find max flow, decompose into paths, and select the optimal subset. It's the "brain" of the program.

**File:** `internal/solver/solver.go:23-69`

```go
// INPUT:  g (flow network), antCount (number of ants)
// OUTPUT: []Path (optimal paths, sorted by length) or error

func FindPaths(g *graph.Graph, antCount int) ([]Path, error) {
    // PHASE 1: Find maximum flow (Edmonds-Karp)
    // Keep finding augmenting paths and pushing flow until no more exist
    for {
        parent := bfs(g)    // Find shortest augmenting path
        if parent == nil {
            break            // No more paths → max flow found
        }
        pushFlow(g, parent)  // Push 1 unit of flow along the path
    }
    // At this point, the graph has flow values on all edges

    // PHASE 2: Extract vertex-disjoint paths from the flow
    rawPaths := decomposePaths(g)  // Returns [][]int (node ID sequences)
    if len(rawPaths) == 0 {
        return nil, fmt.Errorf("ERROR: invalid data format, no path from start to end")
    }

    // Convert node IDs to original room names, removing _in/_out duplicates
    namedPaths := make([]Path, len(rawPaths))
    for i, p := range rawPaths {
        var rooms []string
        for _, nodeID := range p {
            name := graph.OriginalName(g.IDToName[nodeID])
            // OriginalName("room_in") → "room", OriginalName("room_out") → "room"
            if len(rooms) == 0 || rooms[len(rooms)-1] != name {
                rooms = append(rooms, name)
                // Skip consecutive duplicates: "room_in" and "room_out" both → "room"
            }
        }
        namedPaths[i] = Path{Rooms: rooms}
    }

    // Sort paths: shortest first (important for the optimization formula)
    sort.Slice(namedPaths, func(i, j int) bool {
        return namedPaths[i].Length() < namedPaths[j].Length()
    })

    // PHASE 3: Pick the best number of paths to use
    bestTurns := math.MaxInt64  // Start with "infinity"
    bestCount := 0
    for k := 1; k <= len(namedPaths); k++ {
        turns := computeTurns(namedPaths[:k], antCount)
        if turns < bestTurns {
            bestTurns = turns
            bestCount = k
        }
    }

    return namedPaths[:bestCount], nil  // Return only the best k paths
}
```

### 4b: BFS (Breadth-First Search)

**Big Picture:** BFS explores the graph level by level, finding the shortest path from start to end that still has available capacity. It returns a "parent" array that lets you trace the path backwards from end to start.

**File:** `internal/solver/solver.go:72-95`

```go
// INPUT:  g (graph with current flow state)
// OUTPUT: parent array (how to trace path from end to start) or nil (no path)

func bfs(g *graph.Graph) [][2]int {
    // parent[node] = [prevNode, edgeIndex]
    // -1 means "not visited yet"
    parent := make([][2]int, g.NodeCount)
    for i := range parent {
        parent[i] = [2]int{-1, -1}
    }
    parent[g.StartID] = [2]int{g.StartID, -1}  // Start is "visited by itself"

    queue := []int{g.StartID}  // BFS queue: nodes to explore

    for len(queue) > 0 {
        curr := queue[0]    // Take first element (FIFO = breadth-first)
        queue = queue[1:]   // Remove it from the queue

        // Try every edge from the current node
        for idx, e := range g.Adj[curr] {
            // Can we use this edge?
            // parent[e.To][0] == -1: target not visited yet
            // e.Cap - e.Flow > 0: edge has remaining capacity
            if parent[e.To][0] == -1 && e.Cap-e.Flow > 0 {
                parent[e.To] = [2]int{curr, idx}  // "I came from curr via edge idx"

                if e.To == g.EndID {
                    return parent  // Found the end! Return immediately
                }
                queue = append(queue, e.To)  // Explore this node later
            }
        }
    }
    return nil  // Explored everything, no path to end
}
```

**Understanding the parent array:**
```
If BFS finds: start(0) → node(3) → node(5) → end(7)

parent[7] = [5, 2]   → "I reached node 7 from node 5, via edge at index 2"
parent[5] = [3, 1]   → "I reached node 5 from node 3, via edge at index 1"
parent[3] = [0, 0]   → "I reached node 3 from node 0, via edge at index 0"
parent[0] = [0, -1]  → "I'm the start"

To trace the path: start at end, follow parent links backward.
```

### 4c: Push Flow

**Big Picture:** Once BFS finds a path, this function traces it backwards and pushes one unit of flow along every edge. It also updates reverse edges so the algorithm can "undo" this choice later if needed.

**File:** `internal/solver/solver.go:98-108`

```go
// INPUT:  g (graph), parent (path found by BFS)
// OUTPUT: (modifies graph in-place, no return value)

func pushFlow(g *graph.Graph, parent [][2]int) {
    node := g.EndID  // Start tracing from the end

    for node != g.StartID {
        prev := parent[node][0]       // Where did we come from?
        edgeIdx := parent[node][1]    // Which edge did we use?

        g.Adj[prev][edgeIdx].Flow++   // Forward edge: +1 flow
        // This means Cap-Flow decreases by 1 (less remaining capacity)

        revIdx := g.Adj[prev][edgeIdx].RevIdx  // Find the reverse edge
        g.Adj[node][revIdx].Flow--    // Reverse edge: -1 flow
        // Cap=0, Flow=-1 → Cap-Flow = 0-(-1) = 1 (reverse path now OPEN)

        node = prev  // Move backwards along the path
    }
}
```

**Key Insight:** The reverse edge's flow goes NEGATIVE. This is intentional. When `Cap=0` and `Flow=-1`, the available capacity is `Cap - Flow = 0 - (-1) = 1`. This "phantom capacity" on reverse edges is what allows Edmonds-Karp to reroute flow.

### 4d: Turn Count Formula

**Big Picture:** Given a set of paths and an ant count, this calculates the minimum number of turns needed. It's the mathematical core that decides which subset of paths is optimal.

**File:** `internal/solver/solver.go:152-183`

```go
// INPUT:  paths (sorted by length), antCount
// OUTPUT: minimum turns needed

func computeTurns(paths []Path, antCount int) int {
    k := len(paths)
    if k == 0 { return math.MaxInt64 }

    lengths := make([]int, k)
    for i, p := range paths {
        lengths[i] = p.Length()  // Number of edges (rooms - 1)
    }

    lk := lengths[k-1]  // Longest path length (paths are sorted ascending)

    // sumDiff = how much "slack" the shorter paths have compared to the longest
    // If paths are [2, 3, 5], sumDiff = (5-2) + (5-3) + (5-5) = 3+2+0 = 5
    sumDiff := 0
    for i := 0; i < k; i++ {
        sumDiff += lk - lengths[i]
    }

    // remaining = ants that AREN'T "used up" by the slack
    remaining := antCount - sumDiff
    if remaining <= 0 {
        return math.MaxInt64  // Not enough ants to fill even one per path
    }

    // T = longest_path_length - 1 + ceil(remaining / num_paths)
    return lk - 1 + ceilDiv(remaining, k)
}

// ceilDiv: divide and round UP
// Example: ceilDiv(7, 3) = 3  (because 7/3 = 2.33, rounded up = 3)
func ceilDiv(a, b int) int {
    return (a + b - 1) / b  // Classic integer ceiling division trick
}
```

**Why `Lk - 1`?** Because the last ant on the longest path takes `Lk` turns to traverse it, but it doesn't enter on turn 0 - it enters on the last possible turn. The `-1` accounts for the fact that an ant entering on turn T finishes on turn `T + Lk - 1`, and we want all ants to finish by turn T. So `T = Lk - 1 + (number of "waves" of ants)`.

---

## Section 5: Ant Distribution

**Big Picture:** Given the optimal paths and ant count, this function decides exactly which ants go on which paths. It ensures the total turns are minimized by giving more ants to shorter paths.

**File:** `internal/solver/distribute.go:14-67`

```go
// INPUT:  paths (sorted by length), antCount
// OUTPUT: antsPerPath (how many on each), assignments (ant ID → path)

func DistributeAnts(paths []Path, antCount int) ([]int, []AntAssignment) {
    k := len(paths)
    if k == 0 { return nil, nil }

    // STEP 1: Find optimal number of paths (same as in FindPaths)
    bestTurns := math.MaxInt64
    bestK := 1
    for i := 1; i <= k; i++ {
        t := computeTurns(paths[:i], antCount)
        if t < bestTurns {
            bestTurns = t
            bestK = i
        }
    }
    T := bestTurns  // The optimal turn count

    // STEP 2: Assign ants to paths
    // Formula: path i gets T - L_i + 1 ants
    // (so all paths finish at the same turn T)
    antsPerPath := make([]int, k)
    totalAssigned := 0
    for i := 0; i < bestK; i++ {
        count := T - paths[i].Length() + 1
        if count < 0 { count = 0 }
        antsPerPath[i] = count
        totalAssigned += count
    }

    // STEP 3: Fix rounding errors
    // Due to ceiling division, we might assign too many ants
    // Remove excess from the longest paths first
    excess := totalAssigned - antCount
    for i := bestK - 1; excess > 0 && i >= 0; i-- {
        if antsPerPath[i] > 0 {
            antsPerPath[i]--
            excess--
        }
    }

    // STEP 4: Create ordered assignments
    // Lower ant IDs go on shorter paths (they exit faster)
    var assignments []AntAssignment
    antID := 1
    for pathIdx := 0; pathIdx < k; pathIdx++ {
        for j := 0; j < antsPerPath[pathIdx]; j++ {
            assignments = append(assignments, AntAssignment{
                AntID:     antID,   // Ant 1 gets shortest path
                PathIndex: pathIdx,
            })
            antID++
        }
    }

    return antsPerPath, assignments
}
```

**Key Insight:** The excess adjustment (Step 3) is necessary because `ceilDiv` rounds UP, so the formula might over-count by up to `k-1` ants. We remove the excess from the longest paths because they're the least efficient - one fewer ant on a long path saves more turns than one fewer ant on a short path.

---

## Section 6: The Simulator

**Big Picture:** This takes the abstract plan (paths + assignments) and "acts it out," generating the human-readable output. Each turn: advance all active ants one step, launch new ants, and record all movements.

**File:** `internal/simulator/simulator.go:20-116`

```go
// INPUT:  paths (room sequences), assignments (ant → path mapping)
// OUTPUT: []string (one line per turn: "L1-room L2-room ...")

func Simulate(paths []solver.Path, assignments []solver.AntAssignment) []string {
    if len(assignments) == 0 { return nil }

    // STEP 1: Group ants by their assigned path
    // pathAnts[0] = [1, 2, 3] means ants 1,2,3 are on path 0
    pathAnts := make([][]int, len(paths))
    for _, a := range assignments {
        pathAnts[a.PathIndex] = append(pathAnts[a.PathIndex], a.AntID)
    }

    var active []*antState              // Ants currently "in transit"
    nextAnt := make([]int, len(paths))  // Index of next ant to launch per path

    var lines []string

    // STEP 2: Main simulation loop (one iteration = one turn)
    for {
        var moves []struct{ antID int; room string }

        // STEP 2a: Advance all active ants one step forward
        var stillActive []*antState
        for _, ant := range active {
            path := paths[ant.PathIndex]
            ant.StepIndex++                         // Move forward one room
            room := path.Rooms[ant.StepIndex]       // What room are we in now?
            moves = append(moves, struct{ antID int; room string }{ant.AntID, room})

            if ant.StepIndex < len(path.Rooms)-1 {
                stillActive = append(stillActive, ant)  // Not at end, keep going
            }
            // If at end (StepIndex == len-1), ant is NOT added to stillActive → removed
        }
        active = stillActive  // Replace with only the still-active ants

        // STEP 2b: Launch new ants (one per path per turn)
        for pathIdx := 0; pathIdx < len(paths); pathIdx++ {
            if nextAnt[pathIdx] >= len(pathAnts[pathIdx]) {
                continue  // No more ants to launch on this path
            }
            antID := pathAnts[pathIdx][nextAnt[pathIdx]]
            nextAnt[pathIdx]++

            path := paths[pathIdx]

            // Special case: direct path (start → end, length 1)
            if len(path.Rooms) <= 1 {
                moves = append(moves, struct{ antID int; room string }{
                    antID, path.Rooms[len(path.Rooms)-1],
                })
                continue  // No intermediate steps needed
            }

            // Normal: ant enters at step 1 (first room after start)
            ant := &antState{
                AntID:     antID,
                PathIndex: pathIdx,
                StepIndex: 1,  // Skip start room (step 0)
            }
            room := path.Rooms[1]
            moves = append(moves, struct{ antID int; room string }{antID, room})

            if ant.StepIndex < len(path.Rooms)-1 {
                active = append(active, ant)  // Add to active if not already at end
            }
        }

        // STEP 2c: If no moves happened, simulation is complete
        if len(moves) == 0 { break }

        // STEP 2d: Sort moves by ant ID (spec requires ascending order)
        sort.Slice(moves, func(i, j int) bool {
            return moves[i].antID < moves[j].antID
        })

        // STEP 2e: Build the output line
        var sb strings.Builder
        for i, m := range moves {
            if i > 0 { sb.WriteByte(' ') }
            fmt.Fprintf(&sb, "L%d-%s", m.antID, m.room)
        }
        lines = append(lines, sb.String())
    }

    return lines
}
```

**Key Insight:** The `stillActive` pattern (line 44-58) is how the simulator avoids memory leaks. Instead of removing elements from the `active` slice (which is expensive and error-prone), it creates a NEW slice each turn containing only the ants that haven't finished. The old slice is garbage-collected.

**Understanding the antState lifecycle:**

```
BORN:      antState created at StepIndex=1 when launched
ADVANCING: StepIndex incremented each turn (+1)
RETIRED:   When StepIndex == len(path.Rooms)-1, not added to stillActive → gone
```

---

## Section 7: The Format Parser

**Big Picture:** This does the reverse of the solver's output: takes the printed text and converts it back into structured data. Used by the TUI and web visualizers to understand what happened.

**File:** `internal/format/format.go:35-146`

```go
// INPUT:  output (full text from solver's stdout)
// OUTPUT: *ParsedOutput (structured rooms, links, movements) or error

func ParseOutput(output string) (*ParsedOutput, error) {
    // STEP 1: Normalize and check for errors
    output = strings.ReplaceAll(output, "\r\n", "\n")
    output = strings.TrimRight(output, "\n")
    if strings.HasPrefix(output, "ERROR:") {
        return &ParsedOutput{Error: output}, nil  // Pass error through
    }

    lines := strings.Split(output, "\n")
    result := &ParsedOutput{}

    // STEP 2: Find the blank separator line (divides input echo from moves)
    sepIdx := -1
    for i, line := range lines {
        if strings.TrimSpace(line) == "" {
            sepIdx = i
            break
        }
    }

    // STEP 3: Parse file section (before separator)
    // Line 0: ant count
    // Lines 1+: rooms (name X Y), links (name1-name2), commands (##start/##end)
    // Uses isStart/isEnd flags similar to the parser's state machine

    // STEP 4: Parse move section (after separator)
    // Each line: "L1-roomA L2-roomB L3-roomC"
    // Split by spaces, parse each token: "L" + antID + "-" + roomName
    for _, tok := range tokens {
        if !strings.HasPrefix(tok, "L") { continue }
        dashIdx := strings.Index(tok[1:], "-")  // Find first dash AFTER the "L"
        antID, _ := strconv.Atoi(tok[1 : 1+dashIdx])
        roomName := tok[2+dashIdx:]
        turn = append(turn, Movement{AntID: antID, RoomName: roomName})
    }

    return result, nil
}
```

**Key Insight:** The format parser uses `strings.Index(tok[1:], "-")` not `strings.Index(tok, "-")` because ant IDs might contain characters before the dash. By skipping the `L`, it finds the first dash in the ant-ID-room boundary, not any dash that might appear in a room name... except room names can't start with `L` (parser rule), so the `L` prefix is unambiguous.

---

## Next: [05 - Exercises](05-exercises.md) - Test your understanding with hands-on practice
