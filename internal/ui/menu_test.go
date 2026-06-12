package ui

import (
	"os/exec"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

func TestSignalMenuNavigation(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF9, 0, tcell.ModNone))
	if app.mode != ModeSignals {
		t.Fatal("F9 did not open the signal menu")
	}
	if signalEntries[app.sigSel].Name != "SIGTERM" {
		t.Errorf("default selection = %s, want SIGTERM", signalEntries[app.sigSel].Name)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	if signalEntries[app.sigSel].Name != "SIGKILL" {
		t.Errorf("after Up: %s, want SIGKILL", signalEntries[app.sigSel].Name)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if app.mode != ModeNormal {
		t.Error("Esc did not close the menu")
	}
}

func TestKillSendsSignal(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	app, _ := newTestApp(t)
	// Point the table at the child process directly.
	app.rows = []Row{{Proc: proc.Process{PID: cmd.Process.Pid, Nice: 0}}}
	app.table.Sel = 0
	app.mode = ModeSignals
	app.sigSel = 3 // SIGTERM
	app.handleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if app.status != "" {
		t.Fatalf("kill reported error: %s", app.status)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil || err.Error() != "signal: terminated" {
			t.Errorf("child exit: %v, want signal: terminated", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("child did not die after SIGTERM")
	}
}

func TestRenice(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	app, _ := newTestApp(t)
	app.rows = []Row{{Proc: proc.Process{PID: cmd.Process.Pid, Nice: 0}}}
	app.table.Sel = 0
	app.renice(1) // raising nice on own child never needs privileges
	if app.status != "" {
		t.Errorf("renice reported error: %s", app.status)
	}
}
