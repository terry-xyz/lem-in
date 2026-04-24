# Contributing to lem-in

Thank you for your interest in contributing to `lem-in`. This guide explains how to set up the project, work within the existing structure, and submit changes cleanly.

## Getting Started

1. Fork the repository on GitHub.
2. Clone your fork:

```bash
git clone https://github.com/YOUR_USERNAME/lem-in.git
cd lem-in
```

3. Create a feature branch:

```bash
git checkout -b feature/your-feature-name
```

## Development Requirements

- Go 1.25.4 or higher
- `make` is optional, but the provided targets are the easiest way to run common tasks

## Code Standards

- Follow standard Go conventions and keep packages focused
- Run `gofmt -w .` before committing code changes
- Use `make fmt` if you want to check for formatting drift
- Write tests for new behavior and bug fixes
- Keep changes small, direct, and readable

## Testing

Run the full test suite with Make:

```bash
make test
```

Other useful targets:

```bash
make bench
make fuzz
make lint
make coverage
```

Or run the commands directly:

```bash
go test -p 1 ./...
go test -bench=. ./...
go test -fuzz=Fuzz -fuzztime=30s ./internal/parser/
go vet ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Building

Build all binaries:

```bash
make build
```

Without `make`:

```bash
go build -o bin/lem-in ./cmd/lem-in
go build -o bin/visualizer-tui ./cmd/visualizer-tui
go build -o bin/visualizer-web ./cmd/visualizer-web
```

## Running the Project

Run the solver:

```bash
go run ./cmd/lem-in examples/example00.txt
```

Run the terminal visualizer:

```bash
go run ./cmd/lem-in examples/example00.txt | go run ./cmd/visualizer-tui
```

Generate the web visualizer output:

```bash
go run ./cmd/lem-in examples/example01.txt | go run ./cmd/visualizer-web > colony.html
```

## Submitting Changes

1. Make sure the relevant tests pass before opening a pull request.
2. Use clear commit messages. This repository follows a Conventional Commit style such as:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation updates
   - `test:` for test changes
   - `chore:` for maintenance work
3. Push your branch to your fork.
4. Open a Pull Request against `main`.
5. Include context in the PR description: what changed, why it changed, and how you verified it.

## Project Structure

Understanding the main directories will make contributions easier:

- `cmd/lem-in/main.go`: solver CLI entry point
- `cmd/visualizer-tui/main.go`: terminal visualizer entry point
- `cmd/visualizer-web/main.go`: standalone HTML and 3D visualizer generator
- `internal/parser`: input parsing and validation for colony maps
- `internal/graph`: flow-graph construction from parsed input
- `internal/solver`: path finding and ant distribution logic
- `internal/simulator`: turn-by-turn movement generation
- `internal/format`: parsing of solver output for the visualizers
- `examples/`: sample input maps for manual runs and experimentation
- `Makefile`: standard build, test, lint, fuzz, and visualization commands

## Code Review

All contributions should be ready for review before submission. In practice that means:

- Tests relevant to the change pass
- Code matches Go conventions and existing project patterns
- Documentation is updated when behavior, usage, or developer workflow changes
- Generated outputs or demo files are updated only when intentionally part of the change

## Questions?

If you want to discuss a change before implementing it, open an issue in the [GitHub issue tracker](https://github.com/terry-xyz/lem-in/issues).
