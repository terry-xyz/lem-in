# Lesson 01: Core Concepts

## Reading Go Code: Quick Guide

Before diving in, here's a cheat sheet for reading Go syntax:

```go
package main              // This file belongs to the "main" package

import "fmt"              // Use the "fmt" (format) library for printing

func add(a int, b int) int {   // Function: takes two ints, returns one int
    return a + b                // Send the result back to whoever called us
}

type Room struct {         // Define a new type called "Room" (like a form template)
    Name string            // Field "Name" holds text
    X    int               // Field "X" holds a whole number
}

if x > 10 {               // If x is greater than 10...
    fmt.Println("big")    // ...print "big"
}

for i := 0; i < 5; i++ {  // Repeat 5 times (i goes 0,1,2,3,4)
    fmt.Println(i)         // Print the current number
}

// :=  means "create a new variable and assign a value"
name := "Alice"            // Create variable "name", set it to "Alice"

// []string is a "slice" (a growable list of strings)
rooms := []string{"A", "B", "C"}

// map[string]int is a "dictionary" (lookup table: string keys → int values)
index := map[string]int{"A": 0, "B": 1}
```

**Key Go quirks:**
- No semicolons needed (the compiler adds them)
- Opening brace `{` must be on the same line as the `if`/`for`/`func`
- Capitalized names (`BuildGraph`) are public; lowercase (`bfs`) are private
- `:=` creates + assigns; `=` reassigns an existing variable
- `nil` means "nothing" (like `null` in other languages)

---

## The 6 Core Concepts

This project is built on six key ideas. Master these and you understand the whole system.

---

### Concept 1: The Colony (Structured Input)

**What (Simple Definition)**
> A Colony is all the information extracted from the input file: how many ants, which rooms exist, how they connect, and which is start/end.

**Why (Why This Matters)**
Without structured data, the program would be working with raw text every time it needed to check "is room X connected to room Y?" That would be slow and error-prone. The Colony is like filling out a standardized form from a handwritten letter - once the form is filled, everyone downstream can work with clean data.

**Where (Files)**
- `internal/parser/types.go:4-8` - Room struct
- `internal/parser/types.go:11-19` - Colony struct
- `internal/parser/parser.go:13-30` - Parse function (creates the Colony)

**How (Code Walkthrough)**
```go
// A Room is a named location with coordinates (used for visualization)
type Room struct {
    Name string    // Human-readable name like "start", "A", "room3"
    X    int       // X position on the map (for drawing, not for solving)
    Y    int       // Y position on the map (same - visual only)
}

// Colony = everything we know after reading the input file
type Colony struct {
    AntCount  int              // How many ants to move (e.g., 4)
    Rooms     []Room           // All rooms in order they appeared
    RoomMap   map[string]int   // Quick lookup: "roomA" → index 2
    Links     [][2]string      // Tunnel connections: [["A","B"], ["B","C"]]
    StartName string           // Which room name is the start
    EndName   string           // Which room name is the end
    Lines     []string         // Original file text (for echoing back)
}
```

**Key Insight:** The `RoomMap` exists for O(1) lookups. Without it, checking "does room X exist?" would require scanning the entire `Rooms` slice every time.

---

### Concept 2: Node-Splitting (The Capacity Trick)

**What (Simple Definition)**
> Node-splitting transforms each intermediate room into two connected nodes (`room_in` and `room_out`) with a capacity-1 edge between them, so the max-flow algorithm naturally enforces "only one ant in a room at a time."

**Why (Why This Matters)**
This is the cleverest part of the entire project. Here's the problem: standard max-flow algorithms (like Edmonds-Karp) handle **edge capacities** - they can limit how much "stuff" flows through a tunnel. But our constraint is on **rooms** (vertices), not tunnels (edges).

Real-world analogy: imagine a highway system where each tunnel (edge) can carry 1 car. Standard traffic algorithms handle that. But we also need each gas station (room) to hold only 1 car. That's a **vertex capacity** constraint, which standard algorithms don't directly support.

The trick: **split each gas station into an entrance and exit**, connected by a single-lane road. Now the "only 1 car in the station" constraint becomes "only 1 car on this road" - which IS an edge capacity!

**Where (Files)**
- `internal/graph/graph.go:26-80` - BuildGraph (does the splitting)
- `internal/graph/graph.go:83-96` - outNode/inNode helpers
- `internal/graph/graph.go:107-115` - OriginalName (reverses the split)

