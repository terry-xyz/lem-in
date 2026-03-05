package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// helper writes content to a temp file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

// ---------- 1. Valid parsing of real example files ----------

func TestParse_Example00(t *testing.T) {
	c, err := Parse("../../examples/example00.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AntCount != 4 {
		t.Errorf("AntCount = %d, want 4", c.AntCount)
	}
	if c.StartName != "0" {
		t.Errorf("StartName = %q, want %q", c.StartName, "0")
	}
	if c.EndName != "1" {
		t.Errorf("EndName = %q, want %q", c.EndName, "1")
	}

	wantRooms := []struct {
		name string
		x, y int
	}{
		{"0", 0, 3},
		{"2", 2, 5},
		{"3", 4, 0},
		{"1", 8, 3},
	}
	if len(c.Rooms) != len(wantRooms) {
		t.Fatalf("Rooms count = %d, want %d", len(c.Rooms), len(wantRooms))
	}
	for i, wr := range wantRooms {
		r := c.Rooms[i]
		if r.Name != wr.name || r.X != wr.x || r.Y != wr.y {
			t.Errorf("Rooms[%d] = %+v, want {Name:%s X:%d Y:%d}", i, r, wr.name, wr.x, wr.y)
		}
	}

	wantLinks := [][2]string{{"0", "2"}, {"2", "3"}, {"3", "1"}}
	if len(c.Links) != len(wantLinks) {
		t.Fatalf("Links count = %d, want %d", len(c.Links), len(wantLinks))
	}
	for i, wl := range wantLinks {
		if c.Links[i] != wl {
			t.Errorf("Links[%d] = %v, want %v", i, c.Links[i], wl)
		}
	}

	// RoomMap should map every room name to its index
	for i, r := range c.Rooms {
		idx, ok := c.RoomMap[r.Name]
		if !ok {
			t.Errorf("RoomMap missing key %q", r.Name)
		} else if idx != i {
			t.Errorf("RoomMap[%q] = %d, want %d", r.Name, idx, i)
		}
	}
}

// ---------- 2. ##start not on first room line (example03) ----------

func TestParse_Example03(t *testing.T) {
	c, err := Parse("../../examples/example03.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AntCount != 4 {
		t.Errorf("AntCount = %d, want 4", c.AntCount)
	}
	// In example03, ##start is before room "0" which is the second room line.
	// First room line is "4 5 4".
	if c.StartName != "0" {
		t.Errorf("StartName = %q, want %q", c.StartName, "0")
	}
	if c.EndName != "5" {
		t.Errorf("EndName = %q, want %q", c.EndName, "5")
	}

	// First room parsed should be "4" (no command prefix), not start.
	if c.Rooms[0].Name != "4" {
		t.Errorf("first room = %q, want %q", c.Rooms[0].Name, "4")
	}
}

// ---------- 3. File with #rooms comment (example05) ----------

func TestParse_Example05(t *testing.T) {
	c, err := Parse("../../examples/example05.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.AntCount != 9 {
		t.Errorf("AntCount = %d, want 9", c.AntCount)
	}
	if c.StartName != "start" {
		t.Errorf("StartName = %q, want %q", c.StartName, "start")
	}
	if c.EndName != "end" {
		t.Errorf("EndName = %q, want %q", c.EndName, "end")
	}

	// The #rooms comment should be preserved in Lines
	found := false
	for _, l := range c.Lines {
		if l == "#rooms" {
			found = true
			break
		}
	}
	if !found {
		t.Error("comment '#rooms' not found in Lines")
	}
}

// ---------- 4. Ant count validation ----------

