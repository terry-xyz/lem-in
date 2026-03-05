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

	// Colors (foreground)
	fgReset    = esc + "0m"
	fgGreen    = esc + "32m"
	fgRed      = esc + "31m"
	fgYellow   = esc + "33m"
	fgDkGreen  = esc + "38;5;22m"  // dark green (256-color)
	fgBrown    = esc + "38;5;130m" // brown/dark orange (256-color)
	fgWhite    = esc + "37m"
	fgCyan     = esc + "36m"
	fgBoldWht  = esc + "1;37m"
	fgDimWhite = esc + "2;37m"

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

func enableRawMode() {
	// Save current settings
	out, err := exec.Command("stty", "-g").Output()
	if err == nil {
		sttyOriginal = strings.TrimSpace(string(out))
	}
	// Set raw mode: no echo, no canonical mode, read 1 char at a time
	_ = exec.Command("stty", "raw", "-echo", "min", "1", "time", "0").Run()
}

func disableRawMode() {
	if sttyOriginal != "" {
		_ = exec.Command("stty", sttyOriginal).Run()
	} else {
		_ = exec.Command("stty", "sane").Run()
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
// Bresenham's line drawing
// ---------------------------------------------------------------------------

func drawLine(cv *canvas, x0, y0, x1, y1 int, ch rune, fg string) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy

	for {
		// Don't overwrite room cells -- only draw on empty space
		existing := cv.get(x0, y0)
		if existing.ch == ' ' {
			// Choose directional line character
			lineCh := lineChar(x0, y0, x1, y1, ch)
			cv.set(x0, y0, lineCh, fg)
		}

		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// lineChar picks a Unicode box-drawing character for tunnel segments.
func lineChar(x0, y0, x1, y1 int, fallback rune) rune {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	if dx == 0 {
		return '│' // U+2502 vertical
	}
	if dy == 0 {
		return '─' // U+2500 horizontal
	}
	// Nearly horizontal or nearly vertical
	if dx > dy*2 {
		return '─'
	}
	if dy > dx*2 {
		return '│'
	}
	// Diagonal: determine direction
	goingRight := x1 > x0
	goingDown := y1 > y0
	if goingRight == goingDown {
		return '╲' // U+2572 upper-left to lower-right
	}
	return '╱' // U+2571 upper-right to lower-left
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
)

type playback struct {
	mode     playMode
	paused   bool
	turnIdx  int // -1 = initial state (before any moves)
	speed    int // milliseconds per turn
	minSpeed int
	maxSpeed int
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
	panelH := 8 // rows reserved for info panel at bottom
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

func (rn *renderer) render(sb *screenBuf, ants *antState, pb *playback) {
	cv := newCanvas(rn.canvasW, rn.canvasH)

	// 1. Draw tunnels (lines between linked rooms)
	for _, link := range rn.parsed.Links {
		posA, okA := rn.positions[link[0]]
		posB, okB := rn.positions[link[1]]
		if okA && okB {
			drawLine(cv, posA.screenX, posA.screenY, posB.screenX, posB.screenY, '·', fgDkGreen)
		}
	}

	// 2. Draw rooms
	for _, room := range rn.parsed.Rooms {
		pos, ok := rn.positions[room.Name]
		if !ok {
			continue
		}

		var color string
		var symbol rune
		switch {
		case room.IsStart:
			color = fgGreen
			symbol = 'S'
		case room.IsEnd:
			color = fgRed
			symbol = 'E'
		default:
			color = fgBrown
			symbol = 'O'
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
		cv.setStr(labelX, pos.screenY, label, color)
	}

	// 3. Draw ants at their current positions
	for _, room := range rn.parsed.Rooms {
		pos, ok := rn.positions[room.Name]
		if !ok {
			continue
		}
		antIDs := ants.antsAtRoom(room.Name)
		if len(antIDs) == 0 {
			continue
		}

		// Show ant marker below the room if possible, otherwise above
		antY := pos.screenY + 1
		if antY >= rn.canvasH {
			antY = pos.screenY - 1
		}
		if antY < 0 {
			antY = pos.screenY
		}

		// Build ant label
		var antLabel string
		if len(antIDs) <= 3 {
			parts := make([]string, len(antIDs))
			for i, id := range antIDs {
				parts[i] = fmt.Sprintf("L%d", id)
			}
			antLabel = strings.Join(parts, ",")
		} else {
			antLabel = fmt.Sprintf("L%d..(%d)", antIDs[0], len(antIDs))
		}

		if len(antLabel) > rn.canvasW-2 {
			antLabel = antLabel[:rn.canvasW-2]
		}

		// Center the label under the room
		labelX := pos.screenX - len(antLabel)/2
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

	// Build output
	sb.write(clearScreen)
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
	}
	sb.write(fgReset)

	// Render info panel
	rn.renderPanel(sb, pb)

	sb.flush()
}

func (rn *renderer) renderPanel(sb *screenBuf, pb *playback) {
	panelTop := rn.termH - rn.panelH + 1

	// Separator line
	sb.write(moveTo(panelTop, 1))
	sb.write(fgDimWhite)
	sb.write(strings.Repeat("\u2500", rn.termW)) // horizontal box-drawing char
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
	if pb.mode == modeStep {
		modeStr = "STEP"
	}
	if pb.paused && pb.mode == modeAutoPlay {
		modeStr = "PAUSED"
	}
	sb.write(fgCyan)
	sb.writef("Mode: %-8s", modeStr)
	sb.write(fgReset)

	sb.write(moveTo(panelTop+2, 30))
	sb.write(fgCyan)
	sb.writef("Speed: %dms", pb.speed)
	sb.write(fgReset)

	// Legend
	sb.write(moveTo(panelTop+3, 2))
	sb.write(fgGreen)
	sb.write("S")
	sb.write(fgDimWhite)
	sb.write("=Start  ")
	sb.write(fgRed)
	sb.write("E")
	sb.write(fgDimWhite)
	sb.write("=End  ")
	sb.write(fgBrown)
	sb.write("O")
	sb.write(fgDimWhite)
	sb.write("=Room  ")
	sb.write(fgYellow)
	sb.write("Ln")
	sb.write(fgDimWhite)
	sb.write("=Ant  ")
	sb.write(fgDkGreen)
	sb.write("─│╱╲")
	sb.write(fgDimWhite)
	sb.write("=Tunnel")
	sb.write(fgReset)

	// Controls help
	sb.write(moveTo(panelTop+5, 2))
	sb.write(fgDimWhite)
	sb.write("[Space] Pause/Resume   [+/-] Speed   [p] Toggle Step/Auto")
	sb.write(fgReset)

	sb.write(moveTo(panelTop+6, 2))
	sb.write(fgDimWhite)
	sb.write("[Enter/Right] Next Turn   [Left] Prev Turn   [r] Reset   [q] Quit")
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
	keyEnter
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
			case b == 13 || b == 10: // Enter
				ch <- keyEnter
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

	fmt.Print(enableAltBuf)
	fmt.Print(hideCursor)
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

	fmt.Print(showCursor)
	fmt.Print(disableAltBuf)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	// Show loading while reading stdin
	showLoading()

	// Read all stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	input := string(data)
	if len(strings.TrimSpace(input)) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: no input received on stdin\n")
		os.Exit(1)
	}

	// Parse
	parsed, err := format.ParseOutput(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Handle error output from solver: display centered in terminal
	if parsed.Error != "" {
		showCenteredError(parsed.Error)
		os.Exit(1)
	}

	// Get terminal size
	termW, termH := getTerminalSize()

	// Open TTY for keyboard input (stdin is consumed by the pipe)
	tty, err := openTTY()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot open terminal for input: %v\n", err)
		os.Exit(1)
	}
	ttyReader = tty

	// Set up terminal
	fmt.Print(enableAltBuf)
	fmt.Print(hideCursor)
	enableRawMode()

	// Cleanup on exit
	cleanup := func() {
		disableRawMode()
		fmt.Print(showCursor)
		fmt.Print(disableAltBuf)
		fmt.Print(fgReset)
		if ttyReader != nil && ttyReader != os.Stdin {
			ttyReader.Close()
		}
	}
	defer cleanup()

	// Handle interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

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

	// Initial render
	rend.render(sb, ants, pb)

	// Main loop
	ticker := time.NewTicker(time.Duration(pb.speed) * time.Millisecond)
	defer ticker.Stop()

	running := true
	for running {
		select {
		case <-sigCh:
			running = false

		case key := <-keyCh:
			switch key {
			case keyQuit:
				running = false

			case keySpace:
				if pb.mode == modeAutoPlay {
					pb.paused = !pb.paused
				}
				rend.render(sb, ants, pb)

			case keyPlus:
				pb.faster()
				ticker.Reset(time.Duration(pb.speed) * time.Millisecond)
				rend.render(sb, ants, pb)

			case keyMinus:
				pb.slower()
				ticker.Reset(time.Duration(pb.speed) * time.Millisecond)
				rend.render(sb, ants, pb)

			case keyP:
				if pb.mode == modeAutoPlay {
					pb.mode = modeStep
					pb.paused = false
				} else {
					pb.mode = modeAutoPlay
					pb.paused = false
				}
				rend.render(sb, ants, pb)

			case keyR:
				pb.turnIdx = -1
				pb.paused = false
				ants = newAntState(parsed.AntCount, parsed.StartName)
				rend.render(sb, ants, pb)

			case keyEnter, keyRight:
				// Advance one turn
				if pb.turnIdx+1 < len(parsed.Turns) {
					pb.turnIdx++
					ants.applyTurn(parsed.Turns[pb.turnIdx])
					rend.render(sb, ants, pb)
				}

			case keyLeft:
				// Go back one turn
				if pb.turnIdx >= 0 {
					pb.turnIdx--
					ants = recomputeAnts(pb.turnIdx)
					rend.render(sb, ants, pb)
				}
			}

		case <-ticker.C:
			if pb.mode == modeAutoPlay && !pb.paused {
				if pb.turnIdx+1 < len(parsed.Turns) {
					pb.turnIdx++
					ants.applyTurn(parsed.Turns[pb.turnIdx])
					rend.render(sb, ants, pb)
				} else {
					// Playback finished, auto-pause
					pb.paused = true
					rend.render(sb, ants, pb)
				}
			}
		}
	}
}
