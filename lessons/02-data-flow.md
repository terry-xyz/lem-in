# Lesson 02: Data Flow

## The Factory Assembly Line

Think of this program like a factory with four workstations:

1. **Receiving Dock** (Parser) - Raw materials arrive (a text file). Workers inspect them, reject anything damaged, and organize them into labeled bins.
2. **Blueprint Room** (Graph) - Engineers take the organized bins and draw a precise engineering blueprint with extra detail the original description didn't have.
3. **Strategy Office** (Solver) - Planners study the blueprint and figure out the fastest routes. They decide which workers (ants) take which route.
4. **Dispatch Floor** (Simulator) - Workers follow the plan, and a narrator writes down exactly what happens each second.

Data flows in ONE direction: `File → Colony → Graph → Paths + Assignments → Output Lines`

---

## How to Read These Diagrams

- **Boxes** `[...]` = data containers (structs, variables)
- **Arrows** `→` = data flows / transformations
- **Labels on arrows** = the function doing the transformation
- **Dashed boxes** `- - -` = intermediate state (exists briefly)

---

## The Full Pipeline (Sequence Diagram)

```
  example00.txt         Parser              Graph            Solver           Simulator
       │                  │                   │                │                 │
       │  Read file       │                   │                │                 │
       │─────────────────>│                   │                │                 │
       │                  │                   │                │                 │
       │                  │ Parse(filename)   │                │                 │
       │                  │──────┐            │                │                 │
       │                  │      │ validate   │                │                 │
       │                  │      │ rooms,     │                │                 │
       │                  │      │ links,     │                │                 │
       │                  │      │ commands   │                │                 │
       │                  │<─────┘            │                │                 │
       │                  │                   │                │                 │
       │                  │   Colony struct   │                │                 │
       │                  │──────────────────>│                │                 │
       │                  │                   │                │                 │
       │                  │                   │ BuildGraph()   │                 │
       │                  │                   │──────┐         │                 │
       │                  │                   │      │ split   │                 │
       │                  │                   │      │ nodes,  │                 │
       │                  │                   │      │ add     │                 │
       │                  │                   │      │ edges   │                 │
       │                  │                   │<─────┘         │                 │
       │                  │                   │                │                 │
       │                  │                   │  Graph struct  │                 │
       │                  │                   │───────────────>│                 │
       │                  │                   │                │                 │
       │                  │                   │                │ FindPaths()     │
       │                  │                   │                │──────┐          │
       │                  │                   │                │      │ BFS +   │
       │                  │                   │                │      │ push    │
       │                  │                   │                │      │ flow    │
       │                  │                   │                │<─────┘          │
       │                  │                   │                │                 │
       │                  │                   │                │ DistributeAnts()│
       │                  │                   │                │──────┐          │
       │                  │                   │                │      │ assign  │
       │                  │                   │                │      │ ants    │
       │                  │                   │                │<─────┘          │
       │                  │                   │                │                 │
       │                  │                   │                │ []Path +        │
       │                  │                   │                │ []AntAssignment │
       │                  │                   │                │────────────────>│
       │                  │                   │                │                 │
       │                  │                   │                │                 │ Simulate()
       │                  │                   │                │                 │──────┐
       │                  │                   │                │                 │      │
       │                  │                   │                │                 │<─────┘
       │                  │                   │                │                 │
       │                  │                   │                │          []string (moves)
```

---

## The Story in Plain English

Let's follow `example00.txt` (4 ants, rooms 0→2→3→1) through the entire pipeline:

1. **First**, the Parser reads the file and extracts: "4 ants, 4 rooms, 3 tunnels, room '0' is start, room '1' is end."
2. **Then**, the Graph Builder splits intermediate rooms 2 and 3 into `2_in/2_out` and `3_in/3_out`, creating a 6-node flow network.
3. **Next**, the Solver runs Edmonds-Karp BFS to find one path (`0→2→3→1`), determines 1 path is optimal for 4 ants, and assigns all 4 ants to that path.
4. **Finally**, the Simulator walks each ant through the path one step per turn, generating 6 lines of output.

---

## Data Transformations (Step by Step)

### STEP 1: Raw Input → Colony

**Input:** A text file

```
4
##start
0 0 3
2 2 5
3 4 0
##end
1 8 3
0-2
2-3
3-1
```

**After parsing:** A Colony struct

```
Colony {
    AntCount:  4
    Rooms:     [{Name:"0", X:0, Y:3},     ← start room
                {Name:"2", X:2, Y:5},
                {Name:"3", X:4, Y:0},
                {Name:"1", X:8, Y:3}]     ← end room
    RoomMap:   {"0":0, "2":1, "3":2, "1":3}
    Links:     [["0","2"], ["2","3"], ["3","1"]]
    StartName: "0"
    EndName:   "1"
    Lines:     ["4", "##start", "0 0 3", ...]  ← original text preserved
}
```

