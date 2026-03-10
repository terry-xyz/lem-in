# Lesson 06: Gotchas

## Why This Chapter Exists

Every codebase has "gotchas" - things that are obvious AFTER you know them, but can waste hours if you don't. Each gotcha below represents a **real bug** someone could easily introduce. Understanding them makes you a better debugger and a more careful programmer.

---

### Gotcha 1: The Off-By-One in Path Length

**The bug:** `Path.Length()` returns `len(Rooms) - 1` (number of edges), NOT the number of rooms. Confusing the two breaks the turn formula.

```go
// WRONG: treating Length() as room count
turns := path.Length()  // Path ["A","B","C"] → Length() = 2, NOT 3

// RIGHT: understanding Length() = edges = rooms - 1
rooms := path.Length() + 1  // 2 + 1 = 3 rooms
```

**Where it matters:**
- `solver.go:156-178` - `computeTurns` uses Length() as edge count
- `distribute.go:32` - `T - paths[i].Length() + 1` would be wrong without the `+ 1`

**How to spot this bug:** Simulation produces one too many or one too few turns. Ants either don't reach the end or there's an empty final turn.

---

### Gotcha 2: The Formula Is `Lk - 1`, Not `Lk`

**The bug:** The turn count formula uses `Lk - 1 + ceil(remaining/k)`, NOT `Lk + ceil(...)`. Getting this wrong adds one extra turn to every solution.

```go
// WRONG (from a naive derivation)
return lk + ceilDiv(remaining, k)

// RIGHT (corrected derivation)
return lk - 1 + ceilDiv(remaining, k)
```

**Why `-1`?** Because an ant entering a path of length L on turn `t` arrives at turn `t + L - 1` (it takes L steps, but the first step happens on the entry turn itself, not the turn after). So if the last ant enters on turn `T - L + 1`, it finishes on turn `(T - L + 1) + L - 1 = T`.

**How to spot this bug:** All test cases have one extra turn. The audit examples fail by exactly 1.

---

### Gotcha 3: The Ant Formula Is `T - Li + 1`, Not `T - Li`

**The bug:** Each path gets `T - Li + 1` ants, not `T - Li`. Off by one in the other direction.

```go
// WRONG
count := T - paths[i].Length()

// RIGHT
count := T - paths[i].Length() + 1
```

**Why `+ 1`?** Derivation: ant `a_i` on path `i` finishes at turn `a_i + L_i - 1`. Set equal to T: `a_i = T - L_i + 1`. Without the `+ 1`, you assign one fewer ant per path, and some ants are never assigned.

**How to spot this bug:** `totalAssigned < antCount` after the assignment loop. Some ants are missing from the output.

---

### Gotcha 4: Reverse Edge Flow Goes Negative

**The bug:** New developers see `g.Adj[node][revIdx].Flow--` and think the flow shouldn't go below 0. They add a `max(0, ...)` guard that breaks the algorithm.

```go
// WRONG: "protecting" against negative flow
if g.Adj[node][revIdx].Flow > 0 {
    g.Adj[node][revIdx].Flow--
}

// RIGHT: negative flow is intentional and necessary
g.Adj[node][revIdx].Flow--  // Flow = -1 is correct!
// Cap=0, Flow=-1 → available = Cap-Flow = 0-(-1) = 1
// This "opens" the reverse path for future BFS passes
```

**Why negative?** The reverse edge has `Cap=0`. When flow is decremented to -1, the available capacity becomes `0 - (-1) = 1`. This phantom capacity is what allows BFS to find "undo" paths. Without it, Edmonds-Karp can't reroute flow and gets stuck in suboptimal solutions.

**How to spot this bug:** Max flow is lower than expected. Some test cases that need flow rerouting fail (networks with crossing paths).

---

### Gotcha 5: RevIdx Must Be Set Before Append

**The bug:** In `addEdge`, the `RevIdx` values are computed using `len(g.Adj[to])` BEFORE appending. If you rearrange the code to append first, the indices are off by one.

```go
// RIGHT: compute indices BEFORE appending
fwd := Edge{To: to, Cap: cap, RevIdx: len(g.Adj[to])}      // Reverse will be at current length
rev := Edge{To: from, Cap: 0, RevIdx: len(g.Adj[from])}    // Forward was just appended, so len-1? NO!
g.Adj[from] = append(g.Adj[from], fwd)  // NOW append
g.Adj[to] = append(g.Adj[to], rev)      // NOW append

// WRONG: append first, then compute
g.Adj[from] = append(g.Adj[from], Edge{...})
// Now len(g.Adj[from]) is ONE MORE than when we computed RevIdx for the reverse!
```

**How to spot this bug:** `pushFlow` panics with index out of range, or silently modifies the wrong edge. Flow values become inconsistent and paths don't decompose correctly.

---

### Gotcha 6: The `isLink` Function and Ambiguous Dashes

**The bug:** Room names can't contain dashes (parser rejects them), but this isn't obvious. If you allowed dashed room names, `isLink("my-room 3 4")` would incorrectly match as a link `"my"` to `"room 3 4"`.

```go
func isLink(line string) bool {
    dashIdx := strings.Index(line, "-")
    if dashIdx <= 0 || dashIdx >= len(line)-1 {
        return false
    }
    left := line[:dashIdx]
    right := line[dashIdx+1:]
    // KEY: if either side has spaces, it's NOT a link (it's a room definition)
    if strings.Contains(left, " ") || strings.Contains(right, " ") {
        return false
    }
    return true
}
```

