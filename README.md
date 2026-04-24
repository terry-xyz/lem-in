# lem-in

`lem-in` solves the classic ant-colony routing problem: given a map of rooms and tunnels, it computes a set of non-conflicting paths and a turn-by-turn movement schedule that gets all ants from `##start` to `##end` in as few turns as possible.

The repository includes:

- `cmd/lem-in`: the solver CLI
- `cmd/visualizer-tui`: a terminal visualizer for solver output
- `cmd/visualizer-web`: an HTML/3D visualizer for solver output

⭐️ [Here](https://terry-xyz.github.io/lem-in/) is an example of 3D visualizer!

## Requirements

- Go 1.25.4 or newer
- `make` is optional

## Build

```sh
make build
```

Without `make`:

```sh
go build -o bin/lem-in ./cmd/lem-in
go build -o bin/visualizer-tui ./cmd/visualizer-tui
go build -o bin/visualizer-web ./cmd/visualizer-web
```

## Quick Start

Run the solver against one of the bundled maps:

```sh
go run ./cmd/lem-in examples/example00.txt
```

Or use the shortcut:

```sh
make run FILE=examples/example00.txt
```

The solver prints:

1. The original input map
2. A blank separator line
3. One movement line per turn, for example `L1-2 L2-3`

## Visualizers

### TUI

Pipe solver output into the terminal visualizer:

```sh
go run ./cmd/lem-in examples/example00.txt | go run ./cmd/visualizer-tui
```

Or:

```sh
make viz-tui FILE=examples/example00.txt
```

### Web

Generate a standalone HTML visualization:

```sh
go run ./cmd/lem-in examples/example01.txt | go run ./cmd/visualizer-web > colony.html
```

Or:

```sh
make viz-web examples/example01.txt
```

This writes `colony.html`. Open it in a browser.

You can override the output file:

```sh
make viz-web examples/example01.txt OUT=my-colony.html
```

To generate `colony.html` and open it automatically:

```sh
make viz-web-open examples/example01.txt
```

The open target also supports `OUT`:

```sh
make viz-web-open examples/example01.txt OUT=my-colony.html
```

## Input Format

Maps are plain text files:

```text
<ant_count>
##start
<start_room> <x> <y>
<room_name> <x> <y>
...
##end
<end_room> <x> <y>
<room_a>-<room_b>
<room_b>-<room_c>
...
```

Rules:

- The first line is the ant count and must be greater than `0`
- `##start` marks the next room as the start room
- `##end` marks the next room as the end room
- Room names cannot start with `L` or `#`
- Tunnel lines are bidirectional links in `room1-room2` form
- Lines starting with a single `#` are treated as comments

Sample inputs are available in `[examples/](./examples)`.

## Project Layout

```text
.
|-- cmd/
|   |-- lem-in/
|   |-- visualizer-tui/
|   `-- visualizer-web/
|-- examples/
|-- internal/
|   |-- format/
|   |-- graph/
|   |-- parser/
|   |-- simulator/
|   `-- solver/
|-- Makefile
`-- README.md
```

## Development

```sh
make test
make bench
make fuzz
make lint
make coverage
```

Without `make`:

```sh
go test -p 1 ./...
go test -bench=. ./...
go test -fuzz=Fuzz -fuzztime=30s ./internal/parser/
go vet ./...
```

## Notes

- `internal/parser` validates the colony description
- `internal/graph` converts the parsed map into a flow graph
- `internal/solver` finds paths and distributes ants across them
- `internal/simulator` emits the final turn-by-turn schedule
- `internal/format` parses solver output for the visualizers