**What changed:** Unstructured text → organized data with validation.

---

### STEP 2: Colony → Graph (Flow Network)

**Input:** Colony with 4 rooms, 3 links

**After BuildGraph:** A Graph with 6 nodes and many edges

```
Node IDs:
  0: "0"      (start - NOT split)
  1: "2_in"   (intermediate - split)
  2: "2_out"
  3: "3_in"   (intermediate - split)
  4: "3_out"
  5: "1"      (end - NOT split)

Edges (→ = forward, ⟵ = reverse/residual):
  Internal edges (enforce room capacity):
    1 →(cap 1)→ 2    (2_in → 2_out)
    2 ⟵(cap 0)⟵ 1    (reverse)
    3 →(cap 1)→ 4    (3_in → 3_out)
    4 ⟵(cap 0)⟵ 3    (reverse)

  Tunnel edges (bidirectional tunnels):
    0 →(cap 1)→ 1    (start → 2_in)     link: 0-2
    2 →(cap 1)→ 0    (2_out → start)     link: 0-2 (reverse direction)
    2 →(cap 1)→ 3    (2_out → 3_in)      link: 2-3
    4 →(cap 1)→ 1    (3_out → 2_in)      link: 2-3 (reverse direction)
    4 →(cap 1)→ 5    (3_out → end)        link: 3-1
    5 →(cap 1)→ 3    (end → 3_in)        link: 3-1 (reverse direction)
    (plus all reverse/residual edges with cap 0)
```

**Visual representation:**

```
                    cap 1          cap 1          cap 1
  [0:start] ──────→ [1:2_in] ────→ [2:2_out] ────→ [3:3_in]
                         ↑                              │
                    cap 1│                         cap 1│
                         │                              ↓
                         └── [4:3_out] ←──────── [4:3_out] ──→ [5:end]
```

Simplified:
```
  start → 2_in →(1)→ 2_out → 3_in →(1)→ 3_out → end
```

**What changed:** Room names → numeric IDs. Intermediate rooms split. Reverse edges added.

---

### STEP 3: Graph → Paths (via Edmonds-Karp)

**Input:** 6-node graph with residual edges

**During Edmonds-Karp:**

```
Iteration 1:
  BFS from node 0 (start):
    Queue: [0]
    Visit 0 → edges to [1(2_in)]: cap 1, flow 0 → available!
    Queue: [1]
    Visit 1 → edges to [2(2_out)]: cap 1, flow 0 → available!
    Queue: [2]
    Visit 2 → edges to [3(3_in)]: cap 1, flow 0 → available!
    Queue: [3]
    Visit 3 → edges to [4(3_out)]: cap 1, flow 0 → available!
    Queue: [4]
    Visit 4 → edges to [5(end)]: cap 1, flow 0 → available!
    FOUND END! Path: 0 → 1 → 2 → 3 → 4 → 5

  Push flow along path:
    Edge 0→1: flow 0→1 (now full)
    Edge 1→2: flow 0→1
    Edge 2→3: flow 0→1
    Edge 3→4: flow 0→1
    Edge 4→5: flow 0→1
    (All reverse edges: flow 0→-1)

Iteration 2:
  BFS from node 0 (start):
    Visit 0 → edge to 1: cap 1, flow 1 → cap-flow = 0 → BLOCKED
    No other edges from start. NO PATH FOUND.
    Max flow complete. Total flow = 1.

Decompose paths:
  Follow positive-flow edges from start:
    0→1 (flow 1) → 1→2 (flow 1) → 2→3 (flow 1) → 3→4 (flow 1) → 4→5 (flow 1)
  Raw path: [0, 1, 2, 3, 4, 5]
  Convert to names: ["0", "2_in", "2_out", "3_in", "3_out", "1"]
  Deduplicate: ["0", "2", "3", "1"]

Optimal subset selection:
  k=1: T = 3-1 + ceil((4-0)/1) = 2+4 = 6 turns ← only option, use it
```

**After Edmonds-Karp:**

```
paths = [
    Path{Rooms: ["0", "2", "3", "1"]}   ← length 3
]
```

**What changed:** Graph with flow → human-readable room-name paths, optimally selected.

---

### STEP 4: Paths → Ant Assignments

**Input:** 1 path of length 3, 4 ants

**During DistributeAnts:**

```
k=1 path, T=6 turns
Path 0 (length 3): gets T - 3 + 1 = 4 ants

antsPerPath = [4]
assignments = [
    {AntID: 1, PathIndex: 0},
    {AntID: 2, PathIndex: 0},
    {AntID: 3, PathIndex: 0},
    {AntID: 4, PathIndex: 0},
]
```

**What changed:** Paths + ant count → specific ant-to-path assignments.

