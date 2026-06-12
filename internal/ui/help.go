package ui

import "github.com/gdamore/tcell/v2"

var helpLines = []string{
	"htop-go — interactive process viewer for Linux",
	"",
	"  Arrows, PgUp/PgDn, Home/End   navigate the process list",
	"  P M T N    sort by CPU%, MEM%, TIME+, PID",
	"  I          invert the sort order",
	"  F5         toggle tree view",
	"  F3         incremental search (Enter/Esc to leave)",
	"  F4         filter the list (Enter keeps it, Esc clears it)",
	"  F7 / F8    decrease / increase nice value",
	"  F9         send a signal to the selected process",
	"  F1 or h    this help screen",
	"  F10 or q   quit",
	"",
	"Press any key to return.",
}

// drawHelp paints the full-screen help.
func (a *App) drawHelp(w, h int) {
	for i, line := range helpLines {
		if i >= h {
			break
		}
		drawString(a.screen, 0, i, styleDefault, line)
	}
}

// handleHelpKey leaves help on any key.
func (a *App) handleHelpKey(ev *tcell.EventKey) {
	a.mode = ModeNormal
}
