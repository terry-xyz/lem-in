package format

import (
	"fmt"
	"strconv"
	"strings"
)

// Movement represents a single ant move in a turn.
type Movement struct {
	AntID    int
	RoomName string
}

// ParsedOutput holds the structured result of parsing lem-in output.
type ParsedOutput struct {
	AntCount  int
	Rooms     []ParsedRoom
	Links     [][2]string
	StartName string
	EndName   string
	Turns     [][]Movement
	Error     string // non-empty if output was an error
}

// ParsedRoom holds room info extracted from output.
type ParsedRoom struct {
	Name    string
	X, Y    int
	IsStart bool
	IsEnd   bool
}

// ParseOutput parses lem-in stdout into structured data.
func ParseOutput(output string) (*ParsedOutput, error) {
	output = strings.TrimRight(output, "\n\r")
	if strings.HasPrefix(output, "ERROR:") {
		return &ParsedOutput{Error: output}, nil
	}

	lines := strings.Split(output, "\n")
	result := &ParsedOutput{}

	// Find the blank separator line
	sepIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			sepIdx = i
			break
		}
	}

	if sepIdx < 0 {
		return nil, fmt.Errorf("no separator line found in output")
	}

	// Parse file content section
	fileLines := lines[:sepIdx]
	if len(fileLines) == 0 {
		return nil, fmt.Errorf("empty file content section")
	}

	// First line is ant count
	antCount, err := strconv.Atoi(strings.TrimSpace(fileLines[0]))
	if err != nil {
		return nil, fmt.Errorf("invalid ant count: %s", fileLines[0])
	}
	result.AntCount = antCount

	isStart := false
	isEnd := false
	for _, line := range fileLines[1:] {
		if line == "##start" {
			isStart = true
			continue
		}
		if line == "##end" {
			isEnd = true
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Try room
		parts := strings.Fields(line)
		if len(parts) == 3 {
			x, xerr := strconv.Atoi(parts[1])
			y, yerr := strconv.Atoi(parts[2])
			if xerr == nil && yerr == nil {
				room := ParsedRoom{Name: parts[0], X: x, Y: y, IsStart: isStart, IsEnd: isEnd}
				if isStart {
					result.StartName = parts[0]
				}
				if isEnd {
					result.EndName = parts[0]
				}
				result.Rooms = append(result.Rooms, room)
				isStart = false
				isEnd = false
				continue
			}
		}

		// Try link
		if strings.Contains(line, "-") {
			linkParts := strings.SplitN(line, "-", 2)
			if len(linkParts) == 2 {
				result.Links = append(result.Links, [2]string{linkParts[0], linkParts[1]})
			}
		}
	}

	// Parse move lines
	if sepIdx+1 < len(lines) {
		for _, line := range lines[sepIdx+1:] {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var turn []Movement
			tokens := strings.Fields(line)
			for _, tok := range tokens {
				if !strings.HasPrefix(tok, "L") {
					continue
				}
				dashIdx := strings.Index(tok[1:], "-")
				if dashIdx < 0 {
					continue
				}
				antID, err := strconv.Atoi(tok[1 : 1+dashIdx])
				if err != nil {
					continue
				}
				roomName := tok[2+dashIdx:]
				turn = append(turn, Movement{AntID: antID, RoomName: roomName})
			}
			if len(turn) > 0 {
				result.Turns = append(result.Turns, turn)
			}
		}
	}

	return result, nil
}
