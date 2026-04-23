# `cmd/lem-in/main.go`

Purpose: solver CLI entrypoint for `go run ./cmd/lem-in` and `go build ./cmd/lem-in`.

Execution Flow:
- `main` reads `os.Args[1]`, calls `parser.Parse`, `graph.BuildGraph`, `solver.FindPaths`, `solver.DistributeAnts`, and `simulator.Simulate`, then prints `colony.Lines`, a blank separator, and each simulated turn.
- The printed separator and move format are the contract consumed later by `internal/format.ParseOutput` and both visualizers.

## `main`
- Purpose: provide the solver binary under `cmd/lem-in`.
- Behavior: validates that a filename argument exists, aborts on parser or solver errors, and preserves the exact original input text before printing the generated moves.
- Dependencies: `os.Args`, `parser.Parse`, `graph.BuildGraph`, `solver.FindPaths`, `solver.DistributeAnts`, `simulator.Simulate`, `strings.Join`.
- Called from: no Go call sites inside the repo. Invoked by the Go runtime when `cmd/lem-in` is built or run.
- Change impact: any change to output shape breaks `internal/format.ParseOutput`, `cmd/visualizer-tui`, and `cmd/visualizer-web`. Any change to pipeline order changes solver semantics.

# `internal/parser/types.go`

Purpose: define the parsed colony data model shared by parsing, graph construction, and final output reproduction.

Module notes:
- `Colony.Lines` stores the original file verbatim enough for the solver entrypoints to reprint the exact input section expected by the visualizers.

## `Room`
- Purpose: store a room's external identity and coordinates as parsed from the input file.
- Behavior: acts as the canonical room payload before graph node-splitting happens.
- Dependencies: filled by `parseRoom` in `internal/parser/parser.go`.
- Called from: used structurally by `Colony.Rooms`, `graph.BuildGraph`, and parser validation logic.
- Change impact: renaming fields or changing coordinate meaning breaks graph construction and any consumer expecting original room coordinates.

## `Colony`
- Purpose: aggregate all validated input needed by the solver pipeline.
- Behavior: keeps ant count, ordered room list, room-name lookup table, tunnel list, start and end room names, and original input lines.
- Dependencies: populated by `Parse` and `parseLines`.
- Called from: consumed by `graph.BuildGraph` and `cmd/lem-in/main.go`; `Lines` is reprinted by the solver CLI entrypoint.
- Change impact: this is the boundary between parsing and everything else. Field shape changes ripple through the entire pipeline.

# `internal/parser/parser.go`

Purpose: convert a lem-in text file into a validated `Colony` while enforcing the project's input grammar and most user-facing error messages.

Audit Notes:
- Parser error strings are effectively part of the CLI contract.
- Blank lines are skipped during parsing but still preserved in `Colony.Lines`, so parsing behavior and printed reproduction are intentionally not the same thing.

## `Parse`
- Purpose: top-level parser API for reading a file from disk and normalizing line endings before syntax validation.
- Behavior: reads the file, canonicalizes CRLF to LF, trims trailing newlines, rejects empty content as an invalid ant-count file, splits into lines, and delegates to `parseLines`.
- Dependencies: `os.ReadFile`, `strings.ReplaceAll`, `strings.TrimRight`, `strings.Split`, `parseLines`.
- Called from: directly observed in `cmd/lem-in/main.go`; no other production call sites.
- Change impact: changes here affect filesystem error handling and the exact input handed to the state machine below.

## `parseLines`
- Purpose: implement the actual lem-in grammar as a single pass state machine over normalized lines.
- Behavior: parses ant count first, tracks pending `##start` and `##end` markers, rejects malformed command placement, distinguishes room lines from link lines, forbids duplicate rooms and duplicate normalized links, validates known-room references, and ensures one distinct start and end exist.
- Dependencies: `strconv.Atoi`, `strings` helpers, `isLink`, `parseRoom`, `Colony.RoomMap`, local state flags such as `pendingStart`, `pendingEnd`, `inLinks`, and `linkSet`.
- Called from: directly observed only from `Parse`.
- Change impact: this is the highest-blast-radius validation point. Any loosened rule can feed illegal state into graph construction; any tightened rule can reject maps that the rest of the pipeline would otherwise handle.