**The defense:** The parser rejects room names with dashes (`parseRoom` checks `strings.Contains(name, "-")`). But `isLink` also checks for spaces as a safety net. Remove either check and ambiguous input could be misparsed.

**How to spot this bug:** A room with a dash in its name gets parsed as a link, causing "unknown room" errors for the parts on each side of the dash.

---

### Gotcha 7: Pending Command + Pending Command

**The bug:** What if the input has `##start` followed by `##end` with no room in between? Both flags would be pending, and the next room would only satisfy one.

```go
// The protection:
if pendingStart || pendingEnd {
    return nil, fmt.Errorf("ERROR: ... invalid command placement")
}
pendingEnd = true  // Only set if nothing else is pending
```

**Without this check:**
```
##start
##end        ← pendingStart is still true! This would set pendingEnd too.
myRoom 0 0   ← Would this be start or end? BOTH? Neither?
```

**How to spot this bug:** A room is marked as both start and end, causing the `StartName == EndName` check to trigger unexpectedly (or worse, not triggering if the check is also missing).

---

### Gotcha 8: Node-Splitting Exemption for Start/End

**The bug:** If you accidentally split the start or end room into `start_in`/`start_out`, ants can only enter/exit one at a time. With 100 ants and 5 paths, only 1 ant could leave per turn instead of 5.

```go
// The protection: start and end are NOT split
if room.Name == colony.StartName || room.Name == colony.EndName {
    g.NameToID[room.Name] = nodeID  // Single node, unlimited capacity
    nodeID++
} else {
    // Only intermediate rooms get split
    nodeID += 2
}
```

**How to spot this bug:** The number of turns is much higher than expected, especially with many paths. Profiling shows a max flow of 1 (bottleneck at start/end internal edge).

---

### Gotcha 9: Ceiling Division Trick

**The bug:** Integer division in Go truncates toward zero. `7 / 3 = 2`, not 2.33. If you need to round UP (ceiling), you must use the trick: `(a + b - 1) / b`.

```go
// WRONG: integer division rounds down
turns := remaining / k  // 7/3 = 2, but we need 3

// RIGHT: ceiling division
turns := (remaining + k - 1) / k  // (7+2)/3 = 9/3 = 3 ✓
```

**Why this works:** Adding `b - 1` before dividing "pushes" any remainder over the threshold. If `a` is exactly divisible by `b`, the `b - 1` addition doesn't change the result: `(6 + 2) / 3 = 8 / 3 = 2` (since `6/3 = 2` exactly). But if there's a remainder: `(7 + 2) / 3 = 9 / 3 = 3`.

**How to spot this bug:** Off-by-one turn count, but only for ant counts that don't divide evenly across paths.

---

### Gotcha 10: The `stillActive` Slice Swap

**The bug:** In the simulator, instead of deleting ants from the `active` slice (which requires shifting elements), the code builds a NEW `stillActive` slice each turn. If you tried to modify `active` in-place during iteration, you'd skip elements or panic.

```go
// WRONG: modifying slice while iterating
for i, ant := range active {
    if ant.StepIndex >= len(path.Rooms)-1 {
        active = append(active[:i], active[i+1:]...)  // BREAKS iteration!
    }
}

// RIGHT: build a new slice
var stillActive []*antState
for _, ant := range active {
    if ant.StepIndex < len(path.Rooms)-1 {
        stillActive = append(stillActive, ant)  // Keep only active ants
    }
}
active = stillActive  // Old slice is garbage-collected
```

**How to spot this bug:** Some ants are skipped or processed twice. Output has missing movements or duplicate movements. May also panic with index out of range.

---

### Gotcha 11: Windows Line Ending Contamination

**The bug:** On Windows, text files use `\r\n` line endings. If you forget to normalize, room names silently get `\r` appended: `"start\r"` != `"start"`. Map lookups fail, links to "unknown" rooms, mysterious crashes.

```go
// The protection (in both parser.go and format.go):
content = strings.ReplaceAll(content, "\r\n", "\n")
```

**How to spot this bug:** Everything works on Linux/Mac but fails on Windows. Error messages mention rooms that look correct but have invisible characters. Adding `fmt.Printf("%q", name)` reveals the `\r`.

---

### Gotcha 12: Link Duplicate Detection Order

**The bug:** Links are bidirectional: `A-B` and `B-A` are the same tunnel. If you only check `"A-B"` for duplicates, someone could add `"B-A"` as a separate link.

```go
// The protection: normalize order before storing
a, b := name1, name2
if a > b {
    a, b = b, a  // Always store alphabetically smaller name first
}
key := a + "-" + b  // "A-B" and "B-A" both become "A-B"
```

**How to spot this bug:** Duplicate tunnels in the graph double the capacity, allowing two ants through what should be a single-track tunnel.

---

## The Gotcha Mindset

When reading or writing code, always ask:

1. **"What could go wrong?"** - Edge cases, bad input, empty collections, integer overflow
2. **"What order do things happen?"** - Append before or after computing indices? Advance before or after launching?
3. **"What's the spec say?"** - Don't assume "standard" behavior. The formula has `-1`, not `+0`.
4. **"What would happen if...?"** - Empty input? Zero ants? One room? Direct start→end? All rooms in a line?

This defensive thinking separates robust code from fragile code. Every gotcha in this chapter was discovered by asking one of these four questions.

---

## Next: [07 - Glossary](07-glossary.md) - Every term and abbreviation decoded
