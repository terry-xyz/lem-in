package solver_test

import (
	"testing"

	"lem-in/internal/graph"
	"lem-in/internal/parser"
	"lem-in/internal/simulator"
	"lem-in/internal/solver"
)

// BenchmarkParse benchmarks the parser on example04.
func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse("../../examples/example04.txt")
		if err != nil {
			b.Fatalf("parse error: %v", err)
		}
	}
}

// BenchmarkSolver100 benchmarks the full solver pipeline with 100 ants on example04's graph.
func BenchmarkSolver100(b *testing.B) {
	colony, err := parser.Parse("../../examples/example04.txt")
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}
	colony.AntCount = 100

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := graph.BuildGraph(colony)
		paths, err := solver.FindPaths(g, colony.AntCount)
		if err != nil {
			b.Fatalf("find paths error: %v", err)
		}
		_, assignments := solver.DistributeAnts(paths, colony.AntCount)
		_ = simulator.Simulate(paths, assignments)
	}
}

// BenchmarkSolver1000 benchmarks the full solver pipeline with 1000 ants on example04's graph.
func BenchmarkSolver1000(b *testing.B) {
	colony, err := parser.Parse("../../examples/example04.txt")
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}
	colony.AntCount = 1000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := graph.BuildGraph(colony)
		paths, err := solver.FindPaths(g, colony.AntCount)
		if err != nil {
			b.Fatalf("find paths error: %v", err)
		}
		_, assignments := solver.DistributeAnts(paths, colony.AntCount)
		_ = simulator.Simulate(paths, assignments)
	}
}

// BenchmarkSimulator benchmarks only the simulation phase with 1000 ants.
func BenchmarkSimulator(b *testing.B) {
	colony, err := parser.Parse("../../examples/example04.txt")
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}
	colony.AntCount = 1000

	g := graph.BuildGraph(colony)
	paths, err := solver.FindPaths(g, colony.AntCount)
	if err != nil {
		b.Fatalf("find paths error: %v", err)
	}
	_, assignments := solver.DistributeAnts(paths, colony.AntCount)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = simulator.Simulate(paths, assignments)
	}
}

// BenchmarkExample06 benchmarks the full pipeline on example06 (100 ants).
func BenchmarkExample06(b *testing.B) {
	colony, err := parser.Parse("../../examples/example06.txt")
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := graph.BuildGraph(colony)
		paths, err := solver.FindPaths(g, colony.AntCount)
		if err != nil {
			b.Fatalf("find paths error: %v", err)
		}
		_, assignments := solver.DistributeAnts(paths, colony.AntCount)
		_ = simulator.Simulate(paths, assignments)
	}
}

// BenchmarkExample07 benchmarks the full pipeline on example07 (1000 ants).
func BenchmarkExample07(b *testing.B) {
	colony, err := parser.Parse("../../examples/example07.txt")
	if err != nil {
		b.Fatalf("parse error: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := graph.BuildGraph(colony)
		paths, err := solver.FindPaths(g, colony.AntCount)
		if err != nil {
			b.Fatalf("find paths error: %v", err)
		}
		_, assignments := solver.DistributeAnts(paths, colony.AntCount)
		_ = simulator.Simulate(paths, assignments)
	}
}
