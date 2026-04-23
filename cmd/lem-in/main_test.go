package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestIntegration builds the CLI and verifies representative example files produce valid schedules within the expected turn limits.
func TestIntegration(t *testing.T) {
	// Build the binary first
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "lem-in.exe")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = filepath.Join(".") // cmd/lem-in
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	examplesDir, err := filepath.Abs("../../examples")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		file      string
		maxTurns  int
		expectErr bool
	}{
		{"example00", "example00.txt", 6, false},
		{"example01", "example01.txt", 8, false},
		{"example02", "example02.txt", 11, false},
		{"example03", "example03.txt", 6, false},
		{"example04", "example04.txt", 6, false},
		{"example05", "example05.txt", 8, false},
		{"example06", "example06.txt", 100, false},
		{"example07", "example07.txt", 1000, false},
		{"badexample00", "badexample00.txt", 0, true},
		{"badexample01", "badexample01.txt", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(examplesDir, tt.file)
			cmd := exec.Command(binPath, filePath)
			out, err := cmd.CombinedOutput()
			output := string(out)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got success")
				}
				if !strings.HasPrefix(strings.TrimSpace(output), "ERROR: invalid data format") {
					t.Errorf("expected ERROR message, got: %s", output)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v\nOutput: %s", err, output)
			}

			// Read original file
			fileContent, readErr := os.ReadFile(filePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			origLines := strings.TrimRight(strings.ReplaceAll(string(fileContent), "\r\n", "\n"), "\n")

			// Verify output starts with file content
			outputNorm := strings.ReplaceAll(output, "\r\n", "\n")
			if !strings.HasPrefix(outputNorm, origLines) {
				t.Error("output does not start with original file content")
			}

			// Find the separator and move lines
			parts := strings.SplitN(outputNorm, "\n\n", 2)
			if len(parts) < 2 {
				t.Fatal("no blank separator line found")
			}

			moveSection := strings.TrimSpace(parts[1])
			if moveSection == "" {
				t.Fatal("no move lines found")
			}

			moveLines := strings.Split(moveSection, "\n")
			turnCount := len(moveLines)

			if turnCount > tt.maxTurns {
				t.Errorf("turn count %d exceeds max %d", turnCount, tt.maxTurns)
			}

			// Parse colony info for invariant validation
			colony := parseColonyFromFileContent(origLines)

			// Parse ant count from first line
			firstLine := strings.TrimSpace(strings.Split(origLines, "\n")[0])
			antCount, _ := strconv.Atoi(firstLine)

			// Validate all spec invariants on the move output
			validateMoves(t, moveLines, antCount, colony)

			fmt.Printf("  %s: %d turns (limit %d)\n", tt.name, turnCount, tt.maxTurns)
		})
	}
}

// TestNoArgs verifies the CLI prints the usage hint and exits cleanly when no input file is provided.
func TestNoArgs(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected exit 0 for no args: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != "USAGE: go run ./cmd/lem-in <your-file.txt>" {
		t.Errorf("unexpected usage message: %q", got)
	}
}

// TestNonexistentFile verifies the CLI reports a parser-style error when the requested input file cannot be read.
func TestNonexistentFile(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "nonexistent_file.txt")
	cmd.Dir = "."
	out, _ := cmd.CombinedOutput()
	if !strings.Contains(string(out), "ERROR: invalid data format, cannot read file") {
		t.Errorf("expected file error, got: %s", out)
	}
}

// colonyInfo holds colony topology for move validation.
type colonyInfo struct {
	startRoom string
	endRoom   string
	links     map[string]bool // normalized "min\x00max" for undirected tunnels
}

