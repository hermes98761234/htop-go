package proc

import (
	"os"
	"testing"
	"time"
)

func TestScannerScan(t *testing.T) {
	s := NewScanner()
	snap1, err := s.Scan()
	if err != nil {
		t.Fatalf("first Scan: %v", err)
	}
	if len(snap1.Procs) == 0 {
		t.Fatal("first scan found no processes")
	}
	time.Sleep(100 * time.Millisecond)
	snap, err := s.Scan()
	if err != nil {
		t.Fatalf("second Scan: %v", err)
	}
	if snap.Mem.MemTotal == 0 {
		t.Error("MemTotal is 0")
	}
	if snap.UptimeSec <= 0 {
		t.Errorf("UptimeSec = %f", snap.UptimeSec)
	}
	if len(snap.CPUs) == 0 {
		t.Error("no per-core usage")
	}
	if snap.Tasks != len(snap.Procs) {
		t.Errorf("Tasks = %d, len(Procs) = %d", snap.Tasks, len(snap.Procs))
	}
	if snap.Threads < snap.Tasks {
		t.Errorf("Threads = %d < Tasks = %d", snap.Threads, snap.Tasks)
	}
	self := os.Getpid()
	var found *Process
	for i := range snap.Procs {
		if snap.Procs[i].PID == self {
			found = &snap.Procs[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("own pid %d not in scan", self)
	}
	if found.User == "" || found.Cmdline == "" {
		t.Errorf("own process missing User (%q) or Cmdline (%q)", found.User, found.Cmdline)
	}
	if found.MemPercent <= 0 {
		t.Errorf("own MemPercent = %f, want > 0", found.MemPercent)
	}
	if found.CPUPercent < 0 {
		t.Errorf("own CPUPercent = %f, want >= 0", found.CPUPercent)
	}
}