## `isLink`
- Purpose: distinguish tunnel syntax from room syntax before the parser commits to room parsing.
- Behavior: accepts one `name1-name2` shape with non-empty sides and no spaces on either side of the dash.
- Dependencies: `strings.Index`, `strings.Contains`.
- Called from: directly observed only inside `parseLines`.
- Change impact: if it misclassifies a line, the parser either rejects valid rooms or accepts malformed links and later fails in less precise ways.

## `parseRoom`
- Purpose: parse one room declaration and enforce room-name and coordinate syntax that the rest of the system assumes.
- Behavior: requires exactly three fields, rejects names containing `-`, parses non-negative integer coordinates, and returns a `Room`.
- Dependencies: `strings.Fields`, `strconv.Atoi`.
- Called from: directly observed only inside `parseLines`.
- Change impact: this defines the allowable room namespace and coordinate domain. Relaxing it affects duplicate checks, visualizer layout assumptions, and parser error messages.

# `internal/graph/graph.go`

Purpose: convert a validated colony into the residual flow-network representation used by the solver.

Concept hook:
- The key abstraction is node splitting: intermediate rooms become `room_in -> room_out` with capacity 1 so vertex capacity becomes edge capacity.

## `BuildGraph`
- Purpose: build the capacity graph that encodes room occupancy rules as a max-flow problem.
- Behavior: assigns one node ID to start and end, two node IDs to each intermediate room, adds capacity-1 internal edges for split rooms, then adds bidirectional tunnel edges between each room's outbound and inbound side.
- Dependencies: `parser.Colony`, `Graph.addEdge`, `Graph.outNode`, `Graph.inNode`.
- Called from: directly observed in `cmd/lem-in/main.go`.
- Change impact: this is where problem semantics become graph semantics. A mistake here silently yields wrong path sets even when parsing and simulation remain unchanged.

## `(*Graph).outNode`
- Purpose: resolve the node ID that should emit flow for a room name.
- Behavior: returns the unsplit node for start/end and the `_out` node for intermediate rooms.
- Dependencies: `Graph.NameToID`, `parser.Colony`.
- Called from: directly observed inside `BuildGraph`.
- Change impact: if start/end are treated like split nodes or vice versa, tunnel wiring becomes invalid.

## `(*Graph).inNode`
- Purpose: resolve the node ID that should receive flow for a room name.
- Behavior: returns the unsplit node for start/end and the `_in` node for intermediate rooms.
- Dependencies: `Graph.NameToID`, `parser.Colony`.
- Called from: directly observed inside `BuildGraph`.
- Change impact: paired with `outNode`; mismatches break the node-splitting invariant.

## `(*Graph).addEdge`
- Purpose: add one forward residual edge and its reverse edge in adjacency-list form.
- Behavior: stores reverse-edge indexes so later flow updates can jump directly to the paired reverse edge.
- Dependencies: `Graph.Adj`, `Edge.RevIdx`.
- Called from: directly observed inside `BuildGraph`.
- Change impact: solver functions assume these reverse indexes are correct. If they drift, `pushFlow` and `traceOnePath` corrupt the residual graph.

## `OriginalName`
- Purpose: map internal node-splitting labels back to the external room names used in final paths.
- Behavior: strips `_in` and `_out` suffixes when present, otherwise returns the input unchanged.
- Dependencies: string slicing only.
- Called from: directly observed in `internal/solver/solver.go` while converting decomposed graph paths into public `Path` values.
- Change impact: wrong normalization leaks internal graph names into solver output and breaks simulation assumptions.

# `internal/solver/solver.go`

