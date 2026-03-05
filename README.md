# lem-in

A digital ant farm solver that finds the fastest way to move a group of ants from one room to another through a network of tunnels.

---

## What Does This Program Do?

Imagine a maze of rooms connected by tunnels. You have a bunch of ants sitting in a **start** room, and they all need to reach the **end** room. The catch: each tunnel is narrow (only one ant can pass through at a time), and each room (except start/end) can hold only one ant at a time. This program figures out the **fastest possible route plan** so all ants arrive in the fewest number of turns.

On top of solving the problem, the project includes two **bonus visualizers**:
- A **terminal visualizer** that animates the ants moving through the maze right in your console.
- A **3D web visualizer** that generates an interactive HTML page you can open in your browser.

---

## Prerequisites

Before you begin, you need one thing installed on your computer:

### Install Go (the programming language)

1. Go to [https://go.dev/dl/](https://go.dev/dl/)
2. Download the installer for your operating system (Windows, macOS, or Linux).
3. Run the installer and follow the on-screen instructions.
4. To verify it worked, open a terminal (Command Prompt, PowerShell, or Terminal) and type:

```bash
go version
```

You should see something like `go version go1.25.4 ...`. If you see an error, restart your terminal and try again.

### Install Make (optional but recommended)

`make` lets you run shortcut commands. It comes pre-installed on macOS and most Linux systems. On Windows:

- If you use **Git Bash** (comes with [Git for Windows](https://git-scm.com/downloads)), `make` is often already available.
- Otherwise, you can skip `make` entirely — every command below has a plain `go` alternative.

---

## Step 1: Get the Project

Download or clone this project to your computer:

```bash
git clone https://platform.zone01.gr/git/lpapanthy/lem-in.git
cd lem-in
```

If you don't have `git`, you can download the project as a ZIP file and extract it, then open a terminal inside the extracted folder.

---

## Step 2: Build the Program

This step compiles the source code into runnable programs.

**With Make:**

```bash
make build
```

**Without Make:**

```bash
go build -o bin/lem-in ./cmd/lem-in
go build -o bin/visualizer-tui ./cmd/visualizer-tui
go build -o bin/visualizer-web ./cmd/visualizer-web
```

After this, you'll find three programs inside the `bin/` folder:
| Program | What it does |
|---|---|
| `bin/lem-in` | The solver — reads a map file and outputs the solution |
| `bin/visualizer-tui` | Terminal visualizer — animates the solution in your console |
| `bin/visualizer-web` | Web visualizer — generates an HTML file with a 3D animation |

---

## Step 3: Run the Solver

The solver reads a map file (describing rooms, tunnels, and how many ants you have) and prints out the step-by-step moves.

**With Make:**

```bash
make run FILE=examples/example00.txt
```

**Without Make:**

```bash
go run . examples/example00.txt
```

**Or using the compiled binary:**

```bash
./bin/lem-in examples/example00.txt
```

### What You'll See

The program prints your input file, then a blank line, then the solution — one line per turn:

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

L1-2
L1-3 L2-2
L1-1 L2-3 L3-2
L2-1 L3-3 L4-2
L3-1 L4-3
L4-1
```

Each move looks like `L<ant_number>-<room_name>`. So `L1-2` means "Ant 1 moves to room 2". Each line is one turn — all moves on the same line happen simultaneously.

---

## Step 4: Try the Visualizers

### Terminal Visualizer (TUI)

This shows an animated view of the ants moving through the map, right in your terminal.

**With Make:**

```bash
make viz-tui FILE=examples/example00.txt
```

**Without Make:**

```bash
go run . examples/example00.txt | go run ./cmd/visualizer-tui
```

**Controls while running:**
- Press any key to advance turns or use the on-screen controls.
- The visualizer color-codes rooms and ants for easy tracking.

### 3D Web Visualizer

This generates an HTML file that you can open in any web browser for an interactive 3D view.

**With Make:**

```bash
make viz-web FILE=examples/example01.txt > colony.html
```

**Without Make:**

```bash
go run . examples/example01.txt | go run ./cmd/visualizer-web > colony.html
```

Then open `colony.html` in your browser (Chrome, Firefox, Edge, etc.). You'll see rooms as 3D objects with animated ants traveling along tunnels.

---

## Step 5: Try Different Maps

The `examples/` folder contains several pre-made maps of varying complexity:

| File | Ants | Description |
|---|---|---|
| `example00.txt` | 4 | Small, simple network |
| `example01.txt` | 10 | Medium graph, multiple paths |
| `example02.txt` | 20 | Two-path graph |
| `example03.txt` | 4 | Larger topology |
| `example04.txt` – `example07.txt` | Various | Increasingly complex networks |

Try them all:

```bash
go run . examples/example01.txt
go run . examples/example05.txt
```

There are also two **bad** example files that demonstrate error handling:

```bash
go run . examples/badexample00.txt   # invalid ant count (0)
go run . examples/badexample01.txt   # malformed input
```

These will print an error message — that's expected!

---

## Step 6: Create Your Own Maps

You can write your own map files. Create a text file (e.g., `mymap.txt`) with this format:

```
<number_of_ants>
##start
<start_room> <x> <y>
<room_name> <x> <y>
...
##end
<end_room> <x> <y>
<room1>-<room2>
<room1>-<room3>
...
```

### Rules for Map Files

- **First line**: number of ants (must be greater than 0).
- **Rooms**: a name followed by two numbers (coordinates). Names cannot start with `L` or `#`.
- **`##start`**: the next room line after this becomes the starting room.
- **`##end`**: the next room line after this becomes the ending room.
- **Tunnels**: two room names separated by a `-` (e.g., `roomA-roomB`). Tunnels are bidirectional.
- **Comments**: lines starting with a single `#` are ignored.

### Example Custom Map

Create a file called `mymap.txt`:

```
3
##start
entrance 0 0
hallway 1 0
kitchen 2 0
##end
exit 3 0
entrance-hallway
hallway-kitchen
kitchen-exit
entrance-kitchen
```

Then run it:

```bash
go run . mymap.txt
```

---

## Running Tests (For the Curious)

If you want to verify everything works correctly:

```bash
make test        # run all tests
make bench       # run performance benchmarks
make fuzz        # run fuzz tests (30 seconds of random input testing)
make lint        # check for code issues
make coverage    # generate a test coverage report (opens coverage.html)
```

**Without Make:**

```bash
go test -p 1 ./...                              # all tests
go test -bench=. ./...                           # benchmarks
go test -fuzz=Fuzz -fuzztime=30s ./internal/parser/  # fuzz testing
go vet ./...                                     # linting
```

---

## Project Structure

```
lem-in/
├── cmd/
│   ├── lem-in/              # solver program
│   ├── visualizer-tui/      # terminal visualizer
│   └── visualizer-web/      # 3D web visualizer
├── internal/
│   ├── parser/              # reads and validates map files
│   ├── graph/               # builds the internal network model
│   ├── solver/              # finds optimal paths (Edmonds-Karp algorithm)
│   ├── simulator/           # simulates ant movement turn by turn
│   └── format/              # shared output parser for visualizers
├── examples/                # sample map files
├── Makefile                 # shortcut commands
├── go.mod                   # Go module definition
└── README.md                # this file
```

---

## How the Algorithm Works (Simplified)

1. **Read** the map file and validate it.
2. **Build** an internal graph where each room is split into an "in" node and an "out" node — this ensures only one ant uses each room at a time.
3. **Find paths** using the Edmonds-Karp algorithm (a method for finding the maximum number of non-overlapping routes).
4. **Distribute** ants across the paths so the total number of turns is minimized — shorter paths get more ants.
5. **Simulate** the movement turn by turn and print the result.

---

## Troubleshooting

| Problem | Solution |
|---|---|
| `go: command not found` | Install Go from [go.dev/dl](https://go.dev/dl/) and restart your terminal |
| `make: command not found` | Use the `go` commands directly (shown in each section above) |
| `ERROR: invalid data format` | Check your map file follows the format described in "Create Your Own Maps" |
| Program prints nothing | Make sure you're passing a file path: `go run . examples/example00.txt` |
| Visualizer doesn't start | Make sure you're piping with `\|`: `go run . file.txt \| go run ./cmd/visualizer-tui` |

---

## Quick Reference

```bash
# Build everything
make build

# Solve a map
go run . examples/example00.txt

# Watch it in the terminal
go run . examples/example01.txt | go run ./cmd/visualizer-tui

# Generate a 3D web page
go run . examples/example01.txt | go run ./cmd/visualizer-web > colony.html

# Run all tests
make test

# Clean up compiled files
make clean
```
