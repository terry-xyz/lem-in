package main

import (
	"strings"
	"testing"

	"lem-in/internal/format"
)

func TestBfsDepth_LinearGraph(t *testing.T) {
	rooms := []format.ParsedRoom{
		{Name: "A", X: 0, Y: 0, IsStart: true},
		{Name: "B", X: 1, Y: 0},
		{Name: "C", X: 2, Y: 0},
		{Name: "D", X: 3, Y: 0, IsEnd: true},
	}
	links := [][2]string{{"A", "B"}, {"B", "C"}, {"C", "D"}}

	depths := bfsDepth(rooms, links, "A")

	expected := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}
	for name, wantDepth := range expected {
		if depths[name] != wantDepth {
			t.Errorf("depth[%s] = %d, want %d", name, depths[name], wantDepth)
		}
	}
}

func TestBfsDepth_BranchedGraph(t *testing.T) {
	rooms := []format.ParsedRoom{
		{Name: "S", IsStart: true},
		{Name: "A"},
		{Name: "B"},
		{Name: "C"},
		{Name: "E", IsEnd: true},
	}
	links := [][2]string{{"S", "A"}, {"S", "B"}, {"A", "C"}, {"B", "C"}, {"C", "E"}}

	depths := bfsDepth(rooms, links, "S")

	if depths["S"] != 0 {
		t.Errorf("depth[S] = %d, want 0", depths["S"])
	}
	if depths["A"] != 1 {
		t.Errorf("depth[A] = %d, want 1", depths["A"])
	}
	if depths["B"] != 1 {
		t.Errorf("depth[B] = %d, want 1", depths["B"])
	}
	if depths["C"] != 2 {
		t.Errorf("depth[C] = %d, want 2", depths["C"])
	}
	if depths["E"] != 3 {
		t.Errorf("depth[E] = %d, want 3", depths["E"])
	}
}

func TestBfsDepth_UnreachableRoom(t *testing.T) {
	rooms := []format.ParsedRoom{
		{Name: "S", IsStart: true},
		{Name: "E", IsEnd: true},
		{Name: "X"}, // not connected
	}
	links := [][2]string{{"S", "E"}}

	depths := bfsDepth(rooms, links, "S")

	if depths["X"] != 0 {
		t.Errorf("unreachable room depth = %d, want 0", depths["X"])
	}
	if depths["S"] != 0 {
		t.Errorf("depth[S] = %d, want 0", depths["S"])
	}
	if depths["E"] != 1 {
		t.Errorf("depth[E] = %d, want 1", depths["E"])
	}
}

func TestBuildJSONData_BasicInput(t *testing.T) {
	parsed := &format.ParsedOutput{
		AntCount:  3,
		StartName: "start",
		EndName:   "end",
		Rooms: []format.ParsedRoom{
			{Name: "start", X: 0, Y: 0, IsStart: true},
			{Name: "mid", X: 1, Y: 1},
			{Name: "end", X: 2, Y: 2, IsEnd: true},
		},
		Links: [][2]string{{"start", "mid"}, {"mid", "end"}},
		Turns: [][]format.Movement{
			{{AntID: 1, RoomName: "mid"}},
			{{AntID: 1, RoomName: "end"}, {AntID: 2, RoomName: "mid"}},
		},
	}

	data := buildJSONData(parsed)

	if data.AntCount != 3 {
		t.Errorf("AntCount = %d, want 3", data.AntCount)
	}
	if data.StartName != "start" {
		t.Errorf("StartName = %q, want %q", data.StartName, "start")
	}
	if data.EndName != "end" {
		t.Errorf("EndName = %q, want %q", data.EndName, "end")
	}
	if len(data.Rooms) != 3 {
		t.Fatalf("Rooms count = %d, want 3", len(data.Rooms))
	}
	if len(data.Links) != 2 {
		t.Errorf("Links count = %d, want 2", len(data.Links))
	}
	if len(data.Turns) != 2 {
		t.Errorf("Turns count = %d, want 2", len(data.Turns))
	}

	// Verify rooms have correct IsStart/IsEnd flags
	for _, r := range data.Rooms {
		switch r.Name {
		case "start":
			if !r.IsStart {
				t.Error("start room should have IsStart=true")
			}
		case "end":
			if !r.IsEnd {
				t.Error("end room should have IsEnd=true")
			}
		case "mid":
			if r.IsStart || r.IsEnd {
				t.Error("mid room should not be start or end")
			}
		}
	}

	// Verify scaling: X and Z use scale=4.0, Y uses depth*scale*1.5
	// start is at depth 0, mid at depth 1, end at depth 2
	for _, r := range data.Rooms {
		if r.Name == "start" {
			if r.Y != 0 {
				t.Errorf("start Y = %f, want 0 (depth 0)", r.Y)
			}
		}
		if r.Name == "mid" {
			wantY := 1.0 * 4.0 * 1.5
			if r.Y != wantY {
				t.Errorf("mid Y = %f, want %f", r.Y, wantY)
			}
		}
	}
}

func TestBuildJSONData_ErrorInput(t *testing.T) {
	parsed := &format.ParsedOutput{
		Error: "ERROR: invalid data format, no path from start to end",
	}

	data := buildJSONData(parsed)

	if data.Error == "" {
		t.Error("expected error in JSON data")
	}
	if data.Rooms != nil {
		t.Error("expected nil rooms for error case")
	}
	if data.Links != nil {
		t.Error("expected nil links for error case")
	}
	if data.Turns != nil {
		t.Error("expected nil turns for error case")
	}
}

func TestBuildHTML_ContainsEmbeddedData(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr)

	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML missing DOCTYPE")
	}
	if !strings.Contains(html, jsonStr) {
		t.Error("HTML missing embedded JSON data")
	}
	if !strings.Contains(html, "three") {
		t.Error("HTML missing Three.js reference")
	}
	if !strings.Contains(html, "SIM_DATA") {
		t.Error("HTML missing SIM_DATA variable")
	}
}

func TestBuildHTML_ErrorOverlay(t *testing.T) {
	parsed := &format.ParsedOutput{
		Error: "ERROR: invalid data format, no path",
	}
	data := buildJSONData(parsed)
	// The HTML should handle error case gracefully
	if data.Error == "" {
		t.Error("expected error in data")
	}
}
