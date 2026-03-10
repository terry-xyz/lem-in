# Lesson 05: Exercises

## How to Use This Chapter

Work through these exercises **without AI assistance**. That's the whole point - you're training yourself to navigate, understand, and modify code independently.

**Rules:**
- Use only your text editor and `go` commands
- No AI chat, no copilot, no auto-complete explanations
- It's OK to re-read the lesson files
- If you get stuck for 15+ minutes, check the hints at the bottom

---

## Level 0: Warmup (Absolute Beginners)

> **Skills:** Basic file navigation, reading code structure

### Exercise 0.1: Find a File
**Task:** List all the `.go` files in the `internal/` directory (recursively). How many are there?

**How to check:** Run `find internal/ -name "*.go"` or use your editor's file tree.

### Exercise 0.2: Read a Struct
**Task:** Open `internal/parser/types.go`. The `Colony` struct has 7 fields. List all their names and types from memory after reading the file once.

### Exercise 0.3: Count Functions
**Task:** Open `internal/solver/solver.go`. List every function name in the file. Don't count methods (functions attached to a type).

**Expected:** You should find 6 functions: `FindPaths`, `bfs`, `pushFlow`, `decomposePaths`, `traceOnePath`, `computeTurns`, plus `ceilDiv`.

### Exercise 0.4: Understand a Comment
**Task:** Find a comment in the codebase that explains **WHY** something is done (not just WHAT). Copy it here. Then find a comment that only says WHAT.

**Hint:** Look at `graph.go` line 24-25 for a "why" comment.

### Exercise 0.5: Trace an Import
**Task:** Open `cmd/lem-in/main.go`. It imports 4 packages. For each import:
1. Is it from the Go standard library or from this project?
2. What does it provide? (one sentence each)

---

## Level 1: Find (Read the Code)

> **Skills:** Locating specific code, understanding definitions, navigating between files

### Exercise 1.1: Find the Constant
**Task:** There's a constant `maxAnts` in the parser. What is its value? Why does this limit exist?

### Exercise 1.2: Follow the Type Chain
**Task:** In `simulator.go`, the `antState` struct has a field `PathIndex`. Trace what this field refers to:
1. Where is `PathIndex` set? (file and line)
2. What does it index into?
3. What type is at that index?
4. What fields does THAT type have?

### Exercise 1.3: Find the Error Messages
**Task:** The parser can produce many different error messages. List at least 8 distinct error messages by reading `parser.go`. For each one, describe in one sentence what triggers it.

### Exercise 1.4: Map the Node IDs
**Task:** For `example00.txt` (rooms: 0, 2, 3, 1 where 0=start, 1=end), manually compute the node IDs that `BuildGraph` would assign. Write them in a table:

| Room | Internal Name(s) | Node ID(s) |
|------|-------------------|-------------|
| 0    | ?                 | ?           |
| 2    | ?                 | ?           |
| 3    | ?                 | ?           |
| 1    | ?                 | ?           |

**Check:** Run the program and add print statements to verify.

### Exercise 1.5: Spot the Unused Return Value
**Task:** In `cmd/lem-in/main.go`, line 36 has `_, assignments := solver.DistributeAnts(...)`. What is the first return value that's being discarded? What type is it? When would someone want to use it?

---

## Level 2: Trace (Follow the Data)

> **Skills:** Data flow, mental execution, predicting behavior
>
> This is the MOST important skill. If you can trace data through code in your head, you can debug anything.

### Exercise 2.1: Trace a Simple Input
**Task:** Create a file `test_trace.txt`:
```
2
##start
A 0 0
##end
B 1 1
A-B
```
Before running the program, predict:
1. How many nodes will the graph have?
2. How many edges (including reverse edges)?
3. How many paths will Edmonds-Karp find?
4. How many turns in the output?
5. What will each output line be?

Then run the program and check your predictions.

### Exercise 2.2: Trace BFS
**Task:** For this network (3 ants):
```
start → A → B → end
start → C → end
```
Manually run BFS from start:
1. What's in the queue at each step?
2. What path does BFS find first?
3. After pushing flow on that path, what's in the queue on the second BFS?
4. What path does the second BFS find?