**How (Visualized)**

Before splitting:
```
        ┌─── B ───┐
  start ┤         ├─ end
        └─── C ───┘
```

After splitting:
```
         ┌── B_in ──(cap 1)── B_out ──┐
  start ─┤                            ├─ end
         └── C_in ──(cap 1)── C_out ──┘
```

Notice: `start` and `end` are NOT split. They have unlimited capacity (many ants can leave start per turn, many can arrive at end per turn).

---

### Concept 3: Residual Graph (Flow Bookkeeping)

**What (Simple Definition)**
> A residual graph tracks both the current flow and remaining capacity on every edge, plus a "reverse edge" that lets the algorithm undo previous decisions.

**Why (Why This Matters)**
Imagine you're filling water pipes. You send water down path A, but later realize path B would have been better. In a real pipe system, you can't un-send water. But in a residual graph, you CAN - by sending flow along the **reverse edge**, which effectively cancels the original flow.

Without reverse edges, the Edmonds-Karp algorithm might get stuck in a suboptimal solution. Reverse edges let it "change its mind."

**Where (Files)**
- `internal/graph/graph.go:6-11` - Edge struct (Cap, Flow, RevIdx)
- `internal/graph/graph.go:99-104` - addEdge (creates forward + reverse pair)
- `internal/solver/solver.go:98-108` - pushFlow (updates both edges)

**How (Code Walkthrough)**
```go
type Edge struct {
    To     int  // Where this edge goes
    Cap    int  // Maximum flow allowed (1 for tunnels, 0 for reverse edges)
    Flow   int  // Current flow through this edge
    RevIdx int  // "If you need to undo me, I'm at g.Adj[To][RevIdx]"
}

// Every real edge gets a shadow "reverse" edge with capacity 0
func (g *Graph) addEdge(from, to, cap int) {
    fwd := Edge{To: to, Cap: cap, Flow: 0, RevIdx: len(g.Adj[to])}
    rev := Edge{To: from, Cap: 0, Flow: 0, RevIdx: len(g.Adj[from])}
    g.Adj[from] = append(g.Adj[from], fwd)
    g.Adj[to] = append(g.Adj[to], rev)
}
```

**Key Insight:** `RevIdx` is the critical bookkeeping. When you push flow on edge `g.Adj[from][i]`, the reverse edge is at `g.Adj[to][g.Adj[from][i].RevIdx]`. The two edges always point back at each other. Break this link and the algorithm silently produces wrong answers.

---

### Concept 4: Edmonds-Karp Algorithm (Finding Max Flow)

**What (Simple Definition)**
> Edmonds-Karp repeatedly finds the shortest available path from start to end (using BFS), pushes one unit of flow along it, and repeats until no path exists. The result is the maximum number of simultaneous paths.

**Why (Why This Matters)**
This is the engine that solves the problem. Without it, you'd have to try every possible combination of paths - which grows exponentially. Edmonds-Karp guarantees finding the optimal max flow in polynomial time.

Real-world analogy: you're planning parade routes through a city. You find the shortest available route, reserve it, then look for the next shortest route that doesn't conflict, and repeat. When no more routes exist, you've found the maximum number of non-conflicting parades.

**Where (Files)**
- `internal/solver/solver.go:23-69` - FindPaths (the main algorithm)
- `internal/solver/solver.go:72-95` - bfs (breadth-first search for augmenting path)
- `internal/solver/solver.go:98-108` - pushFlow (send flow along found path)
- `internal/solver/solver.go:111-150` - decomposePaths + traceOnePath (extract paths from flow)

**How (Step by Step)**

```
STEP 1: BFS finds shortest path with available capacity
         start → B_in → B_out → end
         (all edges have Cap=1, Flow=0, so Cap-Flow=1 > 0)

STEP 2: Push 1 unit of flow along that path
         start→B_in: Flow becomes 1 (Cap-Flow = 0, "full")
         B_in→B_out: Flow becomes 1
         B_out→end:  Flow becomes 1
         (Reverse edges: Flow becomes -1, so Cap-Flow = 0-(-1) = 1)

STEP 3: BFS again - can it find another path?
         start → C_in → C_out → end  ← YES! Different route
         Push flow again.

STEP 4: BFS again - any more paths?
         NO. All paths from start are full. Done!
         Max flow = 2 (two parallel paths found)

STEP 5: Decompose - extract the actual paths from the flow
         Follow positive-flow edges from start to end, twice.
         Path 1: [start, B, end]
         Path 2: [start, C, end]
```

