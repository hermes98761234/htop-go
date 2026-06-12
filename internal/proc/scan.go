package proc

import (
	"os"
	"strconv"
)

// CoreUsage is one core's busy fractions (0..1) since the previous scan.
type CoreUsage struct {
	User   float64 // user + nice ticks
	System float64 // system + irq + softirq + steal ticks
}

// Snapshot is everything the UI needs from one scan.
type Snapshot struct {
	Procs     []Process
	CPUs      []CoreUsage
	Mem       MemInfo
	Load      LoadAvg
	UptimeSec float64
	Tasks     int
	Threads   int
	Running   int
}

// Scanner scans /proc and computes CPU%/MEM% from deltas between scans.
type Scanner struct {
	NumCPU     int
	PageKB     uint64
	users      *UserTable
	prevTotal  uint64
	prevPerCPU []CPUTimes
	prevProc   map[int]uint64 // pid -> utime+stime at previous scan
}

// NewScanner creates a Scanner. The first Scan reports 0 CPU% everywhere.
func NewScanner() *Scanner {
	return &Scanner{
		PageKB:   uint64(os.Getpagesize() / 1024),
		users:    LoadUsers(),
		prevProc: map[int]uint64{},
	}
}

// readProcess reads one PID's files. Any error means the process vanished
// or is unreadable: callers must skip it.
func (s *Scanner) readProcess(pid int, memTotalKB uint64) (Process, error) {
	dir := "/proc/" + strconv.Itoa(pid)
	statData, err := os.ReadFile(dir + "/stat")
	if err != nil {
		return Process{}, err
	}
	f, err := ParsePIDStat(string(statData))
	if err != nil {
		return Process{}, err
	}
	p := Process{
		PID: f.PID, PPID: f.PPID, Comm: f.Comm, State: f.State,
		Priority: f.Priority, Nice: f.Nice, Threads: f.Threads,
		UTime: f.UTime, STime: f.STime, StartTime: f.StartTime,
		VirtKB: f.VSizeBytes / 1024,
	}
	if f.RSSPages > 0 {
		p.ResKB = uint64(f.RSSPages) * s.PageKB
	}
	if statmData, err := os.ReadFile(dir + "/statm"); err == nil {
		if shared, err := ParseStatmShared(string(statmData)); err == nil {
			p.ShrKB = shared * s.PageKB
		}
	}
	if statusData, err := os.ReadFile(dir + "/status"); err == nil {
		if uid, err := ParseStatusUID(string(statusData)); err == nil {
			p.UID = uid
		}
	}
	p.User = s.users.Name(p.UID)
	cmdRaw, _ := os.ReadFile(dir + "/cmdline")
	p.Cmdline = ParseCmdline(cmdRaw, p.Comm)
	if memTotalKB > 0 {
		p.MemPercent = float64(p.ResKB) / float64(memTotalKB) * 100
	}
	return p, nil
}

// Scan reads /proc and returns a fresh Snapshot.
func (s *Scanner) Scan() (*Snapshot, error) {
	total, perCPU, err := ReadStat()
	if err != nil {
		return nil, err
	}
	if len(perCPU) == 0 {
		perCPU = []CPUTimes{total}
	}
	s.NumCPU = len(perCPU)
	snap := &Snapshot{CPUs: make([]CoreUsage, len(perCPU))}
	if snap.Mem, err = ReadMemInfo(); err != nil {
		return nil, err
	}
	if snap.Load, err = ReadLoadAvg(); err != nil {
		return nil, err
	}
	if snap.UptimeSec, err = ReadUptime(); err != nil {
		return nil, err
	}

	// Per-core usage fractions since the previous scan.
	if len(s.prevPerCPU) == len(perCPU) {
		for i := range perCPU {
			dt := perCPU[i].Total() - s.prevPerCPU[i].Total()
			if dt == 0 {
				continue
			}
			du := (perCPU[i].User + perCPU[i].Nice) - (s.prevPerCPU[i].User + s.prevPerCPU[i].Nice)
			prevSys := s.prevPerCPU[i].System + s.prevPerCPU[i].IRQ + s.prevPerCPU[i].SoftIRQ + s.prevPerCPU[i].Steal
			curSys := perCPU[i].System + perCPU[i].IRQ + perCPU[i].SoftIRQ + perCPU[i].Steal
			snap.CPUs[i] = CoreUsage{
				User:   float64(du) / float64(dt),
				System: float64(curSys-prevSys) / float64(dt),
			}
		}
	}

	// Process CPU% scale: ticks elapsed on ONE core since previous scan.
	var ticksPerCPU float64
	if s.prevTotal > 0 && total.Total() > s.prevTotal {
		ticksPerCPU = float64(total.Total()-s.prevTotal) / float64(len(perCPU))
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	newPrevProc := make(map[int]uint64, len(entries))
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process directory
		}
		p, err := s.readProcess(pid, snap.Mem.MemTotal)
		if err != nil {
			continue // process vanished mid-scan: skip silently
		}
		ticks := p.UTime + p.STime
		newPrevProc[pid] = ticks
		if prev, ok := s.prevProc[pid]; ok && ticksPerCPU > 0 && ticks >= prev {
			p.CPUPercent = float64(ticks-prev) / ticksPerCPU * 100
		}
		snap.Procs = append(snap.Procs, p)
		snap.Threads += p.Threads
		if p.State == 'R' {
			snap.Running++
		}
	}
	snap.Tasks = len(snap.Procs)

	s.prevTotal = total.Total()
	s.prevPerCPU = perCPU
	s.prevProc = newPrevProc
	return snap, nil
}
