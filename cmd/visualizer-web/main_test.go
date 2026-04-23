package main

import (
	"math"
	"strings"
	"testing"

	"lem-in/internal/format"
)

// assertContainsBlock fails the test with the full missing snippet when generated HTML does not contain the expected block.
func assertContainsBlock(t *testing.T, html, block string) {
	t.Helper()

	if !strings.Contains(html, block) {
		t.Fatalf("HTML missing block:\n%s", block)
	}
}

// TestBfsDepth_LinearGraph verifies BFS depth labels increase by one along a simple chain of rooms.
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

// TestBfsDepth_BranchedGraph verifies BFS picks the shortest tunnel distance in a graph with branching paths.
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

// TestBfsDepth_UnreachableRoom verifies disconnected rooms fall back to depth zero instead of being omitted.
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

// TestBuildJSONData_BasicInput verifies parsed solver output becomes the expected JSON payload for the web visualizer.
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

	// Verify scaling: Y uses depth*ySpacing (3.8)
	// start is at depth 0, mid at depth 1, end at depth 2
	for _, r := range data.Rooms {
		if r.Name == "start" {
			if r.Y != 0 {
				t.Errorf("start Y = %f, want 0 (depth 0)", r.Y)
			}
		}
		if r.Name == "mid" {
			wantY := 1.0 * 3.8
			if r.Y != wantY {
				t.Errorf("mid Y = %f, want %f", r.Y, wantY)
			}
		}
	}
}

// TestBuildJSONData_SparseRoomsStayInsideColonyRadius verifies tiny maps still place rooms inside the shell radius budget.
func TestBuildJSONData_SparseRoomsStayInsideColonyRadius(t *testing.T) {
	parsed := &format.ParsedOutput{
		AntCount:  1,
		StartName: "start",
		EndName:   "end",
		Rooms: []format.ParsedRoom{
			{Name: "start", IsStart: true},
			{Name: "end", IsEnd: true},
		},
		Links: [][2]string{{"start", "end"}},
	}

	data := buildJSONData(parsed)

	for _, r := range data.Rooms {
		radius := math.Hypot(r.X, r.Z)
		if radius > 1.25 {
			t.Errorf("room %q radius = %.2f, want <= 1.25 so sparse layouts stay inside the colony", r.Name, radius)
		}
	}
}

// TestBuildJSONData_ErrorInput verifies parser errors short-circuit JSON generation instead of emitting partial scene data.
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

// TestBuildHTML_ContainsEmbeddedData verifies the generated HTML embeds the simulation payload and viewer dependencies.
func TestBuildHTML_ContainsEmbeddedData(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

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
	if !strings.Contains(html, "GLTFLoader") {
		t.Error("HTML missing GLTFLoader reference")
	}
	if !strings.Contains(html, "COLONY_MODEL_GZ_B64") {
		t.Error("HTML missing colony model data variable")
	}
}

// TestBuildHTML_AntOcclusionOutlineUsesDepthBuffer verifies ant outlines are configured to appear only when ants are hidden behind geometry.
func TestBuildHTML_AntOcclusionOutlineUsesDepthBuffer(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		"THREE.GreaterDepth",
		"depthWrite: false",
		"occlusionOutline",
		"mesh.userData.outline = outline",
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing ant occlusion outline snippet %q", snippet)
		}
	}
}

// TestBuildHTML_ShowColonyToggleMarkupAndHooks verifies the colony-visibility control and its script hooks are present in the document.
func TestBuildHTML_ShowColonyToggleMarkupAndHooks(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		`id="colony-toggle-wrap"`,
		`id="btn-show-colony"`,
		`SHOW COLONY`,
		`var showColony = true;`,
		`var colonyShellMaterials = [];`,
		`function applyColonyVisibility()`,
		`groundMat.opacity`,
		`colonyShellMaterials.forEach(function(mat)`,
		`button-shine`,
		`.glass-button`,
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing show colony snippet %q", snippet)
		}
	}
}

// TestBuildHTML_AutoRotateToggleMarkupAndHooks verifies the auto-rotate toggle exposes the expected markup, state, and event wiring.
func TestBuildHTML_AutoRotateToggleMarkupAndHooks(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		`id="auto-rotate-toggle-wrap"`,
		`id="btn-auto-rotate"`,
		`aria-pressed="true"`,
		`AUTO-ROTATE: YES`,
		`var autoRotateEnabled = true;`,
		`var btnAutoRotate = document.getElementById('btn-auto-rotate');`,
		`var btnAutoRotateText = document.getElementById('btn-auto-rotate-text');`,
		`function applyAutoRotateState()`,
		`btnAutoRotate.addEventListener('click', function() {`,
		`autoRotateEnabled = !autoRotateEnabled;`,
		`applyAutoRotateState();`,
		`controls.autoRotate = autoRotateEnabled;`,
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing auto rotate snippet %q", snippet)
		}
	}

	assertContainsBlock(t, html, `function applyAutoRotateState() {
  controls.autoRotate = autoRotateEnabled;
  btnAutoRotate.setAttribute('aria-pressed', autoRotateEnabled ? 'true' : 'false');
  btnAutoRotateText.textContent = autoRotateEnabled ? 'AUTO-ROTATE: YES' : 'AUTO-ROTATE: OFF';
}`)

	assertContainsBlock(t, html, `btnAutoRotate.addEventListener('click', function() {
  autoRotateEnabled = !autoRotateEnabled;
  applyAutoRotateState();
});`)

	assertContainsBlock(t, html, `@media (max-width: 820px) {
  #auto-rotate-toggle-wrap{left:16px;bottom:82px}
}`)
}

