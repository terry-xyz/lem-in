# Concepts

## Solver transcript as an API

The CLI output is not just user-facing text. It is the interchange format for both visualizers.

- Defined by: `main.go`, `cmd/lem-in/main.go`, `internal/simulator/simulator.go`
- Consumed by: `internal/format/format.go`, `cmd/visualizer-tui/main.go`, `cmd/visualizer-web/main.go`
- Why it matters: the blank line between original input and movement lines is structural, not cosmetic. If output formatting changes, the visualizers stop parsing.

## Node splitting

Intermediate rooms are represented as two graph nodes, `room_in` and `room_out`, with a capacity-1 edge between them.

- Implemented in: `internal/graph/graph.go`
- Consumed by: `internal/solver/solver.go`
- Why it matters: this is how the code encodes the lem-in rule that only one ant may occupy an intermediate room at a time. If you miss this, the flow graph looks overcomplicated for no reason and solver changes become dangerous.

## Two-stage optimization

The solver does not stop after finding all disjoint paths. It then chooses the best prefix of those paths for the actual ant count.

- Implemented in: `internal/solver/solver.go`, `internal/solver/distribute.go`
- Why it matters: maximum path count is not automatically minimum total turns. `FindPaths` and `DistributeAnts` both use the same turn formula, first to choose how many paths to keep and then to decide how many ants each path receives.

## Graph mutation is intentional

Both max-flow search and path decomposition mutate the graph's `Flow` values.

- Implemented in: `internal/solver/solver.go`
- Why it matters: `FindPaths` consumes the graph as a working data structure. A caller should not expect to reuse the same `*graph.Graph` afterward as an untouched residual graph.

## Visualizers depend on parsed output, not shared solver internals

The visualizers do not call parser, graph, solver, or simulator packages directly. They parse the solver's stdout.

- Implemented in: `internal/format/format.go`
- Consumed by: `cmd/visualizer-tui/main.go`, `cmd/visualizer-web/main.go`
- Why it matters: visualizers are coupled to transcript format, not to the in-memory `parser.Colony` or solver path types. That keeps them loosely linked to the solver code, but tightly linked to output shape.

## Entry point duplication is deliberate

The repo has both `main.go` and `cmd/lem-in/main.go`, and they currently do the same work.

- Files: `main.go`, `cmd/lem-in/main.go`
- Why it matters: root `go run .` and the packaged CLI binary stay equivalent only as long as these files stay in lockstep.