### Exercise 2.3: Trace the Turn Formula
**Task:** You have 5 ants and two paths: length 2 and length 4.
1. Calculate `sumDiff`
2. Calculate `remaining`
3. Calculate `T` (total turns)
4. Calculate ants per path (`a₁` and `a₂`)
5. Verify: does `a₁ + a₂ = 5`?
6. Verify: does `a₁ + L₁ - 1 ≤ T` and `a₂ + L₂ - 1 ≤ T`?

### Exercise 2.4: Predict the Subset Choice
**Task:** You have 3 ants and three paths: lengths 1, 1, and 10.
1. Calculate turns for k=1 (use only first path)
2. Calculate turns for k=2 (use first two paths)
3. Calculate turns for k=3 (use all three)
4. Which k gives the fewest turns?
5. Why doesn't using all 3 paths help?

### Exercise 2.5: Trace the Simulator
**Task:** Given:
- Path 0: `[start, A, end]` (length 2)
- Path 1: `[start, B, C, end]` (length 3)
- Assignments: Ant 1 → Path 0, Ant 2 → Path 0, Ant 3 → Path 1

Write out each turn manually:
- What's in `active` before and after each step?
- What moves are generated?
- What's the output line?

Do this for all turns until all ants reach end.

---

## Level 3: Modify (Small Changes)

> **Skills:** Targeted changes, understanding ripple effects, testing

### Exercise 3.1: Add a Room Count
**Task:** Modify `cmd/lem-in/main.go` to print `# Rooms: N` (where N is the room count) as a comment line before the move lines. The output should still be valid (comments start with `#`).

**Test:** Run with `example00.txt` and verify the comment appears.

### Exercise 3.2: Count the Turns
**Task:** Modify `cmd/lem-in/main.go` to print `# Turns: N` after the last move line. The number should be the length of `moveLines`.

**Test:** Run with `example00.txt` (expect 6 turns) and `example01.txt` (expect 8 turns or fewer).

### Exercise 3.3: Reverse the Ant Order
**Task:** In `simulator.go`, moves are sorted by ascending ant ID (line 100-102). Change it to descending order. Run the program and observe the output. Then change it back.

**Question:** Does the descending order violate the spec? Why or why not?

### Exercise 3.4: Add a Room Name Validator
**Task:** The parser rejects rooms starting with `L` or `#`. Add a rule that also rejects rooms starting with a digit. Then:
1. Write the validation code
2. Run with `example00.txt` - what happens? (Room "0" starts with a digit!)
3. Realize why this rule would be problematic
4. Revert your change

**Lesson:** This shows why you should understand the data before adding validation.

### Exercise 3.5: Add a Verbose Flag
**Task:** Add a `-v` flag to `cmd/lem-in/main.go` that, when present, prints the discovered paths before the output. Example:
```
# Path 0 (length 3): start → A → B → end
# Path 1 (length 2): start → C → end
```

**Hint:** Check `os.Args` for `-v` before the filename.

---

## Level 4: Extend (Add Features)

> **Skills:** Planning additions, maintaining consistency, architecture awareness

### Exercise 4.1: Path Statistics
**Task:** Create a new function `PrintStats(paths []solver.Path, antCount int)` in a new file `cmd/lem-in/stats.go` that prints:
- Number of paths found
- Shortest and longest path lengths
- Total turns
- Ants per path breakdown

Call it from `main()` when a `-stats` flag is provided.

### Exercise 4.2: Validate Output
**Task:** Write a function that takes the move lines and verifies:
1. No two ants are in the same intermediate room on the same turn
2. All ants start at start and end at end
3. Ants only move to adjacent rooms

**Where to put it:** `internal/simulator/validate.go`

### Exercise 4.3: JSON Output Mode
**Task:** Add a `-json` flag that outputs the result as JSON instead of the default text format. The JSON should contain:
```json
{
  "ant_count": 4,
  "paths": [["start", "A", "B", "end"]],
  "turns": [
    [{"ant": 1, "room": "A"}],
    [{"ant": 1, "room": "B"}, {"ant": 2, "room": "A"}]
  ]
}
```

### Exercise 4.4: Handle Disconnected Graphs
**Task:** Currently, if start and end aren't connected, the solver returns an error. Modify it to also print WHICH rooms are reachable from start. This helps debugging.

### Exercise 4.5: Support Weighted Tunnels
**Task:** (Advanced) Modify the parser to support an optional weight after links: `A-B 3` means the tunnel has capacity 3 instead of 1. This requires changes in parser, graph, and solver. Plan which files need changes before coding.

---

