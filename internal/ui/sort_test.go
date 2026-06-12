package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

func pids(procs []proc.Process) []int {
	out := make([]int, len(procs))
	for i, p := range procs {
		out[i] = p.PID
	}
	return out
}

func eq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSortProcs(t *testing.T) {
	fixture := func() []proc.Process {
		return []proc.Process{
			{PID: 3, CPUPercent: 5, ResKB: 100, UTime: 10},
			{PID: 1, CPUPercent: 50, ResKB: 300, UTime: 5},
			{PID: 2, CPUPercent: 20, ResKB: 200, UTime: 20},
		}
	}
	p := fixture()
	sortProcs(p, SortCPU, true)
	if got := pids(p); !eq(got, []int{1, 2, 3}) {
		t.Errorf("SortCPU desc: %v", got)
	}
	p = fixture()
	sortProcs(p, SortMem, true)
	if got := pids(p); !eq(got, []int{1, 2, 3}) {
		t.Errorf("SortMem desc: %v", got)
	}
	p = fixture()
	sortProcs(p, SortTime, true)
	if got := pids(p); !eq(got, []int{2, 3, 1}) {
		t.Errorf("SortTime desc: %v", got)
	}
	p = fixture()
	sortProcs(p, SortPID, false)
	if got := pids(p); !eq(got, []int{1, 2, 3}) {
		t.Errorf("SortPID asc: %v", got)
	}
}

func TestSortKeys(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'N', tcell.ModNone))
	if app.sortBy != SortPID || app.sortDesc {
		t.Errorf("after N: sortBy=%v desc=%v, want SortPID asc", app.sortBy, app.sortDesc)
	}
	for i := 1; i < len(app.rows); i++ {
		if app.rows[i].Proc.PID < app.rows[i-1].Proc.PID {
			t.Fatalf("rows not PID-ascending at %d", i)
		}
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'I', tcell.ModNone))
	if !app.sortDesc {
		t.Error("I did not invert sort direction")
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'P', tcell.ModNone))
	if app.sortBy != SortCPU || !app.sortDesc {
		t.Errorf("after P: sortBy=%v desc=%v, want SortCPU desc", app.sortBy, app.sortDesc)
	}
}
