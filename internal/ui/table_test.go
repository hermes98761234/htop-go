package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPad(t *testing.T) {
	if got := pad("ab", 5, true); got != "   ab" {
		t.Errorf("right pad: %q", got)
	}
	if got := pad("ab", 5, false); got != "ab   " {
		t.Errorf("left pad: %q", got)
	}
	if got := pad("abcdef", 3, false); got != "abc" {
		t.Errorf("truncate: %q", got)
	}
}

func TestTableDrawAndNavigate(t *testing.T) {
	app, sim := newTestApp(t)
	app.draw()
	text := screenText(sim)
	for _, want := range []string{"PID", "USER", "CPU%", "Command"} {
		if !strings.Contains(text, want) {
			t.Errorf("table header missing %q", want)
		}
	}
	sel := app.table.Sel
	app.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	if app.table.Sel != sel+1 {
		t.Errorf("KeyDown: Sel = %d, want %d", app.table.Sel, sel+1)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone))
	if app.table.Sel != 0 {
		t.Errorf("KeyHome: Sel = %d, want 0", app.table.Sel)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone))
	if app.table.Sel != len(app.rows)-1 {
		t.Errorf("KeyEnd: Sel = %d, want %d", app.table.Sel, len(app.rows)-1)
	}
}
