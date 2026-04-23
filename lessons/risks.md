# Risks

## Output compatibility is fragile

- Files: `cmd/lem-in/main.go`, `internal/simulator/simulator.go`, `internal/format/format.go`
- Risk: the visualizers require the exact “original input, blank line, movement lines” structure.
- Inspect if changed: `cmd/lem-in/main.go`, `Simulate`, and `ParseOutput`.

## Parser rules encode most invariants

- Files: `internal/parser/parser.go`
- Risk: duplicate-room checks, duplicate-link checks, command-placement rules, and start/end validation all live here. Relaxing one rule can feed illegal state into graph construction.
- Inspect if changed: `parseLines`, `isLink`, `parseRoom`, and any tests around malformed examples.

## Graph construction defines problem semantics

- Files: `internal/graph/graph.go`
- Risk: node-splitting and tunnel wiring are the real encoding of the “one ant per intermediate room” rule. A small bug here produces plausible but wrong solutions.
- Inspect if changed: `BuildGraph`, `outNode`, `inNode`, `addEdge`.

## Solver mutates shared graph state

- Files: `internal/solver/solver.go`
- Risk: `FindPaths` both pushes max flow and consumes that flow during decomposition. Reusing the same graph after calling it is unsafe unless you rebuild it.
- Inspect if changed: `pushFlow`, `traceOnePath`, `decomposePaths`.

## Path optimality is split across two files

- Files: `internal/solver/solver.go`, `internal/solver/distribute.go`
- Risk: `FindPaths` chooses the best path prefix, and `DistributeAnts` recomputes the same optimization logic. If the formulas diverge later, the chosen path set and the assignment policy can disagree.
- Inspect if changed: `computeTurns`, `DistributeAnts`.

## TUI cleanup is platform-sensitive

- Files: `cmd/visualizer-tui/main.go`
- Risk: alternate-screen mode, raw mode, `/dev/tty` access, and the goroutine-per-read input loop are all there to avoid a stuck terminal on MINGW64. Simplifying this code casually can leave the shell unusable after exit.
- Inspect if changed: `runStty`, `enableRawMode`, `disableRawMode`, `openTTY`, `readKeys`, `main` cleanup.

## TUI geometry has two coupled implementations

- Files: `cmd/visualizer-tui/main.go`
- Risk: tunnels are drawn with `dirGrid.tracePath`, and ant motion uses `renderer.computePath`. They must stay shape-compatible or ants visibly travel off-tunnel.
- Inspect if changed: `tracePath`, `computePath`, `computeTransition`.

## Web visualizer has one large blast-radius file

- Files: `cmd/visualizer-web/main.go`
- Risk: data normalization, HTML generation, CSS, embedded JS runtime, and Three.js scene logic all live in one file. A small edit can break generation, page boot, controls, or animation.
- Inspect if changed: `buildJSONData`, `buildHTML`, embedded JS helpers, slider and button handlers.

## Embedded asset delivery is runtime-critical

- Files: `cmd/visualizer-web/main.go`, `cmd/visualizer-web/base.glb`
- Risk: the page only works because the embedded model is gzip-compressed in Go and decompressed in the browser. Either side can break the other.
- Inspect if changed: Go-side gzip/base64 steps in `main`, browser-side `loadColonyModel`.

## Solver CLI has one contract surface

- Files: `cmd/lem-in/main.go`
- Risk: the solver entrypoint, usage text, and transcript shape are now centralized in one file, so behavior changes should be reviewed there together with the visualizer parser.
- Inspect if changed: `cmd/lem-in/main.go`, especially output formatting and error handling.
