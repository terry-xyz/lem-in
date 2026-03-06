package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"lem-in/internal/format"
)

// ttyReader is the file used for keyboard input.
// When stdin is a pipe, we reopen /dev/tty (MINGW64) or CONIN$ (Windows).
var ttyReader *os.File

// ---------------------------------------------------------------------------
// ANSI escape helpers
// ---------------------------------------------------------------------------

const (
	esc = "\033["

	// Screen
	clearScreen   = esc + "2J"
	clearLine     = esc + "2K"
	hideCursor    = esc + "?25l"
	showCursor    = esc + "?25h"
	enableAltBuf  = esc + "?1049h"
	disableAltBuf = esc + "?1049l"

	// Colors – Nord palette (ANSI 256-color)
	fgReset    = esc + "0m"
	fgGreen    = esc + "1;38;5;108m" // Nord 14 (#A3BE8C) – start node
	fgRed      = esc + "1;38;5;131m" // Nord 11 (#BF616A) – end node
	fgYellow   = esc + "38;5;222m"   // Nord 13 (#EBCB8B) – ants/highlight
	fgDkGreen  = esc + "38;5;110m"   // Nord  8 (#88C0D0) – tunnels (frost)
	fgBrown    = esc + "38;5;253m"   // Nord  5 (#E5E9F0) – regular rooms
	fgWhite    = esc + "38;5;188m"   // Nord  4 (#D8DEE9) – standard text
	fgCyan     = esc + "38;5;109m"   // Nord  7 (#8FBCBB) – UI chrome
	fgBoldWht  = esc + "1;38;5;255m" // Nord  6 (#ECEFF4) – headers
	fgDimWhite = esc + "38;5;188m"   // Nord  4 (#D8DEE9) – panel text

	// Dimmed label colors – Nord polar-night tones
	fgDimGreen = esc + "38;5;65m" // dimmed start label
	fgDimRed   = esc + "38;5;95m" // dimmed end label
	fgDimGray  = esc + "38;5;59m" // Nord 3 (#4C566A) – regular labels

	// Background
	bgReset = esc + "49m"
)

func moveTo(row, col int) string {
	return fmt.Sprintf("%s%d;%dH", esc, row, col)
}

// ---------------------------------------------------------------------------
// Terminal size
// ---------------------------------------------------------------------------

func getTerminalSize() (cols, rows int) {
	// Try $COLUMNS / $LINES first (set by MINGW64 bash, most shells)
	cols = envInt("COLUMNS", 0)
	rows = envInt("LINES", 0)
	if cols > 0 && rows > 0 {
		return cols, rows
	}

	// Try `stty size` (works in MINGW64 bash)
	out, err := exec.Command("stty", "size").Output()
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) == 2 {
			if r, e1 := strconv.Atoi(parts[0]); e1 == nil {
				if c, e2 := strconv.Atoi(parts[1]); e2 == nil {
					return c, r
				}
			}
		}
	}

	// Try `tput`
	if c, err := tputInt("cols"); err == nil {
		if r, err := tputInt("lines"); err == nil {
			return c, r
		}
	}

	return 80, 24 // fallback
}

func envInt(name string, def int) int {
	s := os.Getenv(name)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func tputInt(cap string) (int, error) {
	out, err := exec.Command("tput", cap).Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

// ---------------------------------------------------------------------------
// Raw terminal mode (via stty)
// ---------------------------------------------------------------------------

var sttyOriginal string

func sttyCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("stty", args...)
	// When stdin is a pipe, stty cannot access the terminal.
	// Redirect its stdin to /dev/tty so it always operates on the real terminal.
	if ttyReader != nil {
		cmd.Stdin = ttyReader
	}
	return cmd
}

func enableRawMode() {
	// Save current settings
	out, err := sttyCmd("-g").Output()
	if err == nil {
		sttyOriginal = strings.TrimSpace(string(out))
	}
	// Set raw mode: no echo, no canonical mode, read 1 char at a time
	_ = sttyCmd("raw", "-echo", "min", "1", "time", "0").Run()
}

func disableRawMode() {
	if sttyOriginal != "" {
		_ = sttyCmd(sttyOriginal).Run()
	} else {
		_ = sttyCmd("sane").Run()
	}
}

// ---------------------------------------------------------------------------
// Buffered screen writer
// ---------------------------------------------------------------------------

type screenBuf struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (s *screenBuf) write(a string) {
	s.mu.Lock()
	s.buf.WriteString(a)
	s.mu.Unlock()
}

func (s *screenBuf) writef(format string, args ...interface{}) {
	s.write(fmt.Sprintf(format, args...))
}

func (s *screenBuf) flush() {
	s.mu.Lock()
	data := s.buf.String()
	s.buf.Reset()
	s.mu.Unlock()
	_, _ = os.Stdout.WriteString(data)
}

// ---------------------------------------------------------------------------
// Canvas: a 2D character + color buffer
// ---------------------------------------------------------------------------

type cell struct {
	ch rune
	fg string
}

type canvas struct {
	w, h  int
	cells [][]cell
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, cells: make([][]cell, h)}
	for r := 0; r < h; r++ {
		c.cells[r] = make([]cell, w)
		for col := 0; col < w; col++ {
			c.cells[r][col] = cell{ch: ' ', fg: fgReset}
		}
	}
	return c
}