func TestParse_AntCount(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "zero ants",
			content: "0\n##start\nA 0 0\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid number of Ants",
		},
		{
			name:    "negative ants",
			content: "-5\n##start\nA 0 0\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid number of Ants",
		},
		{
			name:    "non-numeric ants",
			content: "abc\n##start\nA 0 0\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid number of Ants",
		},
		{
			name:    "empty file",
			content: "",
			wantErr: "invalid number of Ants",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.content)
			_, err := Parse(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// badexample00 has 0 ants
func TestParse_BadExample00(t *testing.T) {
	_, err := Parse("../../examples/badexample00.txt")
	if err == nil {
		t.Fatal("expected error for badexample00, got nil")
	}
	if !strings.Contains(err.Error(), "invalid number of Ants") {
		t.Errorf("error = %q, want it to contain 'invalid number of Ants'", err.Error())
	}
}

// ---------- 5. Room validation ----------

func TestParse_RoomValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "name starting with L",
			content: "1\n##start\nLroom 0 0\n##end\nB 1 1\nLroom-B\n",
			wantErr: "invalid room name",
		},
		{
			name:    "name starting with #",
			content: "1\n##start\n#bad 0 0\n##end\nB 1 1\n",
			// Lines starting with # are treated as comments, so the ##start command
			// is never fulfilled and a room named "#bad" is never created.
			// The error will be about command placement or missing start.
			wantErr: "ERROR:",
		},
		{
			name:    "duplicate room",
			content: "1\n##start\nA 0 0\nA 1 1\n##end\nB 2 2\nA-B\n",
			wantErr: "duplicate room",
		},
		{
			name:    "non-integer X coordinate",
			content: "1\n##start\nA abc 0\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid coordinates",
		},
		{
			name:    "non-integer Y coordinate",
			content: "1\n##start\nA 0 xyz\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid coordinates",
		},
		{
			name:    "negative X coordinate",
			content: "1\n##start\nA -1 0\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid coordinates",
		},
		{
			name:    "negative Y coordinate",
			content: "1\n##start\nA 0 -2\n##end\nB 1 1\nA-B\n",
			wantErr: "invalid coordinates",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.content)
			_, err := Parse(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ---------- 6. Link validation ----------

func TestParse_LinkValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "self-link",
			content: "1\n##start\nA 0 0\n##end\nB 1 1\nC 2 2\nA-B\nC-C\n",
			wantErr: "self-link",
		},
		{
			name:    "duplicate link",
			content: "1\n##start\nA 0 0\n##end\nB 1 1\nA-B\nB-A\n",
			wantErr: "duplicate link",
		},
		{
			name:    "link to unknown room (first)",
			content: "1\n##start\nA 0 0\n##end\nB 1 1\nX-B\n",
			wantErr: "link to unknown room",
		},
		{
			name:    "link to unknown room (second)",
			content: "1\n##start\nA 0 0\n##end\nB 1 1\nA-Z\n",
			wantErr: "link to unknown room",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.content)
			_, err := Parse(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// badexample01 has self-link 3-3
func TestParse_BadExample01(t *testing.T) {
	_, err := Parse("../../examples/badexample01.txt")
	if err == nil {
		t.Fatal("expected error for badexample01, got nil")
	}
	if !strings.Contains(err.Error(), "ERROR: invalid data format") {
		t.Errorf("error = %q, want it to contain 'ERROR: invalid data format'", err.Error())
	}
}

// ---------- 7. Command handling ----------

func TestParse_CommandHandling(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "missing ##start",
			content: "1\nA 0 0\n##end\nB 1 1\nA-B\n",
			wantErr: "no start room found",
		},
		{
			name:    "missing ##end",
			content: "1\n##start\nA 0 0\nB 1 1\nA-B\n",
			wantErr: "no end room found",
		},
		{
			name:    "start equals end",
			content: "1\n##start\n##end\nA 0 0\nB 1 1\nA-B\n",
			wantErr: "start and end are the same room",
		},
		{
			name:    "##start not followed by room (followed by link)",
			content: "1\nA 0 0\n##end\nB 1 1\n##start\nA-B\n",
			wantErr: "invalid command placement",
		},
		{
			name:    "##start at end of file (never fulfilled)",
			content: "1\n##end\nA 0 0\nB 1 1\n##start\n",
			wantErr: "invalid command placement",
		},
		{
			name:    "duplicate ##start",
			content: "1\n##start\nA 0 0\n##start\nC 2 2\n##end\nB 1 1\nA-B\n",
			wantErr: "duplicate start command",
		},
		{
			name:    "duplicate ##end",
			content: "1\n##start\nA 0 0\n##end\nB 1 1\n##end\nC 2 2\nA-B\n",
			wantErr: "duplicate end command",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.content)
			_, err := Parse(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// ---------- 8. Unknown ## commands ignored ----------

func TestParse_UnknownDoubleHashIgnored(t *testing.T) {
	content := "1\n##start\nA 0 0\n##end\nB 1 1\n##foobar\n##anything\nA-B\n"
	path := writeTemp(t, content)

	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AntCount != 1 {
		t.Errorf("AntCount = %d, want 1", c.AntCount)
	}
	if c.StartName != "A" {
		t.Errorf("StartName = %q, want %q", c.StartName, "A")
	}
	if c.EndName != "B" {
		t.Errorf("EndName = %q, want %q", c.EndName, "B")
	}
	if len(c.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(c.Links))
	}
}

// ---------- 9. Comments preserved in Lines ----------

func TestParse_CommentsPreservedInLines(t *testing.T) {
	content := "2\n#this is a comment\n##start\nA 0 0\n##end\nB 1 1\n#link section\nA-B\n"
	path := writeTemp(t, content)

	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lines should contain the original input lines including comments
	foundComment1 := false
	foundComment2 := false
	for _, l := range c.Lines {
		if l == "#this is a comment" {
			foundComment1 = true
		}
		if l == "#link section" {
			foundComment2 = true
		}
	}
	if !foundComment1 {
		t.Error("comment '#this is a comment' not preserved in Lines")
	}
	if !foundComment2 {
		t.Error("comment '#link section' not preserved in Lines")
	}
}

// ---------- 10. File not found ----------

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/path/file.txt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cannot read file") {
		t.Errorf("error = %q, want it to contain 'cannot read file'", err.Error())
	}
}