// TestBuildHTML_ShowColonyToggleRestoresLegacyCameraAndGuardedRecovery verifies re-showing the shell restores a safe camera position and reset animation.
func TestBuildHTML_ShowColonyToggleRestoresLegacyCameraAndGuardedRecovery(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		`var initialCameraPosition = camera.position.clone();`,
		`var initialControlsTarget = controls.target.clone();`,
		`var cameraResetActive = false;`,
		`var cameraResetDuration = 0.6;`,
		`function startCameraResetToOpeningView()`,
		`btnShowColony.addEventListener('click', function() {`,
		`showColony = !showColony;`,
		`applyColonyVisibility();`,
		`if (showColony && camera.position.y < groundY) {`,
		`startCameraResetToOpeningView();`,
		`if (showColony && !cameraResetActive) {`,
		`camera.position.y = Math.max(camera.position.y, groundY + cameraFloorOffset);`,
		`if (cameraResetActive) {`,
		`camera.position.lerpVectors(cameraResetFromPosition, initialCameraPosition, resetT);`,
		`controls.target.lerpVectors(cameraResetFromTarget, initialControlsTarget, resetT);`,
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing camera recovery snippet %q", snippet)
		}
	}

	assertContainsBlock(t, html, `function startCameraResetToOpeningView() {
  cameraResetActive = true;
  cameraResetElapsed = 0;
  cameraResetFromPosition.copy(camera.position);
  cameraResetFromTarget.copy(controls.target);
}`)

	assertContainsBlock(t, html, `btnShowColony.addEventListener('click', function() {
  showColony = !showColony;
  applyColonyVisibility();
  if (!showColony) cameraResetActive = false;
  if (showColony && camera.position.y < groundY) {
    startCameraResetToOpeningView();
  }
});`)

	assertContainsBlock(t, html, `if (cameraResetActive) {
  cameraResetElapsed += dt;
  var resetT = Math.min(cameraResetElapsed / cameraResetDuration, 1.0);
  resetT = resetT < 0.5 ? 4 * resetT * resetT * resetT : 1 - Math.pow(-2 * resetT + 2, 3) / 2;
  camera.position.lerpVectors(cameraResetFromPosition, initialCameraPosition, resetT);
  controls.target.lerpVectors(cameraResetFromTarget, initialControlsTarget, resetT);
  camera.lookAt(controls.target);
  if (resetT >= 1.0) cameraResetActive = false;
}`)

	assertContainsBlock(t, html, `if (showColony && !cameraResetActive) {
  var prevCameraY = camera.position.y;
  camera.position.y = Math.max(camera.position.y, groundY + cameraFloorOffset);
  if (camera.position.y !== prevCameraY) camera.lookAt(controls.target);
}`)
}

// TestBuildHTML_ShowColonyOffHidesShellAndShadowCompletely verifies hiding the shell also hides its ground shadow and opacity state.
func TestBuildHTML_ShowColonyOffHidesShellAndShadowCompletely(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		`var colonyHiddenOpacity = 0.0;`,
		`var groundHiddenOpacity = 0.0;`,
		`colony.visible = showColony;`,
		`groundMesh.visible = showColony;`,
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing full colony hide snippet %q", snippet)
		}
	}
}

// TestBuildHTML_AlignsColonyBaseAndClampsCameraFloor verifies the scene aligns the shell base to the room floor and prevents the camera from falling under it.
func TestBuildHTML_AlignsColonyBaseAndClampsCameraFloor(t *testing.T) {
	jsonStr := `{"antCount":3,"rooms":[],"links":[],"turns":[]}`
	html := buildHTML(jsonStr, "")

	required := []string{
		`var roomFloorY = minY - cy;`,
		`var colonyLift = roomFloorY - scaledMinY;`,
		`colony.position.y += colonyLift;`,
		`var groundY = alignedBox.min.y;`,
		`groundMesh.position.y = groundY - 0.02;`,
		`var cameraFloorOffset =`,
		`camera.position.y = Math.max(camera.position.y, groundY + cameraFloorOffset);`,
	}
	for _, snippet := range required {
		if !strings.Contains(html, snippet) {
			t.Errorf("HTML missing colony floor alignment snippet %q", snippet)
		}
	}
}

// TestBuildHTML_ErrorOverlay verifies the error-data path is preserved so the HTML can render an overlay instead of a broken scene.
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