func (c *canvas) set(x, y int, ch rune, fg string) {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		c.cells[y][x] = cell{ch: ch, fg: fg}
	}
}

func (c *canvas) setStr(x, y int, s string, fg string) {
	for i, ch := range s {
		c.set(x+i, y, ch, fg)
	}
}

func (c *canvas) get(x, y int) cell {
	if x >= 0 && x < c.w && y >= 0 && y < c.h {
		return c.cells[y][x]
	}
	return cell{ch: ' ', fg: fgReset}
}

// ---------------------------------------------------------------------------
// Coordinate scaling
// ---------------------------------------------------------------------------

type roomPos struct {
	screenX int
	screenY int
}

// scaleCoords maps room coordinates into the canvas area with margin.
func scaleCoords(rooms []format.ParsedRoom, canvasW, canvasH int) map[string]roomPos {
	if len(rooms) == 0 {
		return nil
	}

	margin := 3 // cells of clearance from edges

	// Find bounding box
	minX, maxX := rooms[0].X, rooms[0].X
	minY, maxY := rooms[0].Y, rooms[0].Y
	for _, r := range rooms[1:] {
		if r.X < minX {
			minX = r.X
		}
		if r.X > maxX {
			maxX = r.X
		}
		if r.Y < minY {
			minY = r.Y
		}
		if r.Y > maxY {
			maxY = r.Y
		}
	}

	// Usable area
	usableW := canvasW - 2*margin
	usableH := canvasH - 2*margin
	if usableW < 1 {
		usableW = 1
	}
	if usableH < 1 {
		usableH = 1
	}

	rangeX := float64(maxX - minX)
	rangeY := float64(maxY - minY)
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	result := make(map[string]roomPos, len(rooms))
	for _, r := range rooms {
		sx := margin + int(math.Round(float64(r.X-minX)/rangeX*float64(usableW)))
		sy := margin + int(math.Round(float64(r.Y-minY)/rangeY*float64(usableH)))
		// Clamp
		if sx < margin {
			sx = margin
		}
		if sx >= canvasW-margin {
			sx = canvasW - margin - 1
		}
		if sy < margin {
			sy = margin
		}
		if sy >= canvasH-margin {
			sy = canvasH - margin - 1
		}
		result[r.Name] = roomPos{screenX: sx, screenY: sy}
	}
	return result
}

// ---------------------------------------------------------------------------
// Direction-bitmap line drawing (proper merging of crossings & T-junctions)
// ---------------------------------------------------------------------------

type dirFlags uint8

const (
	dirUp    dirFlags = 1 << 0
	dirDown  dirFlags = 1 << 1
	dirLeft  dirFlags = 1 << 2
	dirRight dirFlags = 1 << 3
)

// boxChar converts direction flags to the appropriate box-drawing character.
func boxChar(d dirFlags) rune {
	switch d {
	case dirLeft | dirRight:
		return '─'
	case dirUp | dirDown:
		return '│'
	case dirRight | dirDown:
		return '┌'
	case dirLeft | dirDown:
		return '┐'
	case dirRight | dirUp:
		return '└'
	case dirLeft | dirUp:
		return '┘'
	case dirUp | dirDown | dirRight:
		return '├'
	case dirUp | dirDown | dirLeft:
		return '┤'
	case dirLeft | dirRight | dirDown:
		return '┬'
	case dirLeft | dirRight | dirUp:
		return '┴'
	case dirLeft | dirRight | dirUp | dirDown:
		return '┼'
	default:
		if d&(dirLeft|dirRight) != 0 {
			return '─'
		}
		if d&(dirUp|dirDown) != 0 {
			return '│'
		}
		return ' '
	}
}

type dirGrid struct {
	w, h  int
	flags [][]dirFlags
}

func newDirGrid(w, h int) *dirGrid {
	g := &dirGrid{w: w, h: h, flags: make([][]dirFlags, h)}
	for y := 0; y < h; y++ {
		g.flags[y] = make([]dirFlags, w)
	}
	return g
}

func (g *dirGrid) addFlag(x, y int, d dirFlags) {
	if x >= 0 && x < g.w && y >= 0 && y < g.h {
		g.flags[y][x] |= d
	}
}

// tracePath traces an L-shaped orthogonal path from (x0,y0) to (x1,y1).
// Routes horizontal first, then vertical.
func (g *dirGrid) tracePath(x0, y0, x1, y1 int) {
	if x0 == x1 && y0 == y1 {
		return
	}

	// Horizontal segment from (x0,y0) to (x1,y0)
	if x0 != x1 {
		sx := 1
		var dirFwd, dirBwd dirFlags
		if x1 > x0 {
			dirFwd, dirBwd = dirRight, dirLeft
		} else {
			sx = -1
			dirFwd, dirBwd = dirLeft, dirRight
		}
		for x := x0; x != x1; x += sx {
			g.addFlag(x, y0, dirFwd)
			g.addFlag(x+sx, y0, dirBwd)
		}
	}

	// Vertical segment from (x1,y0) to (x1,y1)
	if y0 != y1 {
		sy := 1
		var dirFwd, dirBwd dirFlags
		if y1 > y0 {
			dirFwd, dirBwd = dirDown, dirUp
		} else {
			sy = -1
			dirFwd, dirBwd = dirUp, dirDown
		}
		for y := y0; y != y1; y += sy {
			g.addFlag(x1, y, dirFwd)
			g.addFlag(x1, y+sy, dirBwd)
		}
	}
}