## Level 5: Break & Fix (Debugging)

> **Skills:** Understanding WHY code exists by seeing what happens without it

### Exercise 5.1: Remove Node-Splitting
**Task:** Temporarily modify `BuildGraph` to NOT split intermediate rooms (give them single nodes like start/end). Run with `example05.txt`. What goes wrong? Why?

**Expected:** The solver finds paths that share intermediate rooms, violating the "one ant per room" constraint.

### Exercise 5.2: Break Reverse Edges
**Task:** In `addEdge`, change the reverse edge capacity from 0 to 1. Run with `example01.txt`. Does the result change? Why?

**Expected:** The algorithm might find "wrong" augmenting paths by treating reverse edges as real tunnels.

### Exercise 5.3: Use DFS Instead of BFS
**Task:** Replace the BFS in `solver.go` with a DFS (use a stack instead of queue: pop from the END instead of the front). Run with `example01.txt`. Compare the number of turns with the original BFS version.

**Expected:** DFS might find longer augmenting paths, leading to a different (possibly worse) max flow decomposition.

### Exercise 5.4: Skip the Optimal Subset
**Task:** In `FindPaths`, comment out the Phase 3 loop and always return ALL paths. Create a test case where this gives more turns than the optimized version.

**Hint:** Try 3 ants with paths of length 1, 1, and 20.

### Exercise 5.5: Break the Turn Formula
**Task:** In `computeTurns`, change `lk - 1` to `lk` (remove the `- 1`). Run all tests with `go test -p 1 ./...`. Which tests fail? What does this tell you about the formula?

---

## The Vibecoding Graduation Test

Can you do these **WITHOUT AI assistance?**

1. **Pseudo-code first:** Write pseudo-code for adding a "max room capacity" feature where some rooms can hold 2 ants instead of 1.

2. **Predict files:** Which files would that feature touch? List them with a one-sentence explanation of what changes in each.

3. **Explain the why:** Why does `computeTurns` return `math.MaxInt64` when `remaining <= 0`? What scenario causes this, and what would happen if it returned 0 instead?

4. **Find the edge case:** What happens if the input file has exactly 1 ant and a direct link from start to end (no intermediate rooms)? Trace through all 4 pipeline stages.

5. **Rubber duck:** Explain the Edmonds-Karp algorithm out loud (to a rubber duck, a pet, a wall) in under 2 minutes. Cover: what problem it solves, how BFS helps, what reverse edges do, and when it stops.

**If you can do all five, you've graduated from vibecoding to understanding.**

---

## Hints (Don't Read Until You're Stuck)

<details>
<summary>Exercise 2.1 Hints</summary>

- A and B: start and end, so 2 nodes (no split). Direct link = 1 path of length 1.
- Special case in simulator: `len(path.Rooms) <= 1` handles this... but actually length is 1 (1 edge, 2 rooms: `[A, B]`). So `len(path.Rooms) = 2 > 1`, normal flow applies.
- 2 ants, 1 path of length 1: T = 1-1 + ceil(2/1) = 0 + 2 = 2 turns.
- Turn 1: L1-B (ant 1 goes straight to end)
- Turn 2: L2-B (ant 2 goes straight to end)
</details>

<details>
<summary>Exercise 2.3 Hints</summary>

- Paths sorted: [2, 4]. Lk = 4.
- sumDiff = (4-2) + (4-4) = 2 + 0 = 2
- remaining = 5 - 2 = 3
- T = 4-1 + ceil(3/2) = 3+2 = 5
- a₁ = 5-2+1 = 4, a₂ = 5-4+1 = 2, total = 6 > 5
- Excess = 1, remove from longest: a₂ becomes 1
- Final: a₁ = 4, a₂ = 1, total = 5
</details>

<details>
<summary>Exercise 2.4 Hints</summary>

- k=1: T = 1-1+ceil(3/1) = 0+3 = 3
- k=2: T = 1-1+ceil((3-0)/2) = 0+2 = 2
- k=3: T = 10-1+ceil((3-9)/3) = remaining = -6 → math.MaxInt64 (impossible!)
- k=2 wins with 2 turns. k=3 fails because sumDiff (9+9+0=18? no, Lk=10, sumDiff=(10-1)+(10-1)+(10-10)=18) exceeds antCount (3).
</details>

---

## Next: [06 - Gotchas](06-gotchas.md) - Tricky spots where bugs love to hide
