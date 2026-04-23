package parser

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const maxAnts = 10_000_000

// Parse reads and validates a lem-in input file, returning a Colony or an error.
func Parse(filename string) (*Colony, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("ERROR: invalid data format, cannot read file")
	}

	content := string(data)
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimRight(content, "\n")

	if content == "" {
		return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
	}

	lines := strings.Split(content, "\n")
	return parseLines(lines)
}

// parseLines walks normalized input lines once, building rooms and links while enforcing lem-in command ordering rules.
func parseLines(lines []string) (*Colony, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
	}

	c := &Colony{
		RoomMap: make(map[string]int),
		Lines:   lines,
	}

	// Phase 1: Parse ant count
	antStr := strings.TrimSpace(lines[0])
	antCount, err := strconv.Atoi(antStr)
	if err != nil || antCount <= 0 {
		return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
	}
	if antCount > maxAnts {
		return nil, fmt.Errorf("ERROR: invalid data format, invalid number of ants")
	}
	c.AntCount = antCount

	// Phase 2 & 3: Parse rooms and links using state machine
	pendingStart := false
	pendingEnd := false
	startFound := false
	endFound := false
	inLinks := false
	linkSet := make(map[string]bool) // normalized "a-b" where a < b

	for i := 1; i < len(lines); i++ {
		line := lines[i]

		// Handle comments and commands
		if strings.HasPrefix(line, "##") {
			cmd := line[2:]
			switch cmd {
			case "start":
				if startFound {
					return nil, fmt.Errorf("ERROR: invalid data format, duplicate start command")
				}
				if pendingStart || pendingEnd {
					return nil, fmt.Errorf("ERROR: invalid data format, invalid command placement")
				}
				pendingStart = true
			case "end":
				if endFound {
					return nil, fmt.Errorf("ERROR: invalid data format, duplicate end command")
				}
				if pendingStart || pendingEnd {
					return nil, fmt.Errorf("ERROR: invalid data format, invalid command placement")
				}
				pendingEnd = true
			default:
				// Unknown ## command: silently ignored
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			// Single # comment: preserved in lines but doesn't affect data
			continue
		}

		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Try to parse as a link (contains exactly one dash in a name1-name2 pattern)
		if isLink(line) {
			if pendingStart || pendingEnd {
				return nil, fmt.Errorf("ERROR: invalid data format, invalid command placement")
			}
			inLinks = true

			parts := strings.SplitN(line, "-", 2)
			name1 := parts[0]
			name2 := parts[1]

			if _, ok := c.RoomMap[name1]; !ok {
				return nil, fmt.Errorf("ERROR: invalid data format, link to unknown room: %s", name1)
			}
			if _, ok := c.RoomMap[name2]; !ok {
				return nil, fmt.Errorf("ERROR: invalid data format, link to unknown room: %s", name2)
			}
			if name1 == name2 {
				return nil, fmt.Errorf("ERROR: invalid data format, self-link: %s", name1)
			}

			// Normalize link for duplicate detection
			a, b := name1, name2
			if a > b {
				a, b = b, a
			}
			key := a + "-" + b
			if linkSet[key] {
				return nil, fmt.Errorf("ERROR: invalid data format, duplicate link: %s-%s", name1, name2)
			}
			linkSet[key] = true

			c.Links = append(c.Links, [2]string{name1, name2})
			continue
		}

		// If we're past the links section, non-link non-comment lines are errors
		if inLinks {
			return nil, fmt.Errorf("ERROR: invalid data format, invalid data")
		}

		// Try to parse as a room
		room, err := parseRoom(line)
		if err != nil {
			return nil, err
		}

		// Validate room name
		if room.Name == "" {
			return nil, fmt.Errorf("ERROR: invalid data format, invalid room name")
		}
		if strings.HasPrefix(room.Name, "L") {
			return nil, fmt.Errorf("ERROR: invalid data format, invalid room name: %s", room.Name)
		}
		if strings.HasPrefix(room.Name, "#") {
			return nil, fmt.Errorf("ERROR: invalid data format, invalid room name: %s", room.Name)
		}

		// Check for duplicate
		if _, exists := c.RoomMap[room.Name]; exists {
			return nil, fmt.Errorf("ERROR: invalid data format, duplicate room: %s", room.Name)
		}

		// Add room
		idx := len(c.Rooms)
		c.Rooms = append(c.Rooms, room)
		c.RoomMap[room.Name] = idx

		// Handle pending commands
		if pendingStart {
			c.StartName = room.Name
			startFound = true
			pendingStart = false
		}
		if pendingEnd {
			c.EndName = room.Name
			endFound = true
			pendingEnd = false
		}
	}

	// Check for pending command that was never fulfilled
	if pendingStart || pendingEnd {
		return nil, fmt.Errorf("ERROR: invalid data format, invalid command placement")
	}

	// Validate start/end
	if !startFound {
		return nil, fmt.Errorf("ERROR: invalid data format, no start room found")
	}
	if !endFound {
		return nil, fmt.Errorf("ERROR: invalid data format, no end room found")
	}
	if c.StartName == c.EndName {
		return nil, fmt.Errorf("ERROR: invalid data format, start and end are the same room")
	}

	return c, nil
}

// isLink checks if a line looks like a link definition (name1-name2).
// A link contains a dash where neither side is empty, and neither side
// contains a space.
func isLink(line string) bool {
	dashIdx := strings.Index(line, "-")
	if dashIdx <= 0 || dashIdx >= len(line)-1 {
		return false
	}
	left := line[:dashIdx]
	right := line[dashIdx+1:]
	// If left or right contains spaces, it's not a valid link format
	// (could be a room with negative coords, but rooms use spaces not dashes)
	if strings.Contains(left, " ") || strings.Contains(right, " ") {
		return false
	}
	return true
}

// parseRoom parses a single room definition line and rejects names or coordinates that cannot belong to a room.
func parseRoom(line string) (Room, error) {
	parts := strings.Fields(line)
	if len(parts) != 3 {
		return Room{}, fmt.Errorf("ERROR: invalid data format, invalid data")
	}

	name := parts[0]
	if strings.Contains(name, "-") {
		return Room{}, fmt.Errorf("ERROR: invalid data format, invalid room name: %s", name)
	}

	x, err := strconv.Atoi(parts[1])
	if err != nil || x < 0 {
		return Room{}, fmt.Errorf("ERROR: invalid data format, invalid coordinates for room: %s", name)
	}
	y, err := strconv.Atoi(parts[2])
	if err != nil || y < 0 {
		return Room{}, fmt.Errorf("ERROR: invalid data format, invalid coordinates for room: %s", name)
	}

	return Room{Name: name, X: x, Y: y}, nil
}
