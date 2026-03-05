package solver

// AntAssignment maps an ant (by 1-based ID) to a path index.
type AntAssignment struct {
	AntID     int
	PathIndex int
}

// DistributeAnts assigns ants to paths optimally.
// Paths must be sorted by ascending length.
// Returns per-path ant counts and ordered assignments.
func DistributeAnts(paths []Path, antCount int) ([]int, []AntAssignment) {
	k := len(paths)
	if k == 0 {
		return nil, nil
	}

	// Compute optimal turn count T using the correct formula
	T := computeTurns(paths, antCount)

	// Each path i gets T - Li + 1 ants (ai + Li - 1 = T => ai = T - Li + 1)
	antsPerPath := make([]int, k)
	totalAssigned := 0
	for i := 0; i < k; i++ {
		count := T - paths[i].Length() + 1
		if count < 0 {
			count = 0
		}
		antsPerPath[i] = count
		totalAssigned += count
	}

	// Adjust for rounding: remove excess from longest paths
	excess := totalAssigned - antCount
	for i := k - 1; excess > 0 && i >= 0; i-- {
		if antsPerPath[i] > 0 {
			antsPerPath[i]--
			excess--
		}
	}

	// Create ordered assignments: lower ant IDs on shorter paths
	var assignments []AntAssignment
	antID := 1
	for pathIdx := 0; pathIdx < k; pathIdx++ {
		for j := 0; j < antsPerPath[pathIdx]; j++ {
			assignments = append(assignments, AntAssignment{
				AntID:     antID,
				PathIndex: pathIdx,
			})
			antID++
		}
	}

	return antsPerPath, assignments
}