Purpose: find vertex-disjoint start-to-end paths with max flow, then choose the subset that minimizes total completion turns for the given ant count.

Audit Notes:
- `FindPaths` mutates `g.Adj[*].Flow` twice: first to push max flow, then to consume that flow during path decomposition. The graph is not reusable as an untouched residual graph after this package finishes.

## `Path.Length`
- Purpose: expose path cost in solver terms as edge count, not room count.
- Behavior: returns `len(Rooms) - 1`.
- Dependencies: `Path.Rooms`.
- Called from: directly observed in `FindPaths`, `computeTurns`, and `DistributeAnts`.
- Change impact: all path ranking and turn formulas depend on this exact definition.

## `FindPaths`
- Purpose: top-level solver API that turns a flow graph into the best set of usable paths for the requested ant count.
- Behavior: repeatedly runs `bfs` plus `pushFlow` until no augmenting path remains, decomposes the resulting positive flow into raw node-ID paths, converts them back to external room names, sorts by ascending path length, and keeps only the prefix that minimizes `computeTurns`.
- Dependencies: `bfs`, `pushFlow`, `decomposePaths`, `graph.OriginalName`, `Path.Length`, `computeTurns`, `sort.Slice`.
- Called from: directly observed in `cmd/lem-in/main.go`.
- Change impact: this is the solver's core correctness boundary. Any mistake can produce too few paths, wrong paths, or a suboptimal subset.

## `bfs`
- Purpose: find one augmenting path in the residual graph.
- Behavior: breadth-first search over residual edges with positive remaining capacity, recording parent node and edge index so the path can be replayed later.
- Dependencies: `Graph.Adj`, `Graph.StartID`, `Graph.EndID`.
- Called from: directly observed only inside `FindPaths`.
- Change impact: this enforces Edmonds-Karp's shortest-augmenting-path behavior. A change here alters both correctness and performance.

## `pushFlow`
- Purpose: apply one unit of flow along the augmenting path returned by `bfs`.
- Behavior: walks backward from sink to source, increments forward-edge flow, decrements reverse-edge flow, and relies on `RevIdx` to find the paired edge.
- Dependencies: `Graph.Adj`, parent data from `bfs`.
- Called from: directly observed only inside `FindPaths`.
- Change impact: if forward and reverse updates stop being exact inverses, later BFS passes and path decomposition operate on corrupted residual state.

## `decomposePaths`
- Purpose: consume all positive flow paths after max flow is complete.
- Behavior: repeatedly calls `traceOnePath` until no more positive-flow source-to-sink path can be found.
- Dependencies: `traceOnePath`.
- Called from: directly observed only inside `FindPaths`.
- Change impact: this bridges residual-flow state to the solver's public path representation. If it misses or duplicates a path, ant distribution is wrong.

## `traceOnePath`
- Purpose: extract one concrete source-to-sink path from the current positive-flow graph.
- Behavior: greedily follows forward edges with positive flow and positive capacity, marks nodes visited to avoid loops, and decrements consumed flow so the same path is not returned twice.
- Dependencies: `Graph.Adj`, `Graph.StartID`, `Graph.EndID`.
- Called from: directly observed only from `decomposePaths`.
- Change impact: because it mutates flow while traversing, mistakes here can erase remaining valid paths.

## `computeTurns`
- Purpose: evaluate how many turns a given prefix of sorted paths would need for a fixed ant count.
- Behavior: applies the balancing formula `T = Lk - 1 + ceil((N - sumDiff) / k)` using the longest path in the chosen prefix as the leveling baseline.
- Dependencies: `Path.Length`, `ceilDiv`, `math.MaxInt64`.
- Called from: directly observed in `FindPaths` and `DistributeAnts`.
- Change impact: this is the optimization metric. A bug here still returns valid-looking paths but can make assignment suboptimal.