---

### STEP 5: Paths + Assignments → Turn Output

**Input:** 1 path `["0","2","3","1"]`, 4 ants all on path 0

**During Simulate:**

```
pathAnts = [[1, 2, 3, 4]]   ← all 4 ants on path 0
nextAnt  = [0]               ← index of next ant to launch on path 0

TURN 1:
  Advance active: (none active yet)
  Launch: ant 1 on path 0, step 1 → room "2"
    active = [ant{ID:1, path:0, step:1}]
  Output: "L1-2"

TURN 2:
  Advance: ant 1: step 1→2 → room "3" (not at end, keep)
    active = [ant{ID:1, path:0, step:2}]
  Launch: ant 2 on path 0, step 1 → room "2"
    active = [ant{ID:1, path:0, step:2}, ant{ID:2, path:0, step:1}]
  Output: "L1-3 L2-2"

TURN 3:
  Advance: ant 1: step 2→3 → room "1" (= end! REMOVE)
  Advance: ant 2: step 1→2 → room "3"
    active = [ant{ID:2, path:0, step:2}]
  Launch: ant 3 on path 0, step 1 → room "2"
    active = [ant{ID:2, path:0, step:2}, ant{ID:3, path:0, step:1}]
  Output: "L1-1 L2-3 L3-2"

TURN 4:
  Advance: ant 2: step 2→3 → room "1" (end, remove)
  Advance: ant 3: step 1→2 → room "3"
    active = [ant{ID:3, path:0, step:2}]
  Launch: ant 4 on path 0, step 1 → room "2"
    active = [ant{ID:3, path:0, step:2}, ant{ID:4, path:0, step:1}]
  Output: "L2-1 L3-3 L4-2"

TURN 5:
  Advance: ant 3: step 2→3 → room "1" (end, remove)
  Advance: ant 4: step 1→2 → room "3"
    active = [ant{ID:4, path:0, step:2}]
  Launch: no more ants
  Output: "L3-1 L4-3"

TURN 6:
  Advance: ant 4: step 2→3 → room "1" (end, remove)
    active = []
  Launch: no more ants
  Output: "L4-1"

TURN 7:
  Advance: (none)
  Launch: (none)
  No moves → STOP
```

**Final output:**

```
["L1-2", "L1-3 L2-2", "L1-1 L2-3 L3-2", "L2-1 L3-3 L4-2", "L3-1 L4-3", "L4-1"]
```

---

## Multi-Path Example (example01.txt)

For a more interesting example, `example01.txt` has 10 ants and a complex network with multiple possible routes. Here's what happens differently:

```
STEP 1: Parse → Colony with 15 rooms, 15+ links
STEP 2: BuildGraph → ~28 nodes (13 intermediate rooms × 2 + start + end)
STEP 3: Edmonds-Karp finds 3 vertex-disjoint paths:
    Path 0: start → h → A → c → k → end   (length 5)
    Path 1: start → 0 → o → n → e → end   (length 5)
    Path 2: start → t → E → a → m → end   (length 5)

STEP 4: Optimal subset: k=3 (all 3 paths, since they're equal length)
    T = 5 - 1 + ceil((10 - 0) / 3) = 4 + 4 = 8 turns
    Path 0: 8 - 5 + 1 = 4 ants → adjusted to 3
    Path 1: 8 - 5 + 1 = 4 ants → adjusted to 4
    Path 2: 8 - 5 + 1 = 4 ants → adjusted to 3

STEP 5: Simulate → 8 turns of output with 3 ants moving per turn
```

---

## The Visualizer Data Flow

The visualizers consume the solver's output. They have their own pipeline:

```
Solver stdout ──→ format.ParseOutput() ──→ ParsedOutput struct ──→ Render
                       │                         │
                  Parse text into              Contains:
                  structured data              - Rooms with coords
                                               - Links
                                               - Turn-by-turn movements
                                               - Start/End names

TUI: ParsedOutput → scaleCoords() → canvas rendering → ANSI terminal
Web: ParsedOutput → bfsDepth() → 3D coordinates → HTML/Three.js file
```

---

## Summary: What Each Stage Produces

| Stage | Input | Output | Key Transformation |
|-------|-------|--------|-------------------|
| Parse | Text file | Colony | Text → structured data with validation |
| BuildGraph | Colony | Graph | Rooms → split nodes + residual edges |
| FindPaths | Graph | []Path | Max-flow → vertex-disjoint paths |
| DistributeAnts | Paths, AntCount | Assignments | Paths → per-ant route assignments |
| Simulate | Paths, Assignments | []string | Assignments → turn-by-turn text |
| (Print) | Colony.Lines + moves | stdout | Combine original input + moves |

---

## Next: [03 - Patterns](03-patterns.md) - Recurring design patterns in the code
