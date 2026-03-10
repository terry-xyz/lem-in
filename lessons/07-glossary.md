# Lesson 07: Glossary

## Start Here: Essential Terms

These terms appear constantly throughout the codebase. Master these first.

| Term | Plain English | Example in Code |
|------|---------------|-----------------|
| **Colony** | All data from the input file, bundled together | `parser.Colony` struct |
| **Room** | A named location where ants can be | `Room{Name: "A", X: 0, Y: 3}` |
| **Link/Tunnel** | A connection between two rooms | `[2]string{"A", "B"}` |
| **Path** | A sequence of rooms from start to end | `Path{Rooms: ["start","A","B","end"]}` |
| **Flow** | How many units of "stuff" move through an edge | `Edge.Flow = 1` |
| **Capacity** | Maximum flow an edge can carry | `Edge.Cap = 1` |
| **Turn** | One time step where every ant moves once | One line of output |
| **Assignment** | Which ant goes on which path | `AntAssignment{AntID:1, PathIndex:0}` |

---

## Domain Terms (The Problem)

These describe the problem being solved - the "ant farm" puzzle.

| Term | Plain English | Why It Matters |
|------|---------------|----------------|
| **Ant** | A numbered entity (1 to N) that must move from start to end | Ants are the "units" being routed |
| **Start room** | The room where all ants begin; unlimited capacity | Can hold and release many ants per turn |
| **End room** | The room where all ants must arrive; unlimited capacity | Can receive many ants per turn |
| **Intermediate room** | Any room that isn't start or end; holds max 1 ant | The constraint that makes the problem hard |
| **Ant count** | Number of ants to move (1 to 10,000,000) | Affects which paths are optimal |
| **Movement** | One ant moving to one room in one turn | Formatted as `L<id>-<room>` |
| **Turn count** | Total number of turns until all ants reach end | The metric we're minimizing |

---

## Code Terms (The Implementation)

These describe the data structures and algorithms used in the code.

| Term | Plain English | Where |
|------|---------------|-------|
| **Node-splitting** | Replacing one room with two nodes (`_in` + `_out`) to enforce capacity | `graph.go` |
| **Residual graph** | The graph with flow values and reverse edges | `Graph.Adj` after flow |
| **Augmenting path** | A path from start to end with available capacity | Found by `bfs()` |
| **Max flow** | The maximum number of parallel non-conflicting paths | Result of Edmonds-Karp |
| **Vertex-disjoint** | Paths that share no intermediate rooms | Guaranteed by node-splitting |
| **Edge-disjoint** | Paths that share no edges | Natural result of max flow |
| **BFS** | Breadth-First Search: explore nearest neighbors first | `solver.go:72` |
| **DFS** | Depth-First Search: explore as deep as possible first | Used in `traceOnePath` |
| **FIFO** | First In, First Out (queue behavior) | BFS queue |
| **Decompose** | Extract individual paths from aggregate flow values | `decomposePaths()` |
| **Optimal subset** | The number of paths (k) that minimizes total turns | Phase 3 of `FindPaths` |

---

## Variable Naming (Why Names Look Weird)

| Variable | What It Means | Where |
|----------|---------------|-------|
| `g` | The Graph (flow network) | Everywhere after `BuildGraph` |
| `c` | The Colony (parsed data) | Inside `parseLines` |
| `k` | Number of paths being considered | `computeTurns`, `DistributeAnts` |
| `T` | Optimal turn count | `DistributeAnts` |
| `lk` | Length of the k-th (longest) path | `computeTurns` |
| `Lk`, `Li` | Path lengths in the formula | Comments |
| `N` | Number of ants (in formula) | Comments |
| `ai` | Number of ants on path i | Comments |
| `fwd` | Forward edge | `addEdge` |
| `rev` | Reverse edge | `addEdge` |
| `curr` | Current node being processed | `bfs`, `traceOnePath` |
| `prev` | Previous node in the path | `pushFlow` |
| `sb` | strings.Builder (efficient string concatenation) | `simulator.go` |
| `idx` | Index (position in a slice) | Throughout |
| `sepIdx` | Separator index (blank line position) | `format.go` |

---

## Abbreviations Decoded

| Abbreviation | Full Form | Context |
|--------------|-----------|---------|
| `Adj` | Adjacency (list) | `Graph.Adj` - the list of edges for each node |
| `Cap` | Capacity | `Edge.Cap` - max flow through this edge |
| `Rev` | Reverse | `Edge.RevIdx` - index of the reverse edge |
| `ID` | Identifier | Node IDs (integers), Ant IDs (1-based) |
| `BFS` | Breadth-First Search | The queue-based graph traversal |
| `DFS` | Depth-First Search | The stack/recursion-based graph traversal |
| `TUI` | Text User Interface | Terminal-based visualizer |
| `ANSI` | American National Standards Institute | Terminal color/formatting codes |
| `GLB` | GL Binary | 3D model format used by web visualizer |
| `CDN` | Content Delivery Network | Where Three.js is loaded from |
| `FIFO` | First In, First Out | Queue behavior (BFS uses this) |

