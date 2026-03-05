package simulator

import (
	"fmt"
	"sort"
	"strings"

	"lem-in/internal/solver"
)

// antState tracks an ant's position along its path.
type antState struct {
	AntID     int
	PathIndex int
	StepIndex int // current position in path (0=start, len-1=end)
}

// Simulate generates the turn-by-turn movement output.
// Returns the move lines (one string per turn).
func Simulate(paths []solver.Path, antsPerPath []int, assignments []solver.AntAssignment) []string {
	if len(assignments) == 0 {
		return nil
	}

	// Group assignments by path
	pathAnts := make([][]int, len(paths))
	for _, a := range assignments {
		pathAnts[a.PathIndex] = append(pathAnts[a.PathIndex], a.AntID)
	}

	// Create ant states - ants enter one per path per turn
	var active []*antState
	nextAnt := make([]int, len(paths)) // index into pathAnts[i] for next ant to enter

	var lines []string

	for {
		var moves []struct {
			antID int
			room  string
		}

		// Advance existing ants one step
		var stillActive []*antState
		for _, ant := range active {
			path := paths[ant.PathIndex]
			ant.StepIndex++
			room := path.Rooms[ant.StepIndex]
			moves = append(moves, struct {
				antID int
				room  string
			}{ant.AntID, room})
			// Keep if not at end
			if ant.StepIndex < len(path.Rooms)-1 {
				stillActive = append(stillActive, ant)
			}
		}
		active = stillActive

		// Launch new ants (one per path per turn)
		for pathIdx := 0; pathIdx < len(paths); pathIdx++ {
			if nextAnt[pathIdx] >= len(pathAnts[pathIdx]) {
				continue
			}
			antID := pathAnts[pathIdx][nextAnt[pathIdx]]
			nextAnt[pathIdx]++

			path := paths[pathIdx]
			if len(path.Rooms) <= 1 {
				// Direct start->end, just move to end
				moves = append(moves, struct {
					antID int
					room  string
				}{antID, path.Rooms[len(path.Rooms)-1]})
				continue
			}

			// Ant enters at step 1 (first room after start)
			ant := &antState{
				AntID:     antID,
				PathIndex: pathIdx,
				StepIndex: 1,
			}
			room := path.Rooms[1]
			moves = append(moves, struct {
				antID int
				room  string
			}{antID, room})

			if ant.StepIndex < len(path.Rooms)-1 {
				active = append(active, ant)
			}
		}

		if len(moves) == 0 {
			break
		}

		// Sort by ant ID
		sort.Slice(moves, func(i, j int) bool {
			return moves[i].antID < moves[j].antID
		})

		// Build output line
		var sb strings.Builder
		for i, m := range moves {
			if i > 0 {
				sb.WriteByte(' ')
			}
			fmt.Fprintf(&sb, "L%d-%s", m.antID, m.room)
		}
		lines = append(lines, sb.String())
	}

	return lines
}