// ---------- 11. Additional room validation edge cases ----------

func TestParse_RoomNameWithDash(t *testing.T) {
	content := "1\n##start\nmy-room 0 0\n##end\nB 1 1\nmy-room-B\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for room name containing dash, got nil")
	}
	if !strings.Contains(err.Error(), "invalid room name") {
		t.Errorf("error = %q, want it to contain 'invalid room name'", err.Error())
	}
}

func TestParse_EmptyRoomName(t *testing.T) {
	// A line like " 0 0" where the room name is empty after Fields split
	// won't produce an empty name via Fields (it splits on whitespace),
	// but the parser's parseRoom checks for empty name explicitly.
	// We can test via a line that would be parsed as a room with empty name.
	content := "1\n##start\n 0 0\n##end\nB 1 1\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for empty room name, got nil")
	}
	// Either "invalid room name" or other parse error is acceptable
	if !strings.Contains(err.Error(), "ERROR:") {
		t.Errorf("error = %q, want it to contain 'ERROR:'", err.Error())
	}
}

func TestParse_AntCountExceedsLimit(t *testing.T) {
	content := "99999999\n##start\nA 0 0\n##end\nB 1 1\nA-B\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for ant count exceeding limit, got nil")
	}
	if !strings.Contains(err.Error(), "invalid number of Ants") {
		t.Errorf("error = %q, want it to contain 'invalid number of Ants'", err.Error())
	}
}

// ---------- Additional edge cases ----------

func TestParse_MinimalValid(t *testing.T) {
	content := "1\n##start\nA 0 0\n##end\nB 1 1\nA-B\n"
	path := writeTemp(t, content)

	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AntCount != 1 {
		t.Errorf("AntCount = %d, want 1", c.AntCount)
	}
	if c.StartName != "A" {
		t.Errorf("StartName = %q, want %q", c.StartName, "A")
	}
	if c.EndName != "B" {
		t.Errorf("EndName = %q, want %q", c.EndName, "B")
	}
	if len(c.Rooms) != 2 {
		t.Errorf("Rooms count = %d, want 2", len(c.Rooms))
	}
	if len(c.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(c.Links))
	}
}

func TestParse_AllErrorsPrefixed(t *testing.T) {
	// Every error from the parser must start with "ERROR: invalid data format, "
	badInputs := []string{
		"",                                   // empty
		"abc",                                // non-numeric ants
		"0\n##start\nA 0 0\n##end\nB 1 1\n",  // zero ants
		"1\nA 0 0\n##end\nB 1 1\n",           // no start
		"1\n##start\nA 0 0\nB 1 1\n",         // no end
		"1\n##start\nLx 0 0\n##end\nB 1 1\n", // L-prefix room
		"1\n##start\nA 0 0\nA 1 1\n##end\nB 2 2\n",           // duplicate room
		"1\n##start\nA 0 0\n##end\nB 1 1\nA-B\nC-B\n",        // unknown room in link
		"1\n##start\nA 0 0\n##end\nB 1 1\nC 2 2\nA-B\nC-C\n", // self-link
		"1\n##start\n##end\nA 0 0\nB 1 1\nA-B\n",             // start == end
		"1\n##start\nA abc 0\n##end\nB 1 1\nA-B\n",           // bad coords
	}

	for i, input := range badInputs {
		path := writeTemp(t, input)
		_, err := Parse(path)
		if err == nil {
			t.Errorf("case %d: expected error, got nil", i)
			continue
		}
		if !strings.HasPrefix(err.Error(), "ERROR: invalid data format, ") {
			t.Errorf("case %d: error = %q, want prefix 'ERROR: invalid data format, '", i, err.Error())
		}
	}
}