## `ceilDiv`
- Purpose: integer helper for the turn formula.
- Behavior: returns ceiling division for positive integers.
- Dependencies: none beyond integer arithmetic.
- Called from: directly observed only in `computeTurns`.
- Change impact: small arithmetic changes here propagate directly into path-selection and distribution decisions.

# `internal/solver/distribute.go`

Purpose: assign individual ant IDs across the chosen paths so the simulator can launch ants in path order and still achieve the solver's target completion time.

## `DistributeAnts`
- Purpose: convert path-level optimality into concrete per-ant assignments.
- Behavior: re-evaluates the best usable prefix of the provided sorted paths, computes how many ants each selected path should carry, removes rounding excess from the longest used paths, then emits `AntAssignment` values in increasing ant-ID order with shorter paths filled first.
- Dependencies: `computeTurns`, `Path.Length`, `math.MaxInt64`.
- Called from: directly observed in `cmd/lem-in/main.go`.
- Change impact: if assignment order or counts change, `simulator.Simulate` still runs but the emitted move schedule can take more turns than necessary.

# `internal/simulator/simulator.go`

Purpose: transform path assignments into the lem-in move transcript printed by the CLI.

## `Simulate`
- Purpose: generate one output line per turn in the `L<ant>-<room>` format.
- Behavior: groups ants by path, advances all active ants one step per turn, launches at most one new ant per path each turn, sorts same-turn moves by ant ID, and emits the final move lines until no further moves remain.
- Dependencies: `solver.Path`, `solver.AntAssignment`, `sort.Slice`, `strings.Builder`, `fmt.Fprintf`.
- Called from: directly observed in `cmd/lem-in/main.go`.
- Change impact: this is the user-visible transcript contract. Any formatting change breaks `internal/format.ParseOutput` and both visualizers.

# `internal/format/format.go`

Purpose: parse the solver's stdout format back into structured data for the visualizers.

Audit Notes:
- This package intentionally trusts the solver transcript shape more than the original parser does. It is a consumer-side parser, not a full validator.

## `ParseOutput`
- Purpose: split solver stdout into original file content, metadata, and per-turn movements.
- Behavior: normalizes newlines, short-circuits `ERROR:` output into `ParsedOutput.Error`, finds the required blank separator line, parses the leading file-content section into ant count, rooms, links, start/end markers, then parses movement tokens after the separator.
- Dependencies: `strings` helpers, `strconv.Atoi`, `Movement`, `ParsedRoom`, `ParsedOutput`.
- Called from: directly observed in `cmd/visualizer-tui/main.go` and `cmd/visualizer-web/main.go`.
- Change impact: this is the compatibility layer between the solver CLI and both visualizers. If solver output changes, this file must change with it.

# `cmd/visualizer-tui/main.go`

Purpose: terminal visualizer that reads solver stdout from stdin, parses it with `internal/format`, and renders an interactive animated map in an alternate terminal buffer.

Module notes:
- This file mixes terminal portability, layout, animation state, and input handling in one package.
- `ttyReader`, `sttyOriginal`, and cleanup ordering are high leverage because they guard against terminal corruption and blocked reads on MINGW64.

Execution Flow:
- `main` enters the alternate screen, reads all stdin, parses with `format.ParseOutput`, opens a direct TTY for keyboard control, creates `playback`, `antState`, and `renderer`, then drives a render loop via keyboard events plus a 30fps ticker.
- Forward and backward motion are not recomputed from geometry each frame. `computeTransition` snapshots one turn into `animEntry` paths, and the render loop interpolates along those paths.

## `moveTo`
- Purpose: build an ANSI cursor-position escape sequence.
- Behavior: returns `ESC[row;colH]`.
- Dependencies: `fmt.Sprintf`, `esc`.
- Called from: directly observed in `showLoading`, `showCenteredError`, `renderer.render`, and `renderer.renderPanel`.
- Change impact: wrong cursor movement garbles the whole screen compositor.