// parseColonyFromFileContent extracts start/end rooms and links from the input file content.
func parseColonyFromFileContent(content string) colonyInfo {
	ci := colonyInfo{
		links: make(map[string]bool),
	}
	lines := strings.Split(content, "\n")
	pendingStart := false
	pendingEnd := false

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "##start") {
			pendingStart = true
			continue
		}
		if strings.HasPrefix(line, "##end") {
			pendingEnd = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 3 {
			_, err1 := strconv.Atoi(fields[1])
			_, err2 := strconv.Atoi(fields[2])
			if err1 == nil && err2 == nil {
				if pendingStart {
					ci.startRoom = fields[0]
					pendingStart = false
				}
				if pendingEnd {
					ci.endRoom = fields[0]
					pendingEnd = false
				}
				continue
			}
		}

		dashIdx := strings.Index(line, "-")
		if dashIdx > 0 && dashIdx < len(line)-1 && !strings.Contains(line, " ") {
			a := line[:dashIdx]
			b := line[dashIdx+1:]
			ci.links[normalizeTunnel(a, b)] = true
		}
	}

	return ci
}

// normalizeTunnel returns a stable undirected tunnel key so start-end and end-start compare as the same edge.
func normalizeTunnel(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "\x00" + b
}

// validateMoves checks all spec invariants on move output:
// - Move format is Lx-y with ascending ant IDs per turn
// - Each ant moves at most once per turn
// - Each move uses an existing tunnel
// - No tunnel is used more than once per turn
// - No intermediate room holds more than one ant after any turn
// - All N ants reach ##end by the final turn
func validateMoves(t *testing.T, lines []string, totalAnts int, colony colonyInfo) {
	t.Helper()

	moveRe := regexp.MustCompile(`^L(\d+)-(\S+)$`)

	// All ants start in the start room
	antRoom := make(map[int]string, totalAnts)
	for i := 1; i <= totalAnts; i++ {
		antRoom[i] = colony.startRoom
	}

	for turnIdx, line := range lines {
		tokens := strings.Fields(line)
		movedThisTurn := make(map[int]bool)
		tunnelsUsed := make(map[string]bool)
		lastID := 0

		for _, tok := range tokens {
			matches := moveRe.FindStringSubmatch(tok)
			if matches == nil {
				t.Errorf("turn %d: invalid move format: %s", turnIdx+1, tok)
				continue
			}
			antID, _ := strconv.Atoi(matches[1])
			dest := matches[2]

			if antID <= lastID {
				t.Errorf("turn %d: ant IDs not ascending (%d after %d)", turnIdx+1, antID, lastID)
			}
			lastID = antID

			if antID < 1 || antID > totalAnts {
				t.Errorf("turn %d: unexpected ant ID: %d", turnIdx+1, antID)
				continue
			}

			if movedThisTurn[antID] {
				t.Errorf("turn %d: ant %d moved twice", turnIdx+1, antID)
			}
			movedThisTurn[antID] = true

			if antRoom[antID] == colony.endRoom {
				t.Errorf("turn %d: ant %d already at end, should not move", turnIdx+1, antID)
			}

			src := antRoom[antID]
			tunnelKey := normalizeTunnel(src, dest)
			if !colony.links[tunnelKey] {
				t.Errorf("turn %d: ant %d moved %s->%s but no tunnel exists", turnIdx+1, antID, src, dest)
			}

			if tunnelsUsed[tunnelKey] {
				t.Errorf("turn %d: tunnel %s-%s used more than once", turnIdx+1, src, dest)
			}
			tunnelsUsed[tunnelKey] = true

			antRoom[antID] = dest
		}

		// Check intermediate room capacity after all moves this turn
		roomCount := make(map[string]int)
		for _, room := range antRoom {
			roomCount[room]++
		}
		for room, count := range roomCount {
			if room == colony.startRoom || room == colony.endRoom {
				continue
			}
			if count > 1 {
				t.Errorf("turn %d: intermediate room %q has %d ants (max 1)", turnIdx+1, room, count)
			}
		}
	}

	// All ants must have reached the end room
	for antID := 1; antID <= totalAnts; antID++ {
		if antRoom[antID] != colony.endRoom {
			t.Errorf("ant %d did not reach end room (last position: %s)", antID, antRoom[antID])
		}
	}
}
