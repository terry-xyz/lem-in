package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootGoRun verifies that "go run ." works from the project root
// with valid examples, producing correct output format.
func TestRootGoRun(t *testing.T) {
	rootDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		file string
	}{
		{"example00", "examples/example00.txt"},
		{"example01", "examples/example01.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(rootDir, tt.file)
			cmd := exec.Command("go", "run", ".", filePath)
			cmd.Dir = rootDir
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go run . %s failed: %v\n%s", tt.file, err, out)
			}

			output := strings.ReplaceAll(string(out), "\r\n", "\n")

			// Read original file to verify it appears in output
			fileContent, readErr := os.ReadFile(filePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			origLines := strings.TrimRight(strings.ReplaceAll(string(fileContent), "\r\n", "\n"), "\n")

			if !strings.HasPrefix(output, origLines) {
				t.Error("output does not start with original file content")
			}

			// Verify blank separator and move lines exist
			parts := strings.SplitN(output, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("no blank separator line found between input and moves")
			}

			moveSection := strings.TrimSpace(parts[1])
			if moveSection == "" {
				t.Fatal("no move lines found after separator")
			}

			// Verify move format (Lx-y tokens)
			for _, line := range strings.Split(moveSection, "\n") {
				for _, tok := range strings.Fields(line) {
					if !strings.HasPrefix(tok, "L") {
						t.Errorf("invalid move token (missing L prefix): %s", tok)
					}
				}
			}
		})
	}
}

// TestRootBadExamples verifies that bad inputs produce ERROR messages.
func TestRootBadExamples(t *testing.T) {
	rootDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		file string
	}{
		{"badexample00", "examples/badexample00.txt"},
		{"badexample01", "examples/badexample01.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(rootDir, tt.file)
			cmd := exec.Command("go", "run", ".", filePath)
			cmd.Dir = rootDir
			out, err := cmd.CombinedOutput()
			output := strings.TrimSpace(string(out))

			if err == nil {
				t.Error("expected non-zero exit for bad input")
			}
			if !strings.HasPrefix(output, "ERROR: invalid data format") {
				t.Errorf("expected ERROR message, got: %s", output)
			}
		})
	}
}

// TestRootNoArgs verifies USAGE message when no arguments are provided.
func TestRootNoArgs(t *testing.T) {
	rootDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 for no args: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "USAGE") {
		t.Errorf("expected USAGE message, got: %s", out)
	}
}

// TestBuildAll verifies that "go build ./..." compiles all packages.
func TestBuildAll(t *testing.T) {
	rootDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build ./... failed: %v\n%s", err, out)
	}
}