## `getTerminalSize`
- Purpose: detect terminal dimensions across bash, Unix, and Windows-like environments.
- Behavior: checks `$COLUMNS/$LINES`, then `stty size`, then `tput`, then falls back to `80x24`.
- Dependencies: `envInt`, `tputInt`, `exec.Command`, `strings.Fields`.
- Called from: directly observed in `showCenteredError` and `main`.
- Change impact: layout errors here distort coordinate scaling and panel placement.

## `envInt`
- Purpose: parse an integer environment variable with fallback.
- Behavior: returns the parsed value or the provided default.
- Dependencies: `os.Getenv`, `strconv.Atoi`.
- Called from: directly observed only in `getTerminalSize`.
- Change impact: narrow helper; mainly affects terminal sizing heuristics.

## `tputInt`
- Purpose: query a numeric terminal capability from `tput`.
- Behavior: shells out to `tput <cap>` and parses the trimmed output.
- Dependencies: `exec.Command`, `strconv.Atoi`.
- Called from: directly observed only in `getTerminalSize`.
- Change impact: affects portability fallback behavior.

## `runStty`
- Purpose: run `stty` against its own `/dev/tty` handle so keyboard reads do not contend with mode changes.
- Behavior: opens `/dev/tty`, attaches it to an `stty` subprocess, and returns that command's output.
- Dependencies: `os.OpenFile`, `exec.Command`.
- Called from: directly observed in `enableRawMode` and `disableRawMode`.
- Change impact: if this shares the same descriptor as `readKeys`, terminal restore can hang.

## `enableRawMode`
- Purpose: switch the terminal into raw keyboard mode.
- Behavior: snapshots current `stty` settings and then enables raw, no-echo input.
- Dependencies: `runStty`, module-level `sttyOriginal`.
- Called from: directly observed in `showCenteredError` and `main`.
- Change impact: broken raw mode means arrow keys and single-key controls stop working.

## `disableRawMode`
- Purpose: restore terminal settings without risking an indefinite block.
- Behavior: attempts restore in a goroutine and gives up after 100ms if the PTY stays blocked.
- Dependencies: `runStty`, `sttyOriginal`, `time.After`.
- Called from: directly observed in `showCenteredError` and in `main` cleanup paths.
- Change impact: this protects the user's shell from staying in raw mode after exit.

## `(*screenBuf).write`, `(*screenBuf).writef`, `(*screenBuf).flush`
- Purpose: serialize many small render operations into one buffered stdout flush.
- Behavior: append formatted output under a mutex and then write it once to stdout.
- Dependencies: `sync.Mutex`, `strings.Builder`, `os.Stdout`.
- Called from: `writef` is observed from `renderer.render` and `renderer.renderPanel`; `flush` is observed from `renderer.render`; `write` underpins both.
- Change impact: these helpers are the anti-flicker boundary for the renderer.

## `newCanvas`, `(*canvas).set`, `(*canvas).setStr`, `(*canvas).get`
- Purpose: maintain an off-screen text/color grid before rendering it to the terminal.
- Behavior: allocate a blank cell matrix, set one cell, set a string across cells, or read a cell with bounds checks.
- Dependencies: `cell`, canvas dimensions.
- Called from: `newCanvas` is observed in `renderer.render`; `set` is observed from `setStr`, `dirGrid.applyToCanvas`, and room drawing; `setStr` is observed in `renderer.render` and `renderer.drawAntLabel`; `get` appears unused inside the repo.
- Change impact: coordinate safety and write behavior determine whether complex tunnel and label drawing corrupts adjacent cells.

## `scaleCoords`
- Purpose: map parsed room coordinates into terminal canvas coordinates with margins.
- Behavior: computes the bounding box, normalizes into the drawable area, clamps to stay away from edges, and returns one screen position per room name.
- Dependencies: `format.ParsedRoom`, `math.Round`.
- Called from: directly observed in `newRenderer`.
- Change impact: every room, tunnel, and ant position depends on this projection.

