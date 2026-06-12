package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

// newTestApp builds an App on a 120x40 simulation screen with a real
// scan already loaded (tests run on Linux, /proc is available).
func newTestApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	sim.SetSize(120, 40)
	app := NewApp(sim, proc.NewScanner(), 1500*time.Millisecond)
	app.refresh()
	return app, sim
}

// screenText flattens the simulation screen into one string for assertions.
func screenText(sim tcell.SimulationScreen) string {
	cells, w, h := sim.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 {
				b.WriteRune(c.Runes[0])
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestDrawSmoke(t *testing.T) {
	app, sim := newTestApp(t)
	app.draw()
	text := screenText(sim)
	if !strings.Contains(text, "Quit") {
		t.Errorf("function bar not drawn; screen:\n%s", text)
	}
	if len(app.rows) == 0 {
		t.Error("no rows after refresh")
	}
}

func TestQuitKeys(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))
	if !app.quit {
		t.Error("q did not quit")
	}
	app.quit = false
	app.handleKey(tcell.NewEventKey(tcell.KeyF10, 0, tcell.ModNone))
	if !app.quit {
		t.Error("F10 did not quit")
	}
}
