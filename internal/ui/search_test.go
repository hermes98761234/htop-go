package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

func fixtureRows() []Row {
	return []Row{
		{Proc: proc.Process{PID: 1, Cmdline: "/sbin/init splash"}},
		{Proc: proc.Process{PID: 2, Cmdline: "nginx: worker"}},
		{Proc: proc.Process{PID: 3, Cmdline: "/usr/bin/NGINX-helper"}},
	}
}

func TestFindNext(t *testing.T) {
	rows := fixtureRows()
	if got := findNext(rows, "nginx", 0); got != 1 {
		t.Errorf("findNext from 0 = %d, want 1", got)
	}
	// Case-insensitive and wrapping.
	if got := findNext(rows, "NgInX", 2); got != 2 {
		t.Errorf("findNext from 2 = %d, want 2", got)
	}
	if got := findNext(rows, "init", 1); got != 0 {
		t.Errorf("findNext wrap = %d, want 0", got)
	}
	if got := findNext(rows, "nomatch", 0); got != -1 {
		t.Errorf("findNext nomatch = %d, want -1", got)
	}
	if got := findNext(rows, "", 0); got != -1 {
		t.Errorf("findNext empty = %d, want -1", got)
	}
}

func TestFilterMode(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF4, 0, tcell.ModNone))
	if app.mode != ModeFilter {
		t.Fatal("F4 did not enter filter mode")
	}
	for _, r := range "zzz-no-such-process-zzz" {
		app.handleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	if len(app.rows) != 0 {
		t.Errorf("bogus filter left %d rows, want 0", len(app.rows))
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if app.mode != ModeNormal || app.filter != "" {
		t.Error("Esc did not clear the filter")
	}
	if len(app.rows) == 0 {
		t.Error("rows still empty after clearing filter")
	}
}

func TestSearchMode(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF3, 0, tcell.ModNone))
	if app.mode != ModeSearch {
		t.Fatal("F3 did not enter search mode")
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if app.mode != ModeNormal {
		t.Error("Enter did not leave search mode")
	}
}