---

## File Types & Suffixes

| Suffix | Meaning | Example |
|--------|---------|---------|
| `_test.go` | Test file (Go convention) | `parser_test.go` |
| `_in` / `_out` | Split node suffixes | `room_in`, `room_out` |
| `.txt` | Input file for the solver | `example00.txt` |
| `.mod` | Go module definition | `go.mod` |
| `.glb` | 3D model binary (GL Binary) | `base.glb` |

---

## Magic Numbers (Why These Values?)

| Value | Where | Why |
|-------|-------|-----|
| `10_000_000` | `parser.go:10` (`maxAnts`) | Spec limit: prevents memory explosion from absurd input |
| `-1` | `bfs` parent init | Sentinel value meaning "not visited" |
| `0` | Reverse edge capacity | Starts with no flow, gains effective capacity when flow is pushed |
| `1` | Internal edge capacity | Enforces "one ant per room" constraint |
| `math.MaxInt64` | `computeTurns` fallback | "Infinity" - this subset is impossible (more slack than ants) |

---

## Function Naming Patterns

| Pattern | Meaning | Examples |
|---------|---------|---------|
| `Parse*` | Convert text → structured data | `Parse`, `ParseOutput`, `parseLines`, `parseRoom` |
| `Build*` | Construct a complex data structure | `BuildGraph` |
| `Find*` | Search for something | `FindPaths` |
| `Distribute*` | Assign resources to destinations | `DistributeAnts` |
| `Simulate` | Act out a plan step by step | `Simulate` |
| `compute*` | Calculate a value from inputs | `computeTurns` |
| `trace*` | Follow a path through a graph | `traceOnePath` |
| `is*` | Boolean check (returns true/false) | `isLink` |
| `*Node` | Returns a node ID | `outNode`, `inNode` |
| `Original*` | Reverse a transformation | `OriginalName` |
| `ceil*` | Round up (ceiling division) | `ceilDiv` |

---

## Error Messages Decoded

| Error Message | Translation | Fix |
|---------------|-------------|-----|
| `invalid number of ants` | First line isn't a positive integer, or exceeds 10M | Check the first line of your input file |
| `invalid room name` | Room name starts with `L`, `#`, contains `-`, or is empty | Rename the room |
| `invalid coordinates` | Room coordinates aren't non-negative integers | Fix the X Y values |
| `duplicate room` | Two rooms have the same name | Rename one of them |
| `duplicate link` | Same tunnel listed twice (A-B and B-A count as same) | Remove the duplicate |
| `link to unknown room` | A link references a room that wasn't defined | Define the room or fix the name |
| `self-link` | A tunnel from a room to itself (A-A) | Remove it |
| `duplicate start/end command` | `##start` or `##end` appears more than once | Remove the duplicate command |
| `invalid command placement` | `##start`/`##end` not followed by a room, or two commands in a row | Add a room after the command |
| `no start/end room found` | Missing `##start` or `##end` in input | Add the command |
| `start and end are the same room` | `##start` and `##end` point to the same room | Use different rooms |
| `no path from start to end` | Start and end aren't connected through any sequence of tunnels | Add tunnels to create a path |
| `cannot read file` | File doesn't exist or no permission | Check the file path |
| `invalid data` | Non-room, non-link line after the links section | Clean up the input file |

---

## Glossary Complete!

**You now have:**
- Essential vocabulary for every concept in the codebase
- Decoder rings for abbreviations and variable names
- Quick reference for error messages
- Understanding of magic numbers and naming conventions

**Congratulations!** You're ready to:
1. Navigate the codebase with confidence
2. Understand WHY code is written the way it is
3. Debug issues by tracing data flow
4. Communicate using shared vocabulary
5. Read new Go code without looking up every syntax element

**The journey from "vibecoding" to understanding is complete.**

---

## Quick Reference Card

```
Pipeline:    Parse → BuildGraph → FindPaths → DistributeAnts → Simulate
Key Files:   parser.go → graph.go → solver.go → distribute.go → simulator.go
Formula:     T = Lk - 1 + ceil((N - sumDiff) / k)
Ants/Path:   ai = T - Li + 1
Node Split:  room → room_in ---(cap 1)--→ room_out
Edge Pair:   forward (cap 1, flow 0) + reverse (cap 0, flow 0)
BFS Check:   Cap - Flow > 0  (remaining capacity)
Output:      L<antID>-<roomName>  (sorted by ant ID per turn)
```
