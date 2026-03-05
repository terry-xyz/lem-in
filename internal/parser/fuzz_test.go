package parser

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzParse exercises the parser with arbitrary input to ensure it never panics.
// Seeded with all 10 audit example files.
func FuzzParse(f *testing.F) {
	// Seed corpus from example files
	examples := []string{
		"example00.txt", "example01.txt", "example02.txt", "example03.txt",
		"example04.txt", "example05.txt", "example06.txt", "example07.txt",
		"badexample00.txt", "badexample01.txt",
	}
	for _, name := range examples {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", name))
		if err != nil {
			f.Fatalf("failed to read seed %s: %v", name, err)
		}
		f.Add(string(data))
	}

	// Additional edge-case seeds
	f.Add("")
	f.Add("0\n")
	f.Add("1\n##start\nA 0 0\n##end\nB 1 1\nA-B\n")
	f.Add("abc\n")
	f.Add("-1\n##start\nA 0 0\n##end\nB 1 1\nA-B\n")
	f.Add("1\n##start\nLbad 0 0\n##end\nB 1 1\n")
	f.Add("1\n##start\n##end\nA 0 0\nA-A\n")

	f.Fuzz(func(t *testing.T, input string) {
		// Write input to a temp file
		dir := t.TempDir()
		path := filepath.Join(dir, "fuzz_input.txt")
		if err := os.WriteFile(path, []byte(input), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		// Parse must not panic; errors are acceptable
		_, _ = Parse(path)
	})
}