## `boxChar`
- Purpose: translate directional tunnel flags into a Unicode box-drawing rune.
- Behavior: emits straight, corner, tee, or crossing characters based on four directional bits.
- Dependencies: `dirFlags`.
- Called from: directly observed in `(*dirGrid).applyToCanvas`.
- Change impact: visual tunnel topology becomes unreadable if the flag-to-glyph mapping is wrong.

## `newDirGrid`, `(*dirGrid).addFlag`, `(*dirGrid).tracePath`, `(*dirGrid).applyToCanvas`
- Purpose: draw orthogonal tunnel paths that merge cleanly at intersections.
- Behavior: allocate the flag grid, OR direction bits into cells, trace an L-shaped horizontal-then-vertical route between two rooms, and finally emit box characters onto the canvas.
- Dependencies: `boxChar`, `canvas.set`.
- Called from: `newDirGrid` and the other methods are directly observed inside `renderer.render`.
- Change impact: these functions define the TUI's topology rendering; route shape changes also affect perceived ant motion because `renderer.computePath` matches this geometry.

## `abs`
- Purpose: integer absolute-value helper.
- Behavior: returns the non-negative magnitude of `x`.
- Dependencies: none.
- Called from: appears unused inside the repo.
- Change impact: none today unless later code starts using it.

## `newPlayback`, `(*playback).faster`, `(*playback).slower`, `(*playback).animDuration`
- Purpose: initialize and mutate the shared playback state machine.
- Behavior: create default autoplay settings, clamp speed adjustments, and convert playback speed into animation frame count.
- Dependencies: `playback` fields.
- Called from: `newPlayback` is observed in `main`; `faster` and `slower` are observed in `main` key handling; `animDuration` is observed in `startTransitionForward` and `startTransitionBackward`.
- Change impact: these functions control pacing and minimum animation smoothness.

## `newAntState`, `(*antState).applyTurn`, `(*antState).clone`, `(*antState).antsAtRoom`
- Purpose: track where every ant currently is for static rendering and transition snapshots.
- Behavior: initialize all ants at the start room, apply one parsed turn, deep-clone the position map, and list sorted ants at a room.
- Dependencies: parsed move data from `internal/format`.
- Called from: `newAntState` is observed in `main` and `recomputeAnts`; `applyTurn` in `main` transition closures; `clone` in both transition-start closures; `antsAtRoom` in `renderer.render`.
- Change impact: every rendered ant label and reverse-animation path depends on this state being exact.

## `newRenderer`, `(*renderer).computePath`, `(*renderer).computeTransition`, `(*renderer).render`, `(*renderer).drawAntLabel`, `(*renderer).renderPanel`
- Purpose: own all terminal drawing and animation-path projection logic.
- Behavior: `newRenderer` derives canvas dimensions and scaled room positions; `computePath` mirrors tunnel routing with an L-shaped path; `computeTransition` converts one turn into per-ant screen paths; `render` composes tunnels, rooms, ants, and the bottom panel; `drawAntLabel` packs room occupants into a compact label; `renderPanel` writes status, legend, and control help.
- Dependencies: `scaleCoords`, `newCanvas`, `newDirGrid`, `moveTo`, `screenBuf`, `antState`, `playback`, parsed rooms and links.
- Called from: `newRenderer` is observed in `main`; `computePath` is observed in `computeTransition` and the local `startTransitionBackward` closure; `computeTransition` is observed in `startTransitionForward`; `render` is observed throughout `main`; `drawAntLabel` and `renderPanel` are observed from `render`.
- Change impact: this is the TUI's visual contract. `computePath` must stay aligned with tunnel drawing or animated ants diverge from the map.

## `openTTY`
- Purpose: obtain a keyboard input handle even when stdin is already consumed by a pipe.
- Behavior: tries `/dev/tty`, then `CONIN$`, then falls back to `os.Stdin`.
- Dependencies: `os.OpenFile`.
- Called from: directly observed in `showCenteredError` and `main`.
- Change impact: failure here makes the TUI read solver output but not accept controls.