---

### Concept 5: Optimal Path Count (Not Always Use All)

**What (Simple Definition)**
> Even though Edmonds-Karp finds the maximum number of parallel paths, using ALL of them isn't always fastest. The solver tests every subset size (1 path, 2 paths, ... N paths) and picks the one that minimizes total turns.

**Why (Why This Matters)**
Imagine you have 3 ants and find 3 paths: lengths 1, 1, and 10. Using all 3 paths means one ant takes 10 turns. Using just the two short paths means all ants finish in 2 turns. More paths ≠ fewer turns.

Real-world analogy: you're shipping 3 packages. Route A takes 1 day, Route B takes 1 day, Route C takes 10 days. Sending one package on each route means you're waiting 10 days. Better to send 2 on Route A and 1 on Route B - done in 2 days.

**Where (Files)**
- `internal/solver/solver.go:57-68` - Optimal subset selection loop
- `internal/solver/solver.go:152-179` - computeTurns (the formula)
- `internal/solver/distribute.go:14-67` - DistributeAnts (assignment)

**How (The Formula)**

For `k` paths sorted by ascending length L₁ ≤ L₂ ≤ ... ≤ Lₖ, with N ants:

```
sumDiff = (Lₖ - L₁) + (Lₖ - L₂) + ... + (Lₖ - Lₖ)
remaining = N - sumDiff
T = Lₖ - 1 + ceil(remaining / k)
```

Each path i gets: `aᵢ = T - Lᵢ + 1` ants

**Why this works:** The formula equalizes finish times. If path i has length Lᵢ, and we assign aᵢ ants to it, the last ant on that path finishes at turn `aᵢ + Lᵢ - 1`. Setting `aᵢ = T - Lᵢ + 1` makes every path finish at exactly turn T.

---

### Concept 6: Turn-by-Turn Simulation (The Output Machine)

**What (Simple Definition)**
> The simulator takes the paths and ant assignments, then produces a line of text for each "turn" showing which ants move where, like a play-by-play sports commentary.

**Why (Why This Matters)**
The solver tells us WHICH paths each ant takes and HOW MANY ants go on each. But the actual output format requires showing every individual movement: "L1-roomA L2-roomB" for each turn. The simulator "acts out" the solution.

Real-world analogy: the solver is like a chess engine that says "this is the winning sequence." The simulator is the person who physically moves the pieces on the board, one move at a time, and announces each one.

**Where (Files)**
- `internal/simulator/simulator.go:12-16` - antState struct
- `internal/simulator/simulator.go:20-116` - Simulate function

**How (The Algorithm)**

Each turn does three things in order:
1. **Advance** all active ants one step forward on their path
2. **Launch** one new ant per path (if any remain to launch)
3. **Format** all movements as `L<id>-<room>` sorted by ant ID

```
Turn 1: Launch ant 1 on path 0 → moves to first room after start
Turn 2: Advance ant 1 → next room. Launch ant 2 on path 0.
Turn 3: Advance ant 1 → end (remove). Advance ant 2 → next. Launch ant 3.
...until all ants reach end
```

**Key Insight:** Ants on a path form a "train" - one enters per turn, each advances one step per turn. They never collide because the path is vertex-disjoint and only one new ant enters per turn.

---

## How the 6 Concepts Connect

```
Input File ──→ [COLONY] ──→ [NODE-SPLIT GRAPH] ──→ [EDMONDS-KARP]
                  (1)              (2,3)                  (4)
                                                          │
                                                          ▼
Output ◄── [SIMULATION] ◄── [ANT ASSIGNMENT] ◄── [OPTIMAL SUBSET]
               (6)                (5)                    (5)
```

1. **Colony** structures the raw input
2. **Node-splitting** makes rooms into edge-capacity constraints
3. **Residual graph** enables the flow algorithm to undo bad choices
4. **Edmonds-Karp** finds all possible parallel paths
5. **Optimal selection** picks the best subset and assigns ants
6. **Simulation** generates the human-readable output

Each concept builds on the previous. You can't understand node-splitting without understanding the Colony. You can't understand Edmonds-Karp without understanding the residual graph. And the simulation only makes sense once you know how ants are assigned.

---

## Next: [02 - Data Flow](02-data-flow.md) - Watch the data transform step by step
