package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

func TestBuildTreeRows(t *testing.T) {
	procs := []proc.Process{
		{PID: 4, PPID: 2},
		{PID: 2, PPID: 1},
		{PID: 1, PPID: 0},
		{PID: 3, PPID: 1},
		{PID: 9, PPID: 999}, // orphan: parent not in snapshot -> root
	}
	rows := buildTreeRows(procs, SortPID, false)
	wantPIDs := []int{1, 2, 4, 3, 9}
	wantPrefix := []string{"", "├─ ", "│  └─ ", "└─ ", ""}
	if len(rows) != len(wantPIDs) {
		t.Fatalf("got %d rows, want %d", len(rows), len(wantPIDs))
	}
	for i := range rows {
		if rows[i].Proc.PID != wantPIDs[i] {
			t.Errorf("row %d PID = %d, want %d", i, rows[i].Proc.PID, wantPIDs[i])
		}
		if rows[i].Prefix != wantPrefix[i] {
			t.Errorf("row %d Prefix = %q, want %q", i, rows[i].Prefix, wantPrefix[i])
		}
	}
}

func TestTreeToggleKey(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF5, 0, tcell.ModNone))
	if !app.treeMode {
		t.Fatal("F5 did not enable tree mode")
	}
	// PID 1 (init) must be first in tree mode on a real system.
	if len(app.rows) == 0 || app.rows[0].Proc.PID != 1 {
		t.Errorf("first tree row PID = %d, want 1", app.rows[0].Proc.PID)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyF5, 0, tcell.ModNone))
	if app.treeMode {
		t.Error("second F5 did not disable tree mode")
	}
}