## `readKeys`
- Purpose: turn raw terminal bytes into higher-level input events without blocking shutdown forever.
- Behavior: launches a goroutine per read, decodes single-key commands and arrow-key escape sequences, and stops promptly when `done` closes.
- Dependencies: `ttyReader`, `time.After`, `keyEvent` constants.
- Called from: directly observed only in `main` as a goroutine.
- Change impact: this is the input side of the TUI state machine. Bugs cause hung exits, missed keys, or accidental repeats.

## `showLoading`
- Purpose: draw a minimal loading screen while stdin is being consumed.
- Behavior: clears the alternate buffer and prints a centered-ish status message.
- Dependencies: ANSI helpers and `moveTo`.
- Called from: directly observed only in `main`.
- Change impact: cosmetic only, unless screen state setup regresses.

## `showCenteredError`
- Purpose: display solver-side errors in the full-screen UI instead of dumping raw text into the alternate buffer.
- Behavior: centers an error message, waits for a keypress in raw mode, then restores the terminal.
- Dependencies: `getTerminalSize`, `enableRawMode`, `openTTY`, `disableRawMode`.
- Called from: directly observed only in `main` when `format.ParseOutput` returns `ParsedOutput.Error`.
- Change impact: this is the only user-friendly error path once the TUI has entered alternate-screen mode.

## `main`
- Purpose: own the TUI lifecycle, including terminal setup, parsed-data loading, playback state, input loop, and cleanup.
- Behavior: enters alternate screen, installs SIGINT cleanup, reads solver stdout, parses it, initializes rendering state, defines local transition helpers (`recomputeAnts`, `finishAnimation`, `reverseAnimation`, `startTransitionForward`, `startTransitionBackward`), then runs a select loop over keyboard events and a frame ticker.
- Dependencies: nearly every helper in this file plus `format.ParseOutput`.
- Called from: no Go call sites inside the repo. Invoked by the Go runtime for `cmd/visualizer-tui`.
- Change impact: this is the TUI control root. Changes here affect replay behavior, direction reversal, cleanup safety, and interactive semantics.

## Local closures inside `main`
- `recomputeAnts`: rebuilds state up to an arbitrary turn by replaying parsed moves; called from `startTransitionBackward`.
- `finishAnimation`: drops transient animation state without altering the already-snapped `ants`; called from key handlers and the ticker loop.
- `reverseAnimation`: flips the current transition mid-flight by swapping pre/post ant snapshots and reversing entry paths; called from Left/Right handling when the user changes direction during an animation.
- `startTransitionForward`: snapshots current ant state, computes the next turn's paths, applies the turn to `ants`, and starts a forward animation; called from autoplay, slider stepping, step mode, and replay restart.
- `startTransitionBackward`: reconstructs the previous-state snapshot, reverses the current turn's path list, rewinds `turnIdx`, and starts a backward animation; called from Left-arrow handling.

# `cmd/visualizer-web/main.go`

Purpose: web visualizer generator that reads solver stdout, converts it into JSON, embeds a compressed 3D model, and emits a self-contained HTML document with Three.js-based playback.

Module notes:
- `base.glb` is embedded via `//go:embed` and compressed to base64-gzipped payload before insertion into the HTML.
- This file has two execution layers: Go builds the HTML shell, then the browser executes the embedded JavaScript runtime.

Execution Flow:
- Go-side `main` reads stdin, parses with `format.ParseOutput`, builds normalized visualization JSON with `buildJSONData`, compresses the embedded model, then passes both into `buildHTML`.
- Browser-side `animate` is the control root after page load: it advances turns, updates DOM controls, updates orbit controls, and renders the Three.js scene every frame.

