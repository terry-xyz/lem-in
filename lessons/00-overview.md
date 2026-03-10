# Lesson 00: Overview

## Your Goal

You've built (or inherited) a working **lem-in** project. Maybe an AI wrote it. Maybe you copied it. Maybe you wrote it at 3 AM and forgot everything. This guide will take you from "it works, I think?" to **"I understand every line and can change anything confidently."**

By the end of these lessons you'll be able to:
- Explain what every file does without looking at the code
- Trace data from input file to final output in your head
- Debug failures by reasoning about the algorithm
- Add features or fix bugs without AI assistance

---

## What is Code? (If You're New)

Think of code like a **recipe**. You have:
- **Ingredients** (data) - the numbers, text, and structures the program works with
- **Instructions** (logic) - the step-by-step operations that transform ingredients into a finished dish

Every program, no matter how complex, is built from just **5 building blocks**:

| Building Block   | Plain English                  | Go Example                     |
|------------------|--------------------------------|--------------------------------|
| **Variable**     | A labeled box that holds a value | `name := "Alice"`             |
| **Function**     | A reusable recipe              | `func add(a, b int) int`      |
| **Condition**    | A yes/no question              | `if x > 10 { ... }`           |
| **Loop**         | Repeat until done              | `for i := 0; i < 10; i++`    |
| **Data Structure** | An organized container       | `[]string`, `map[string]int`  |

That's it. Everything in this project - the graph algorithms, the flow networks, the visualizers - is just combinations of these five things.

---

## What Does lem-in Do? (One Sentence)

**lem-in finds the fastest way to move N ants from a start room to an end room through a network of tunnels, where each tunnel and intermediate room can hold only one ant at a time.**

Think of it like a **subway system**: you have stations (rooms) connected by single-track tunnels (links). You need to move a crowd of passengers from station A to station B as fast as possible, but each track segment only fits one train at a time, and each intermediate station only has one platform.

---

## High-Level Architecture

```
 +-----------+     +-----------+     +-----------+     +-----------+
 |  PARSER   | --> |   GRAPH   | --> |  SOLVER   | --> | SIMULATOR |
 | Read file |     | Build     |     | Find best |     | Generate  |
 | Validate  |     | flow      |     | paths &   |     | turn-by-  |
 | Extract   |     | network   |     | assign    |     | turn      |
 | data      |     | (split    |     | ants      |     | output    |
 |           |     |  nodes)   |     |           |     |           |
 +-----------+     +-----------+     +-----------+     +-----------+
       ^                                                     |
       |                                                     v
  example00.txt                                    L1-room L2-room ...
```

This is a **pipeline** - data flows in one direction, each stage transforms it, and the next stage picks up where the last one left off. Like a factory assembly line.

---

## Directory Map

```
lem-in/
├── main.go                          # Front door - delegates to cmd/lem-in
├── cmd/
│   ├── lem-in/main.go               # The solver pipeline (4 steps)
│   ├── visualizer-tui/main.go       # Terminal animation (bonus)
│   └── visualizer-web/main.go       # 3D browser visualization (bonus)
├── internal/
│   ├── parser/
│   │   ├── types.go                 # Data definitions (Room, Colony)
│   │   └── parser.go                # Reads + validates input files
│   ├── graph/
│   │   └── graph.go                 # Builds the flow network
│   ├── solver/
│   │   ├── solver.go                # Finds optimal paths (Edmonds-Karp)
│   │   └── distribute.go            # Assigns ants to paths
│   ├── simulator/
│   │   └── simulator.go             # Produces turn-by-turn output
│   └── format/
│       └── format.go                # Parses solver output (for visualizers)
├── examples/                        # Test input files
├── Makefile                         # Build/run shortcuts
└── go.mod                           # Go module definition
```

### What Each Directory Does (Plain English)

| Directory | Responsibility | Analogy |
|-----------|---------------|---------|
| `internal/parser/` | Reads the input file and checks it's valid | **Mail room** - opens the envelope, checks the address is real |
| `internal/graph/` | Converts rooms+tunnels into a math structure | **Architect** - draws the blueprint from the description |
| `internal/solver/` | Finds the best routes and assigns ants | **Traffic planner** - finds the fastest routes through the city |
| `internal/simulator/` | Generates the step-by-step output | **Narrator** - describes what happens each second |
| `internal/format/` | Reads solver output back into structured data | **Translator** - converts the story back into data for visualizers |
| `cmd/lem-in/` | Wires everything together | **Manager** - calls each department in order |
| `cmd/visualizer-tui/` | Shows an animated terminal display | **TV crew** - makes a live broadcast |
| `cmd/visualizer-web/` | Generates a 3D browser visualization | **Film crew** - makes a movie you watch in a browser |

---

## Entry Points

### Where the program starts:

1. **`main.go` (root)** - The front door. Identical to `cmd/lem-in/main.go`. Either can be used.
2. **`cmd/lem-in/main.go`** - The real entry point. Runs the 4-stage pipeline:
   ```
   Parse → BuildGraph → FindPaths + DistributeAnts → Simulate → Print
   ```

### Where the visualizers start:

3. **`cmd/visualizer-tui/main.go`** - Reads solver output from stdin, animates in terminal
4. **`cmd/visualizer-web/main.go`** - Reads solver output from stdin, generates an HTML file

### How to run:

```bash
# Just the solver
go run . examples/example00.txt

# Solver piped to TUI visualizer
go run ./cmd/lem-in examples/example00.txt | go run ./cmd/visualizer-tui

# Solver piped to web visualizer
go run ./cmd/lem-in examples/example00.txt | go run ./cmd/visualizer-web

# Using the Makefile
make run FILE=examples/example00.txt
make viz-tui FILE=examples/example01.txt
```

---

## Input/Output at a Glance

### Input (example00.txt):
```
4               ← Number of ants
##start         ← Next line is the start room
0 0 3           ← Room "0" at coordinates (0,3)
2 2 5           ← Room "2" at coordinates (2,5)
3 4 0           ← Room "3" at coordinates (4,0)
##end           ← Next line is the end room
1 8 3           ← Room "1" at coordinates (8,3)
0-2             ← Tunnel connecting rooms "0" and "2"
2-3             ← Tunnel connecting rooms "2" and "3"
3-1             ← Tunnel connecting rooms "3" and "1"
```

### Output:
```
4               ← (Echo of the original input)
##start
0 0 3
2 2 5
3 4 0
##end
1 8 3
0-2
2-3
3-1
                ← (Blank separator line)
L1-2            ← Turn 1: Ant 1 moves to room 2
L1-3 L2-2      ← Turn 2: Ant 1 moves to room 3, Ant 2 enters room 2
L1-1 L2-3 L3-2 ← Turn 3: Ant 1 reaches end, Ant 2 to room 3, Ant 3 enters
L2-1 L3-3 L4-2 ← Turn 4: ...
L3-1 L4-3      ← Turn 5
L4-1            ← Turn 6: Last ant reaches end
```

---

## What's Next?

| Lesson | What You'll Learn |
|--------|-------------------|
| [01 - Core Concepts](01-core-concepts.md) | The 6 key ideas that make this project work |
| [02 - Data Flow](02-data-flow.md) | How data transforms from file to output |
| [03 - Patterns](03-patterns.md) | Recurring design patterns in the code |
| [04 - Line by Line](04-line-by-line.md) | Deep walkthrough of every important function |
| [05 - Exercises](05-exercises.md) | Practice tasks to test your understanding |
| [06 - Gotchas](06-gotchas.md) | Tricky spots where bugs love to hide |
| [07 - Glossary](07-glossary.md) | Every term and abbreviation decoded |
