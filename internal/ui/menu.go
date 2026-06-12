package ui

import (
	"fmt"
	"syscall"

	"github.com/gdamore/tcell/v2"
)

// signalEntries is the F9 menu, a subset of htop's list.
// Index 3 (SIGTERM) is the default selection.
var signalEntries = []struct {
	Num  int
	Name string
}{
	{1, "SIGHUP"}, {2, "SIGINT"}, {9, "SIGKILL"},
	{15, "SIGTERM"}, {18, "SIGCONT"}, {19, "SIGSTOP"},
}

// drawSignalMenu paints the menu box at the left edge, starting at row y0.
func (a *App) drawSignalMenu(y0 int) {
	const boxW = 16
	drawString(a.screen, 0, y0, styleMenuBox, pad("Send signal:", boxW, false))
	for i, e := range signalEntries {
		style := styleMenuBox
		if i == a.sigSel {
			style = styleMenuSel
		}
		label := fmt.Sprintf("%3d %s", e.Num, e.Name)
		drawString(a.screen, 0, y0+1+i, style, pad(label, boxW, false))
	}
}

// handleSignalsKey drives the F9 menu.
func (a *App) handleSignalsKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		a.mode = ModeNormal
	case tcell.KeyUp:
		if a.sigSel > 0 {
			a.sigSel--
		}
	case tcell.KeyDown:
		if a.sigSel < len(signalEntries)-1 {
			a.sigSel++
		}
	case tcell.KeyEnter:
		a.mode = ModeNormal
		if a.table.Sel < len(a.rows) {
			p := a.rows[a.table.Sel].Proc
			sig := signalEntries[a.sigSel]
			if err := syscall.Kill(p.PID, syscall.Signal(sig.Num)); err != nil {
				a.status = fmt.Sprintf("Cannot send %s to PID %d: %v", sig.Name, p.PID, err)
				return
			}
			a.refresh()
		}
	}
}

// renice changes the selected process's nice value by delta.
// Lowering nice (delta < 0) usually requires root; errors go to the status line.
func (a *App) renice(delta int) {
	if a.table.Sel >= len(a.rows) {
		return
	}
	p := a.rows[a.table.Sel].Proc
	newNice := int(p.Nice) + delta
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, p.PID, newNice); err != nil {
		a.status = fmt.Sprintf("Cannot renice PID %d: %v", p.PID, err)
		return
	}
	a.refresh()
}