// applyToCanvas writes box-drawing characters to the canvas.
func (g *dirGrid) applyToCanvas(cv *canvas, fg string) {
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			if g.flags[y][x] != 0 {
				cv.set(x, y, boxChar(g.flags[y][x]), fg)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---------------------------------------------------------------------------
// Playback state
// ---------------------------------------------------------------------------

type playMode int

const (
	modeAutoPlay playMode = iota
	modeStep
	modeSlider
)

// point is a screen coordinate pair.
type point struct{ x, y int }

// animEntry describes one ant's animation path for a transition.
type animEntry struct {
	antID int
	path  []point // sequence of screen cells from source to destination
}

type playback struct {
	mode     playMode
	paused   bool
	turnIdx  int // -1 = initial state (before any moves)
	speed    int // milliseconds per animation transition
	minSpeed int
	maxSpeed int

	// Animation state
	animating  bool
	animForward bool         // true = forward transition, false = backward
	animFrame  int           // current frame (0..animTotal-1)
	animTotal  int           // total frames for this transition
	entries    []animEntry   // ants being animated this turn
	preAnts    *antState     // snapshot BEFORE the transition
}

func newPlayback() *playback {
	return &playback{
		mode:     modeAutoPlay,
		paused:   false,
		turnIdx:  -1,
		speed:    800,
		minSpeed: 100,
		maxSpeed: 3000,
	}
}

func (p *playback) faster() {
	p.speed -= 100
	if p.speed < p.minSpeed {
		p.speed = p.minSpeed
	}
}

func (p *playback) slower() {
	p.speed += 100
	if p.speed > p.maxSpeed {
		p.speed = p.maxSpeed
	}
}

// animDuration returns the number of frames for one transition based on speed.
func (p *playback) animDuration() int {
	const frameDurationMs = 33
	frames := p.speed / frameDurationMs
	if frames < 2 {
		frames = 2
	}
	return frames
}

// ---------------------------------------------------------------------------
// Ant state tracking
// ---------------------------------------------------------------------------

type antState struct {
	positions map[int]string // antID -> current room name
}

func newAntState(antCount int, startRoom string) *antState {
	as := &antState{positions: make(map[int]string, antCount)}
	for i := 1; i <= antCount; i++ {
		as.positions[i] = startRoom
	}
	return as
}

func (as *antState) applyTurn(moves []format.Movement) {
	for _, m := range moves {
		as.positions[m.AntID] = m.RoomName
	}
}

// clone returns a deep copy of the ant state.
func (as *antState) clone() *antState {
	c := &antState{positions: make(map[int]string, len(as.positions))}
	for k, v := range as.positions {
		c.positions[k] = v
	}
	return c
}

// antsAtRoom returns a sorted list of ant IDs currently at the given room.
func (as *antState) antsAtRoom(roomName string) []int {
	var ids []int
	for id, pos := range as.positions {
		if pos == roomName {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

type renderer struct {
	parsed    *format.ParsedOutput
	positions map[string]roomPos
	termW     int
	termH     int
	canvasW   int
	canvasH   int
	panelH    int
}

func newRenderer(parsed *format.ParsedOutput, termW, termH int) *renderer {
	panelH := 7 // rows reserved for info panel at bottom
	canvasH := termH - panelH
	if canvasH < 5 {
		canvasH = 5
	}
	canvasW := termW
	if canvasW < 10 {
		canvasW = 10
	}

	positions := scaleCoords(parsed.Rooms, canvasW, canvasH)

	return &renderer{
		parsed:    parsed,
		positions: positions,
		termW:     termW,
		termH:     termH,
		canvasW:   canvasW,
		canvasH:   canvasH,
		panelH:    panelH,
	}
}

// computePath generates an L-shaped screen path from srcRoom to dstRoom.
// Horizontal first, then vertical (matching tunnel drawing).
func (rn *renderer) computePath(srcRoom, dstRoom string) []point {
	src, okS := rn.positions[srcRoom]
	dst, okD := rn.positions[dstRoom]
	if !okS || !okD {
		return []point{{src.screenX, src.screenY}}
	}
	if src.screenX == dst.screenX && src.screenY == dst.screenY {
		return []point{{src.screenX, src.screenY}}
	}

	var path []point
	// Horizontal segment
	x, y := src.screenX, src.screenY
	if dst.screenX != x {
		dx := 1
		if dst.screenX < x {
			dx = -1
		}
		for x != dst.screenX {
			path = append(path, point{x, y})
			x += dx
		}
	}
	// Vertical segment
	if dst.screenY != y {
		dy := 1
		if dst.screenY < y {
			dy = -1
		}
		for y != dst.screenY {
			path = append(path, point{x, y})
			y += dy
		}
	}
	// Final destination point
	path = append(path, point{dst.screenX, dst.screenY})
	return path
}

// computeTransition builds animation entries for one turn's movements.
func (rn *renderer) computeTransition(ants *antState, moves []format.Movement) []animEntry {
	entries := make([]animEntry, 0, len(moves))
	for _, m := range moves {
		srcRoom := ants.positions[m.AntID]
		path := rn.computePath(srcRoom, m.RoomName)
		entries = append(entries, animEntry{antID: m.AntID, path: path})
	}
	return entries
}

func (rn *renderer) render(sb *screenBuf, ants *antState, pb *playback) {
	cv := newCanvas(rn.canvasW, rn.canvasH)

	// 1. Draw tunnels using direction grid for proper line merging
	dg := newDirGrid(rn.canvasW, rn.canvasH)
	for _, link := range rn.parsed.Links {
		posA, okA := rn.positions[link[0]]
		posB, okB := rn.positions[link[1]]
		if okA && okB {
			dg.tracePath(posA.screenX, posA.screenY, posB.screenX, posB.screenY)
		}
	}
	dg.applyToCanvas(cv, fgDkGreen)

	// 2. Draw rooms
	for _, room := range rn.parsed.Rooms {
		pos, ok := rn.positions[room.Name]
		if !ok {
			continue
		}

		var color, labelColor string
		var symbol rune
		switch {
		case room.IsStart:
			color = fgGreen
			labelColor = fgDimGreen
			symbol = '\u25C9' // ◉
		case room.IsEnd:
			color = fgRed
			labelColor = fgDimRed
			symbol = '\u25CE' // ◎
		default:
			color = fgBrown
			labelColor = fgDimGray
			symbol = '\u25CF' // ●
		}

		// Draw room symbol
		cv.set(pos.screenX, pos.screenY, symbol, color)

		// Draw room name (to the right or left if space allows)
		label := room.Name
		if len(label) > 8 {
			label = label[:8]
		}

		// Try placing label to the right
		labelX := pos.screenX + 2
		if labelX+len(label) >= rn.canvasW {
			// Place to the left
			labelX = pos.screenX - len(label) - 1
		}
		if labelX < 0 {
			labelX = 0
		}
		cv.setStr(labelX, pos.screenY, label, labelColor)
	}

	// 3. Draw ants
	if pb.animating && pb.preAnts != nil {
		// Build set of ant IDs that are moving this frame
		movingAnts := make(map[int]bool, len(pb.entries))
		for _, e := range pb.entries {
			movingAnts[e.antID] = true
		}

		// Draw non-moving ants from pre-transition snapshot
		for _, room := range rn.parsed.Rooms {
			pos, ok := rn.positions[room.Name]
			if !ok {
				continue
			}
			allIDs := pb.preAnts.antsAtRoom(room.Name)
			var staticIDs []int
			for _, id := range allIDs {
				if !movingAnts[id] {
					staticIDs = append(staticIDs, id)
				}
			}
			if len(staticIDs) == 0 {
				continue
			}
			rn.drawAntLabel(cv, pos.screenX, pos.screenY, staticIDs)
		}

		// Draw moving ants at interpolated positions
		for _, e := range pb.entries {
			idx := pb.animFrame * (len(e.path) - 1) / (pb.animTotal - 1)
			if idx >= len(e.path) {
				idx = len(e.path) - 1
			}
			p := e.path[idx]
			label := fmt.Sprintf("\u25B8L%d", e.antID)
			lx := p.x - len(label)/2
			if lx < 0 {
				lx = 0
			}
			if lx+len(label) >= rn.canvasW {
				lx = rn.canvasW - len(label) - 1
			}
			if lx < 0 {
				lx = 0
			}
			// Draw at the cell row offset by 1 if possible
			ay := p.y + 1
			if ay >= rn.canvasH {
				ay = p.y - 1
			}
			if ay < 0 {
				ay = p.y
			}
			cv.setStr(lx, ay, label, fgYellow)
		}
	} else {
		// Static: draw ants at their current room positions
		for _, room := range rn.parsed.Rooms {
			pos, ok := rn.positions[room.Name]
			if !ok {
				continue
			}
			antIDs := ants.antsAtRoom(room.Name)
			if len(antIDs) == 0 {
				continue
			}
			rn.drawAntLabel(cv, pos.screenX, pos.screenY, antIDs)
		}
	}

	// Build output – overwrite in place to prevent flicker (no clearScreen)
	sb.write(moveTo(1, 1))

	// Render canvas rows
	prevFg := ""
	for row := 0; row < rn.canvasH && row < rn.termH-rn.panelH; row++ {
		sb.write(moveTo(row+1, 1))
		for col := 0; col < rn.canvasW; col++ {
			c := cv.cells[row][col]
			if c.fg != prevFg {
				sb.write(c.fg)
				prevFg = c.fg
			}
			sb.writef("%c", c.ch)
		}
		sb.write(esc + "K") // clear to end of line
	}
	// Clear gap rows between canvas and panel
	panelTop := rn.termH - rn.panelH + 1
	for row := rn.canvasH + 1; row < panelTop; row++ {
		sb.write(moveTo(row, 1))
		sb.write(esc + "2K")
	}
	sb.write(fgReset)

	// Render info panel
	rn.renderPanel(sb, pb)

	sb.flush()
}

// drawAntLabel draws an ant label for the given IDs near the given screen position.
func (rn *renderer) drawAntLabel(cv *canvas, sx, sy int, antIDs []int) {
	antY := sy + 1
	if antY >= rn.canvasH {
		antY = sy - 1
	}
	if antY < 0 {
		antY = sy
	}

	var antLabel string
	if len(antIDs) <= 3 {
		parts := make([]string, len(antIDs))
		for i, id := range antIDs {
			parts[i] = fmt.Sprintf("L%d", id)
		}
		antLabel = "\u25B8" + strings.Join(parts, ",")
	} else {
		antLabel = fmt.Sprintf("\u25B8L%d..(%d)", antIDs[0], len(antIDs))
	}

	if len(antLabel) > rn.canvasW-2 {
		antLabel = antLabel[:rn.canvasW-2]
	}

	labelX := sx - len(antLabel)/2
	if labelX < 0 {
		labelX = 0
	}
	if labelX+len(antLabel) >= rn.canvasW {
		labelX = rn.canvasW - len(antLabel) - 1
	}
	if labelX < 0 {
		labelX = 0
	}

	cv.setStr(labelX, antY, antLabel, fgYellow)
}

func (rn *renderer) renderPanel(sb *screenBuf, pb *playback) {
	panelTop := rn.termH - rn.panelH + 1

	// Clear all panel rows to prevent stale content
	for row := panelTop; row <= rn.termH; row++ {
		sb.write(moveTo(row, 1))
		sb.write(esc + "2K")
	}

	// Separator line
	sb.write(moveTo(panelTop, 1))
	sb.write(fgDimWhite)
	sb.write(strings.Repeat("\u2550", rn.termW)) // double horizontal box-drawing char
	sb.write(fgReset)

	// Turn info
	sb.write(moveTo(panelTop+1, 2))
	sb.write(fgBoldWht)
	turnDisplay := pb.turnIdx + 1
	if turnDisplay < 0 {
		turnDisplay = 0
	}
	sb.writef("Turn: %d / %d", turnDisplay, len(rn.parsed.Turns))
	sb.write(fgReset)

	// Ant count
	sb.write(moveTo(panelTop+1, 30))
	sb.write(fgBoldWht)
	sb.writef("Ants: %d", rn.parsed.AntCount)
	sb.write(fgReset)

	// Mode and speed
	sb.write(moveTo(panelTop+2, 2))
	modeStr := "AUTO"
	switch pb.mode {
	case modeStep:
		modeStr = "STEP"
	case modeSlider:
		modeStr = "SLIDER"
	}
	if pb.paused && pb.mode == modeAutoPlay {
		modeStr = "PAUSED"
	}
	sb.write(fgCyan)
	sb.writef("Mode: %-8s", modeStr)
	sb.write(fgReset)

	sb.write(moveTo(panelTop+2, 30))
	sb.write(fgCyan)
	mult := math.Round(800.0/float64(pb.speed)*10) / 10
	sb.writef("Speed: x%g", mult)
	sb.write(fgReset)

	// Legend
	sb.write(moveTo(panelTop+3, 2))
	sb.write(fgGreen)
	sb.write("\u25C9")
	sb.write(fgDimWhite)
	sb.write("=Start  ")
	sb.write(fgRed)
	sb.write("\u25CE")
	sb.write(fgDimWhite)
	sb.write("=End  ")
	sb.write(fgBrown)
	sb.write("\u25CF")
	sb.write(fgDimWhite)
	sb.write("=Room  ")
	sb.write(fgYellow)
	sb.write("\u25B8Ln")
	sb.write(fgDimWhite)
	sb.write("=Ant  ")
	sb.write(fgDkGreen)
	sb.write("───")
	sb.write(fgDimWhite)
	sb.write("=Tunnel")
	sb.write(fgReset)

	// Controls help
	sb.write(moveTo(panelTop+4, 2))
	sb.write(fgDimWhite)
	sb.write("[Space] Pause/Resume   [+/-] Speed   [p] Mode: Auto/Step/Slider")
	sb.write(fgReset)

	sb.write(moveTo(panelTop+5, 2))
	sb.write(fgDimWhite)
	sb.write("[\u2190 Left] Prev Turn   [\u2192 Right] Next Turn   [r] Reset   [q] Quit")
	sb.write(fgReset)
}

// ---------------------------------------------------------------------------
// Input reader (non-blocking key reader)
// ---------------------------------------------------------------------------

type keyEvent int

const (
	keyNone keyEvent = iota
	keyQuit
	keySpace
	keyPlus
	keyMinus
	keyP
	keyR
	keyRight
	keyLeft
)

// openTTY opens a direct handle to the terminal for keyboard input,
// bypassing stdin which may be consumed by a pipe.
func openTTY() (*os.File, error) {
	// Try /dev/tty first (works on MINGW64, Linux, macOS)
	f, err := os.Open("/dev/tty")
	if err == nil {
		return f, nil
	}
	// Fallback: try CONIN$ (native Windows)
	f, err = os.Open("CONIN$")
	if err == nil {
		return f, nil
	}
	// Last resort: use stdin directly (may not work if piped)
	return os.Stdin, nil
}

// readKeys reads raw bytes from ttyReader and sends key events on a channel.
func readKeys(ch chan<- keyEvent, done <-chan struct{}) {
	reader := ttyReader
	if reader == nil {
		reader = os.Stdin
	}
	buf := make([]byte, 8)
	for {
		select {
		case <-done:
			return
		default:
		}

		n, err := reader.Read(buf)
		if err != nil || n == 0 {
			// If EOF (pipe exhausted), sleep briefly to avoid spin
			time.Sleep(50 * time.Millisecond)
			continue
		}

		for i := 0; i < n; {
			b := buf[i]
			switch {
			case b == 'q' || b == 'Q' || b == 3: // q or Ctrl-C
				ch <- keyQuit
				i++
			case b == ' ':
				ch <- keySpace
				i++
			case b == '+' || b == '=':
				ch <- keyPlus
				i++
			case b == '-' || b == '_':
				ch <- keyMinus
				i++
			case b == 'p' || b == 'P':
				ch <- keyP
				i++
			case b == 'r' || b == 'R':
				ch <- keyR
				i++
			case b == 27 && i+2 < n && buf[i+1] == '[': // ESC sequence
				switch buf[i+2] {
				case 'C': // Right arrow
					ch <- keyRight
				case 'D': // Left arrow
					ch <- keyLeft
				}
				i += 3
			default:
				i++
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Loading indicator
// ---------------------------------------------------------------------------

func showLoading() {
	fmt.Print(clearScreen)
	fmt.Print(moveTo(1, 1))
	fmt.Print(fgCyan)
	fmt.Print("  lem-in TUI Visualizer")
	fmt.Print(fgReset)
	fmt.Print(moveTo(3, 2))
	fmt.Print(fgDimWhite)
	fmt.Print("Receiving data...")
	fmt.Print(fgReset)
	fmt.Print(moveTo(4, 2))
	fmt.Print(fgDimWhite)
	fmt.Print("(pipe lem-in output: ./lem-in file | ./visualizer-tui)")
	fmt.Print(fgReset)
}

// showCenteredError displays an error message centered in the terminal and
// waits for the user to press any key before returning.
func showCenteredError(msg string) {
	cols, rows := getTerminalSize()

	fmt.Print(clearScreen)

	// Center vertically and horizontally
	row := rows / 2
	col := (cols - len(msg)) / 2
	if col < 1 {
		col = 1
	}

	fmt.Print(moveTo(row, col))
	fmt.Print(fgRed)
	fmt.Print(msg)
	fmt.Print(fgReset)

	// Hint below
	hint := "Press any key to exit"
	hintCol := (cols - len(hint)) / 2
	if hintCol < 1 {
		hintCol = 1
	}
	fmt.Print(moveTo(row+2, hintCol))
	fmt.Print(fgDimWhite)
	fmt.Print(hint)
	fmt.Print(fgReset)

	// Wait for a keypress
	enableRawMode()
	tty, _ := openTTY()
	if tty != nil {
		buf := make([]byte, 1)
		_, _ = tty.Read(buf)
		if tty != os.Stdin {
			tty.Close()
		}
	}
	disableRawMode()
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	// Switch to alternate screen buffer early so all output (including
	// the loading screen) goes to the alt buffer. When we exit and
	// restore the main buffer the terminal is left clean.
	fmt.Print(enableAltBuf)
	fmt.Print(hideCursor)

	// Idempotent cleanup that restores the terminal.
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			disableRawMode()
			fmt.Print(showCursor)
			fmt.Print(disableAltBuf)
			fmt.Print(fgReset)
			if ttyReader != nil && ttyReader != os.Stdin {
				ttyReader.Close()
			}
		})
	}
	defer cleanup()

	// Dedicated goroutine: force-exit on any SIGINT (works at any phase).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()

	// Show loading while reading stdin
	showLoading()

	// Read all stdin (SIGINT during read is handled by the goroutine above).
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read stdin: %v\n", err)
		return
	}

	input := string(data)
	if len(strings.TrimSpace(input)) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: no input received on stdin\n")
		return
	}

	// Parse
	parsed, err := format.ParseOutput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return
	}

	// Handle error output from solver: display centered in terminal
	if parsed.Error != "" {
		showCenteredError(parsed.Error)
		return
	}

	// Get terminal size
	termW, termH := getTerminalSize()

	// Open TTY for keyboard input (stdin is consumed by the pipe)
	tty, err := openTTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot open terminal for input: %v\n", err)
		return
	}
	ttyReader = tty

	// Enable raw mode for keyboard input
	enableRawMode()

	// Initialize state
	pb := newPlayback()
	ants := newAntState(parsed.AntCount, parsed.StartName)
	rend := newRenderer(parsed, termW, termH)
	sb := &screenBuf{}

	// Key input channel
	keyCh := make(chan keyEvent, 16)
	doneCh := make(chan struct{})
	defer close(doneCh)
	go readKeys(keyCh, doneCh)

	// Snapshot function: recompute ant positions from scratch up to turnIdx
	recomputeAnts := func(targetTurn int) *antState {
		as := newAntState(parsed.AntCount, parsed.StartName)
		for i := 0; i <= targetTurn && i < len(parsed.Turns); i++ {
			as.applyTurn(parsed.Turns[i])
		}
		return as
	}

	// finishAnimation snaps the current animation to its end state.
	finishAnimation := func() {
		if !pb.animating {
			return
		}
		pb.animating = false
		pb.entries = nil
		pb.preAnts = nil
	}

	// reverseAnimation smoothly reverses the current animation from its
	// current midpoint position, so ants travel back the way they came.
	reverseAnimation := func() {
		if !pb.animating || pb.preAnts == nil {
			return
		}
		// Swap ants (destination) and preAnts (source)
		oldAnts := ants
		ants = pb.preAnts
		pb.preAnts = oldAnts
		// Reverse each animation path
		for i := range pb.entries {
			p := pb.entries[i].path
			for l, r := 0, len(p)-1; l < r; l, r = l+1, r-1 {
				p[l], p[r] = p[r], p[l]
			}
		}
		// Mirror the frame so the visual position stays the same
		pb.animFrame = pb.animTotal - 1 - pb.animFrame
		if pb.animFrame < 0 {
			pb.animFrame = 0
		}
		// Adjust turnIdx and flip direction
		if pb.animForward {
			pb.turnIdx--
		} else {
			pb.turnIdx++
		}
		pb.animForward = !pb.animForward
	}

	// startTransition begins a smooth animation for the next/prev turn.
	startTransitionForward := func() {
		nextTurn := pb.turnIdx + 1
		if nextTurn >= len(parsed.Turns) {
			return
		}
		pb.preAnts = ants.clone()
		pb.entries = rend.computeTransition(ants, parsed.Turns[nextTurn])
		pb.turnIdx = nextTurn
		ants.applyTurn(parsed.Turns[nextTurn])
		pb.animFrame = 0
		pb.animTotal = pb.animDuration()
		pb.animForward = true
		pb.animating = true
	}

	startTransitionBackward := func() {
		if pb.turnIdx < 0 {
			return
		}
		pb.preAnts = ants.clone()
		// We compute the reverse: ants move from current positions back to previous
		prevTurn := pb.turnIdx - 1
		prevAnts := recomputeAnts(prevTurn)
		// Build entries: for each ant that moved in the current turn, animate backward
		// along the same L-shaped route the forward animation used (reversed).
		moves := parsed.Turns[pb.turnIdx]
		entries := make([]animEntry, 0, len(moves))
		for _, m := range moves {
			// Compute the forward path (source → destination) and reverse it
			fwdSrc := prevAnts.positions[m.AntID]
			path := rend.computePath(fwdSrc, m.RoomName)
			for l, r := 0, len(path)-1; l < r; l, r = l+1, r-1 {
				path[l], path[r] = path[r], path[l]
			}
			entries = append(entries, animEntry{antID: m.AntID, path: path})
		}
		pb.turnIdx = prevTurn
		ants = prevAnts
		pb.entries = entries
		pb.animFrame = 0
		pb.animTotal = pb.animDuration()
		pb.animForward = false
		pb.animating = true
	}

	// Step-mode per-direction hold suppression: each direction has its own
	// lock so that holding Right doesn't break when Left reverses (and
	// vice-versa). A lock stays active while events keep arriving within
	// stepLockGap; a gap longer than that is interpreted as a new key press.
	var stepFwdActive, stepBwdActive bool
	var stepFwdLast, stepBwdLast time.Time
	const stepLockGap = 500 * time.Millisecond  // held-key suppression during opposite-dir animation
	const stepRepeatGap = 100 * time.Millisecond // post-animation auto-repeat suppression

	// Slider mode: event-driven. Each key event advances the animation
	// by a time-proportional number of frames so that one full transition
	// takes exactly pb.speed milliseconds regardless of auto-repeat rate.
	var sliderLastEvent time.Time
	var sliderFrac float64                          // fractional frame accumulator
	const sliderNewTapGap = 150 * time.Millisecond  // gap longer than this = new tap (not held key)

	// Initial render
	rend.render(sb, ants, pb)

	// Frame-based ticker (~30fps)
	const frameDuration = 33 * time.Millisecond
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	elapsed := 0         // accumulated ms for auto-advance
	replayWaiting := false // waiting to auto-replay after last turn
	replayElapsed := 0     // ms accumulated during replay wait

	running := true
	for running {
		select {
		case key := <-keyCh:
			switch key {
			case keyQuit:
				running = false

			case keySpace:
				if pb.mode == modeAutoPlay {
					pb.paused = !pb.paused
					replayWaiting = false
				}
				sliderLastEvent = time.Time{}
				sliderFrac = 0
				finishAnimation()
				rend.render(sb, ants, pb)

			case keyPlus:
				pb.faster()
				rend.render(sb, ants, pb)

			case keyMinus:
				pb.slower()
				rend.render(sb, ants, pb)

			case keyP:
				switch pb.mode {
				case modeAutoPlay:
					pb.mode = modeStep
				case modeStep:
					pb.mode = modeSlider
				case modeSlider:
					pb.mode = modeAutoPlay
				}
				pb.paused = false
				replayWaiting = false
				stepFwdActive = false
				stepBwdActive = false
				sliderLastEvent = time.Time{}
				sliderFrac = 0
				finishAnimation()
				rend.render(sb, ants, pb)

			case keyR:
				finishAnimation()
				replayWaiting = false
				stepFwdActive = false
				stepBwdActive = false
				sliderLastEvent = time.Time{}
				sliderFrac = 0
				pb.turnIdx = -1
				pb.paused = false
				ants = newAntState(parsed.AntCount, parsed.StartName)
				rend.render(sb, ants, pb)

			case keyRight:
				if pb.mode == modeAutoPlay {
					break
				}
				if pb.mode == modeSlider {
					if pb.animating && !pb.animForward {
						reverseAnimation()
						sliderLastEvent = time.Time{}
						sliderFrac = 0
					}
					if !pb.animating {
						startTransitionForward()
					}
					if pb.animating {
						now := time.Now()
						var advance int
						if sliderLastEvent.IsZero() || now.Sub(sliderLastEvent) > sliderNewTapGap {
							advance = 1
							sliderFrac = 0
						} else {
							elapsedMs := float64(now.Sub(sliderLastEvent)) / float64(time.Millisecond)
							sliderFrac += elapsedMs * float64(pb.animTotal) / float64(pb.speed)
							advance = int(sliderFrac)
							sliderFrac -= float64(advance)
						}
						sliderLastEvent = now
						if advance > 0 {
							pb.animFrame += advance
							for pb.animFrame >= pb.animTotal {
								pb.animFrame -= pb.animTotal
								finishAnimation()
								startTransitionForward()
								if !pb.animating {
									break
								}
							}
						}
					}
					rend.render(sb, ants, pb)
					break
				}
				if pb.mode == modeStep && stepFwdActive {
					now := time.Now()
					if pb.animating {
						if pb.animForward {
							// Same direction: always suppress
							stepFwdLast = now
							break
						}
						// Opposite direction: suppress if recent (held key)
						if now.Sub(stepFwdLast) < stepLockGap {
							stepFwdLast = now
							break
						}
						stepFwdActive = false
					} else {
						// Post-animation: only suppress rapid auto-repeat
						if now.Sub(stepFwdLast) < stepRepeatGap {
							stepFwdLast = now
							break
						}
						stepFwdActive = false
					}
				}
				if pb.animating {
					if !pb.animForward {
						reverseAnimation()
						if pb.mode == modeStep {
							stepFwdActive = true
							stepFwdLast = time.Now()
							if stepBwdActive {
								stepBwdLast = time.Now()
							}
						}
						rend.render(sb, ants, pb)
						break
					}
					finishAnimation()
				}
				elapsed = 0
				startTransitionForward()
				if pb.mode == modeStep && pb.animating {
					stepFwdActive = true
					stepFwdLast = time.Now()
				}
				rend.render(sb, ants, pb)

			case keyLeft:
				if pb.mode == modeAutoPlay {
					break
				}
				if pb.mode == modeSlider {
					if pb.animating && pb.animForward {
						reverseAnimation()
						sliderLastEvent = time.Time{}
						sliderFrac = 0
					}
					if !pb.animating {
						startTransitionBackward()
					}
					if pb.animating {
						now := time.Now()
						var advance int
						if sliderLastEvent.IsZero() || now.Sub(sliderLastEvent) > sliderNewTapGap {
							advance = 1
							sliderFrac = 0
						} else {
							elapsedMs := float64(now.Sub(sliderLastEvent)) / float64(time.Millisecond)
							sliderFrac += elapsedMs * float64(pb.animTotal) / float64(pb.speed)
							advance = int(sliderFrac)
							sliderFrac -= float64(advance)
						}
						sliderLastEvent = now
						if advance > 0 {
							pb.animFrame += advance
							for pb.animFrame >= pb.animTotal {
								pb.animFrame -= pb.animTotal
								finishAnimation()
								startTransitionBackward()
								if !pb.animating {
									break
								}
							}
						}
					}
					rend.render(sb, ants, pb)
					break
				}
				if pb.mode == modeStep && stepBwdActive {
					now := time.Now()
					if pb.animating {
						if !pb.animForward {
							// Same direction (backward): always suppress
							stepBwdLast = now
							break
						}
						// Opposite direction: suppress if recent (held key)
						if now.Sub(stepBwdLast) < stepLockGap {
							stepBwdLast = now
							break
						}
						stepBwdActive = false
					} else {
						// Post-animation: only suppress rapid auto-repeat
						if now.Sub(stepBwdLast) < stepRepeatGap {
							stepBwdLast = now
							break
						}
						stepBwdActive = false
					}
				}
				if pb.animating {
					if pb.animForward {
						reverseAnimation()
						if pb.mode == modeStep {
							stepBwdActive = true
							stepBwdLast = time.Now()
							if stepFwdActive {
								stepFwdLast = time.Now()
							}
						}
						rend.render(sb, ants, pb)
						break
					}
					finishAnimation()
				}
				elapsed = 0
				startTransitionBackward()
				if pb.mode == modeStep && pb.animating {
					stepBwdActive = true
					stepBwdLast = time.Now()
				}
				rend.render(sb, ants, pb)
			}

		case <-ticker.C:
			if pb.animating {
				if pb.mode == modeSlider {
					// Slider: key events drive frames, ticker does nothing
				} else {
					pb.animFrame++
					if pb.animFrame >= pb.animTotal {
						finishAnimation()
						if pb.mode == modeAutoPlay && !pb.paused {
							startTransitionForward()
							if !pb.animating {
								replayWaiting = true
								replayElapsed = 0
							}
						}
					}
					rend.render(sb, ants, pb)
				}
			} else if replayWaiting && pb.mode == modeAutoPlay && !pb.paused {
				replayElapsed += int(frameDuration / time.Millisecond)
				if replayElapsed >= 1000 {
					replayWaiting = false
					replayElapsed = 0
					pb.turnIdx = -1
					elapsed = 0
					ants = newAntState(parsed.AntCount, parsed.StartName)
					startTransitionForward()
					rend.render(sb, ants, pb)
				}
			} else if pb.mode == modeAutoPlay && !pb.paused {
				elapsed += int(frameDuration / time.Millisecond)
				if elapsed >= 800 {
					elapsed = 0
					startTransitionForward()
					if !pb.animating {
						// No more turns — start replay countdown
						replayWaiting = true
						replayElapsed = 0
					}
					rend.render(sb, ants, pb)
				}
			}
		}
	}

	// Force-exit after cleanup to prevent go run from hanging on MINGW64
	// after catching SIGINT. cleanup is idempotent via sync.Once.
	cleanup()
	os.Exit(0)
}