// ---------- 12. Additional edge case coverage ----------

func TestParse_RoomNameWithSpace(t *testing.T) {
	// "my room 0 0" splits into 4 fields, so parseRoom rejects it as invalid data
	content := "1\n##start\nmy room 0 0\n##end\nB 1 1\nmy room-B\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for room name with embedded space, got nil")
	}
	if !strings.Contains(err.Error(), "ERROR:") {
		t.Errorf("error = %q, want ERROR: prefix", err.Error())
	}
}

func TestParse_BlankLinesHandledGracefully(t *testing.T) {
	// Blank lines interspersed between rooms and links should be ignored
	content := "1\n\n##start\nA 0 0\n\n##end\nB 1 1\n\nA-B\n"
	path := writeTemp(t, content)
	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.AntCount != 1 {
		t.Errorf("AntCount = %d, want 1", c.AntCount)
	}
	if c.StartName != "A" {
		t.Errorf("StartName = %q, want %q", c.StartName, "A")
	}
	if c.EndName != "B" {
		t.Errorf("EndName = %q, want %q", c.EndName, "B")
	}
	if len(c.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(c.Links))
	}
}

func TestParse_CommentBetweenCommandAndRoom(t *testing.T) {
	// ##start followed by a comment, then the room definition -- should succeed
	content := "1\n##start\n#this is a comment\nA 0 0\n##end\nB 1 1\nA-B\n"
	path := writeTemp(t, content)
	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.StartName != "A" {
		t.Errorf("StartName = %q, want %q", c.StartName, "A")
	}
	if c.EndName != "B" {
		t.Errorf("EndName = %q, want %q", c.EndName, "B")
	}
}

func TestParse_MultipleErrorsOneReported(t *testing.T) {
	// Input has multiple problems: 0 ants AND missing ##end AND duplicate room
	content := "0\n##start\nA 0 0\nA 1 1\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should report exactly one error, not multiple joined together
	errStr := err.Error()
	count := strings.Count(errStr, "ERROR:")
	if count != 1 {
		t.Errorf("expected exactly 1 ERROR: prefix, got %d in %q", count, errStr)
	}
}

func TestParse_BlankFirstLine(t *testing.T) {
	// First line is blank -- should fail to parse ant count
	content := "\n1\n##start\nA 0 0\n##end\nB 1 1\nA-B\n"
	path := writeTemp(t, content)
	_, err := Parse(path)
	if err == nil {
		t.Fatal("expected error for blank first line, got nil")
	}
	if !strings.Contains(err.Error(), "invalid number of Ants") {
		t.Errorf("error = %q, want it to contain 'invalid number of Ants'", err.Error())
	}
}

func TestParse_MaxAntsBoundary(t *testing.T) {
	// Exactly 10,000,000 should succeed
	content := "10000000\n##start\nA 0 0\n##end\nB 1 1\nA-B\n"
	path := writeTemp(t, content)
	c, err := Parse(path)
	if err != nil {
		t.Fatalf("10M ants should succeed: %v", err)
	}
	if c.AntCount != 10_000_000 {
		t.Errorf("AntCount = %d, want 10000000", c.AntCount)
	}

	// 10,000,001 should fail
	content2 := "10000001\n##start\nA 0 0\n##end\nB 1 1\nA-B\n"
	path2 := writeTemp(t, content2)
	_, err2 := Parse(path2)
	if err2 == nil {
		t.Fatal("10M+1 ants should fail")
	}
	if !strings.Contains(err2.Error(), "invalid number of Ants") {
		t.Errorf("error = %q, want it to contain 'invalid number of Ants'", err2.Error())
	}
}

func TestParse_LinesSliceMatchesInput(t *testing.T) {
	content := "3\n##start\nA 0 0\n##end\nB 1 1\nA-B"
	path := writeTemp(t, content)

	c, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLines := strings.Split(content, "\n")
	if len(c.Lines) != len(expectedLines) {
		t.Fatalf("Lines count = %d, want %d", len(c.Lines), len(expectedLines))
	}
	for i, l := range expectedLines {
		if c.Lines[i] != l {
			t.Errorf("Lines[%d] = %q, want %q", i, c.Lines[i], l)
		}
	}
}
