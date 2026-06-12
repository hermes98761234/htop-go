package ui

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

// Mode is the current input mode of the app.
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeFilter
	ModeSignals
	ModeHelp
)

// Row is one display row: a process plus its tree prefix (empty when flat).
type Row struct {
	Proc   proc.Process
	Prefix string
}

// App owns the screen, the scanner, and all UI state.
type App struct {
	screen   tcell.Screen
	scanner  *proc.Scanner
	delay    time.Duration
	snap     *proc.Snapshot
	rows     []Row
	table    Table
	sortBy   SortBy
	sortDesc bool
	treeMode bool
	mode     Mode
	input    string // text being typed in search/filter mode
	filter   string // committed filter (empty = none)
	sigSel   int    // selected entry in the F9 signal menu
	status   string // transient error/status line
	quit     bool
}

// NewApp wires a ready-to-run App. The screen must already be Init()ed.
func NewApp(screen tcell.Screen, scanner *proc.Scanner, delay time.Duration) *App {
	return &App{
		screen:   screen,
		scanner:  scanner,
		delay:    delay,
		sortBy:   SortCPU,
		sortDesc: true,
	}
}

// setSort switches the sort key and resets to its natural direction.
func (a *App) setSort(by SortBy) {
	a.sortBy = by
	a.sortDesc = by.defaultDesc()
	a.rebuild()
}

// Run drives the event loop until quit. It calls screen.Fini on exit.
func (a *App) Run() error {
	defer a.screen.Fini()
	a.refresh()
	events := make(chan tcell.Event, 16)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				close(events)
				return
			}
			events <- ev
		}
	}()
	ticker := time.NewTicker(a.delay)
	defer ticker.Stop()
	a.draw()
	for !a.quit {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			a.handleEvent(ev)
		case <-ticker.C:
			a.refresh()
		}
		a.draw()
	}
	return nil
}

// refresh rescans /proc and rebuilds the display rows.
func (a *App) refresh() {
	snap, err := a.scanner.Scan()
	if err != nil {
		a.status = err.Error()
		return
	}
	a.snap = snap
	a.rebuild()
}

// rebuild recomputes a.rows from a.snap: filter, then tree or sort, then
// keep the selection on the same PID when possible. An active filter
// bypasses tree mode (flat list), like htop.
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	selPID := 0
	if len(a.rows) > 0 && a.table.Sel < len(a.rows) {
		selPID = a.rows[a.table.Sel].Proc.PID
	}
	needle := strings.ToLower(a.filter)
	procs := make([]proc.Process, 0, len(a.snap.Procs))
	for _, p := range a.snap.Procs {
		if needle != "" && !strings.Contains(strings.ToLower(p.Cmdline), needle) {
			continue
		}
		procs = append(procs, p)
	}
	if a.treeMode && needle == "" {
		a.rows = buildTreeRows(procs, a.sortBy, a.sortDesc)
	} else {
		sortProcs(procs, a.sortBy, a.sortDesc)
		rows := make([]Row, 0, len(procs))
		for _, p := range procs {
			rows = append(rows, Row{Proc: p})
		}
		a.rows = rows
	}
	a.table.ClampTo(len(a.rows))
	if selPID != 0 {
		for i := range a.rows {
			if a.rows[i].Proc.PID == selPID {
				a.table.Sel = i
				break
			}
		}
	}
}

func (a *App) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventKey:
		a.handleKey(ev)
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	a.status = ""
	switch a.mode {
	case ModeNormal:
		a.handleNormalKey(ev)
	case ModeSearch:
		a.handleSearchKey(ev)
	case ModeFilter:
		a.handleFilterKey(ev)
	case ModeSignals:
		a.handleSignalsKey(ev)
	}
}

func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	case ev.Key() == tcell.KeyF3:
		a.mode = ModeSearch
		a.input = ""
	case ev.Key() == tcell.KeyF4:
		a.mode = ModeFilter
		a.input = a.filter
	case ev.Key() == tcell.KeyF5:
		a.treeMode = !a.treeMode
		a.rebuild()
	case ev.Key() == tcell.KeyF7:
		a.renice(-1)
	case ev.Key() == tcell.KeyF8:
		a.renice(1)
	case ev.Key() == tcell.KeyF9:
		if len(a.rows) > 0 {
			a.sigSel = 3 // SIGTERM
			a.mode = ModeSignals
		}
	case ev.Rune() == 'P':
		a.setSort(SortCPU)
	case ev.Rune() == 'M':
		a.setSort(SortMem)
	case ev.Rune() == 'T':
		a.setSort(SortTime)
	case ev.Rune() == 'N':
		a.setSort(SortPID)
	case ev.Rune() == 'I':
		a.sortDesc = !a.sortDesc
		a.rebuild()
	}
}

// draw paints the whole screen.
func (a *App) draw() {
	a.screen.Clear()
	w, h := a.screen.Size()
	a.drawMain(w, h)
	if a.mode == ModeSignals {
		a.drawSignalMenu(a.headerHeight() + 1)
	}
	a.drawBottom(w, h)
	a.screen.Show()
}

// headerHeight is the row count drawHeader uses (without drawing).
func (a *App) headerHeight() int {
	if a.snap == nil {
		return 0
	}
	return (len(a.snap.CPUs)+1)/2 + 3
}

// drawMain paints the header and the process table.
func (a *App) drawMain(w, h int) {
	headerH := a.drawHeader(w)
	tableH := h - headerH - 1 // one row reserved for the bottom bar
	if tableH < 2 {
		return
	}
	a.table.Draw(a.screen, w, headerH, tableH, a.rows, a.sortBy.columnIndex())
}

// fnBarItems is the bottom function-key bar, htop style.
var fnBarItems = []struct{ Key, Label string }{
	{"F1", "Help"}, {"F3", "Search"}, {"F4", "Filter"}, {"F5", "Tree"},
	{"F7", "Nice-"}, {"F8", "Nice+"}, {"F9", "Kill"}, {"F10", "Quit"},
}

// drawBottom paints the bottom row: search/filter prompt when typing,
// else the status line if set, else the fn-key bar.
func (a *App) drawBottom(w, h int) {
	y := h - 1
	switch a.mode {
	case ModeSearch:
		drawString(a.screen, 0, y, styleDefault, "Search: "+a.input)
		return
	case ModeFilter:
		drawString(a.screen, 0, y, styleDefault, "Filter: "+a.input)
		return
	}
	if a.status != "" {
		drawString(a.screen, 0, y, styleStatus, a.status)
		return
	}
	x := 0
	for _, item := range fnBarItems {
		x = drawString(a.screen, x, y, styleFnKey, item.Key)
		x = drawString(a.screen, x, y, styleFnLabel, item.Label+" ")
	}
}
