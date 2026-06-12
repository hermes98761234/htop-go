package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestHelpScreen(t *testing.T) {
	app, sim := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
	if app.mode != ModeHelp {
		t.Fatal("F1 did not open help")
	}
	app.draw()
	text := screenText(sim)
	for _, want := range []string{"tree view", "sort by", "quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q", want)
		}
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
	if app.mode != ModeNormal {
		t.Error("key press did not close help")
	}
}