## `main`
- Purpose: CLI wrapper that turns solver stdout into one standalone HTML page.
- Behavior: reads stdin, parses the solver transcript, JSON-encodes normalized visualization data, gzip-compresses the embedded `.glb`, base64-encodes it, and prints the HTML.
- Dependencies: `format.ParseOutput`, `buildJSONData`, `buildHTML`, `gzip`, `base64`, embedded `colonyModel`.
- Called from: no Go call sites inside the repo. Invoked by the Go runtime for `cmd/visualizer-web`.
- Change impact: changes here affect the generator contract and whether the page remains self-contained.

## `bfsDepth`
- Purpose: assign each room a graph depth from the start room for 3D layout purposes.
- Behavior: builds an undirected adjacency map from the parsed links, BFSes from the start, and assigns depth 0 to unreachable rooms.
- Dependencies: parsed rooms and links.
- Called from: directly observed only in `buildJSONData`.
- Change impact: this determines vertical layering in the web scene. Layout changes do not affect solver correctness but do affect readability.

## `buildJSONData`
- Purpose: convert parsed solver output into the simplified JSON shape consumed by the browser runtime.
- Behavior: preserves ant count and terminal error state, groups rooms by BFS depth, lays each depth level out on a radial ring with slight deterministic name-based offset, copies links, and rewrites turns into JS-friendly movement objects.
- Dependencies: `bfsDepth`, parsed output from `internal/format`, `math` ring-layout helpers.
- Called from: directly observed only in `main`.
- Change impact: this is the browser contract boundary. Any field-name or layout-rule changes require matching updates inside the generated JS.

## `buildHTML`
- Purpose: emit the complete HTML, CSS, import map, inline data payload, and browser-side animation runtime.
- Behavior: writes the page chrome, injects `SIM_DATA` and the compressed model payload, imports Three.js modules, defines the browser-side helper functions, installs UI event listeners, and boots the animation loop.
- Dependencies: `jsonStr`, `modelB64`, embedded JavaScript runtime.
- Called from: directly observed only in `main`.
- Change impact: this is the highest-blast-radius file for the web visualizer. It owns both page appearance and nearly all runtime behavior.

## Embedded browser-side helpers inside `buildHTML`
- `hash3`, `noise3d`, `fbm`: procedural-noise helpers used to perturb tunnel curves and organic motion midpoints. Called from tunnel generation and ant path interpolation.
- `loadColonyModel`: browser-side async loader that base64-decodes, gunzips, and parses the embedded GLB. Called once at startup; if it fails, the page cannot render the colony geometry.
- `getOrCreateAnt`: lazily allocate one Three.js mesh per ant ID and remember its current room. Called from `animateAnts`, `snapAnts`, and `rebuildAntsToTurn`.
- `buildTurnAnims`: precompute per-turn ant motions from `turns` and `roomMap`. Called once during startup to populate `turnAnimations`.
- `getAnimPos`: compute a smoothed curve position between two rooms for a normalized progress value. Called from `animateAnts`.
- `animateAnts`: place and orient ant meshes for the in-progress turn. Called from `animate` while a turn is mid-flight.
- `snapAnts`: finalize one turn by placing ants directly at their destination room. Called from `animate` when turn progress reaches 1.
- `hideAllAnts`: hide every spawned ant mesh. Called from `rebuildAntsToTurn` and the restart handler.
- `rebuildAntsToTurn`: reconstruct ant visibility and placement for an arbitrary timeline position. Called from Prev/Next buttons and the timeline slider.
- `hideFinishedAnts`: hide ants whose final known room is the end room. Called after timeline jumps to the end and when autoplay completes.
- `animate`: browser-side main loop. Called first at startup and then via `requestAnimationFrame`; advances turn progress, updates UI text and sliders, steps orbit controls, and renders the scene.

Audit Notes:
- `buildHTML` contains several large inline anonymous callbacks for hover and UI controls. They are not exported functions, but they are part of the runtime control flow and must stay in sync with the JSON field names produced by `buildJSONData`.
- `colonyMat` and `baseMat` are defined but never applied; they look like leftovers from an earlier material pass.
