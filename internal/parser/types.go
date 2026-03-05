package parser

// Room represents a room in the ant colony with a name and coordinates.
type Room struct {
	Name string
	X    int
	Y    int
}

// Colony holds all parsed data from the input file.
type Colony struct {
	AntCount  int
	Rooms     []Room
	RoomMap   map[string]int // name -> index in Rooms slice
	Links     [][2]string    // pairs of room names
	StartName string
	EndName   string
	Lines     []string // original input lines for verbatim reproduction
}
