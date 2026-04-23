package main

import (
	"fmt"
	"os"
	"strings"

	"lem-in/internal/graph"
	"lem-in/internal/parser"
	"lem-in/internal/simulator"
	"lem-in/internal/solver"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("USAGE: go run ./cmd/lem-in <your-file.txt>")
		return
	}

	filename := os.Args[1]

	colony, err := parser.Parse(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	g := graph.BuildGraph(colony)

	paths, err := solver.FindPaths(g, colony.AntCount)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, assignments := solver.DistributeAnts(paths, colony.AntCount)

	moveLines := simulator.Simulate(paths, assignments)

	// Print original input
	fmt.Println(strings.Join(colony.Lines, "\n"))

	// Blank separator line
	fmt.Println()

	// Print moves
	for _, line := range moveLines {
		fmt.Println(line)
	}
}
