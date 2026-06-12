package ui

import (
	"strings"
	"testing"
)

func TestDrawHeader(t *testing.T) {
	app, sim := newTestApp(t)
	app.draw()
	text := screenText(sim)
	for _, want := range []string{"Mem", "Swp", "Tasks:", "Load average:", "Uptime:"} {
		if !strings.Contains(text, want) {
			t.Errorf("header missing %q; screen:\n%s", want, text)
		}
	}
}
