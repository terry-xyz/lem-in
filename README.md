# lem-in

A digital ant farm solver that finds the optimal way to move N ants from a start room to an end room through a network of tunnels, using vertex-disjoint paths and minimum turns.

## Build

```bash
make build              # compile all binaries to bin/
# or
go build ./...          # build all packages
```

## Usage

```bash
# Run the solver
./bin/lem-in examples/example00.txt
# or
go run ./cmd/lem-in examples/example00.txt

# TUI visualizer (pipe solver output)
./bin/lem-in examples/example00.txt | ./bin/visualizer-tui
# or
make viz-tui FILE=examples/example00.txt

# 3D Web visualizer (generates HTML to stdout)
./bin/lem-in examples/example00.txt | ./bin/visualizer-web > colony.html
# or
make viz-web FILE=examples/example00.txt > colony.html
```

## Testing

```bash
make test               # run all tests
make bench              # run benchmarks
make fuzz               # run fuzz tests (30s)
make lint               # go vet
make fmt                # check formatting
make coverage           # generate coverage report
```

## Project Structure

```
cmd/lem-in/             main solver binary
cmd/visualizer-tui/     terminal visualizer (bonus)
cmd/visualizer-web/     3D web visualizer (bonus)
internal/parser/        input file parsing + validation
internal/graph/         flow network with node-splitting
internal/solver/        Edmonds-Karp pathfinding + ant distribution
internal/simulator/     turn-by-turn output generation
internal/format/        shared output parser for visualizers
examples/               audit test input files
specs/                  project specifications
```

## Algorithm

1. **Parse** input file into colony structure (rooms, tunnels, ant count)
2. **Build** a flow network with node-splitting to enforce vertex-disjoint paths
3. **Solve** using Edmonds-Karp (BFS-based max flow) with optimal stopping
4. **Distribute** ants across paths to minimize total turns
5. **Simulate** turn-by-turn movement and output in `Lx-y` format

## Input Format

```
<number_of_ants>
##start
<start_room> <x> <y>
<room_name> <x> <y>
##end
<end_room> <x> <y>
<room1>-<room2>
```

## Output Format

The program prints the input file content, a blank line, then one line per turn with ant movements:

```
L1-room1 L2-room2
L1-room3 L2-room4 L3-room1
```
