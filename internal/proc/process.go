package proc

import (
	"fmt"
	"strconv"
	"strings"
)

// Process is one row of the process table. CPUPercent/MemPercent are
// filled by Scanner (Task 4), not by the parsers here.
type Process struct {
	PID        int
	PPID       int
	Comm       string
	State      byte
	Priority   int64
	Nice       int64
	Threads    int
	UTime      uint64 // ticks
	STime      uint64 // ticks
	StartTime  uint64 // ticks since boot
	VirtKB     uint64
	ResKB      uint64
	ShrKB      uint64
	UID        int
	User       string
	Cmdline    string
	CPUPercent float64
	MemPercent float64
}

// StatFields are the raw fields parsed from /proc/[pid]/stat.
type StatFields struct {
	PID        int
	Comm       string
	State      byte
	PPID       int
	UTime      uint64
	STime      uint64
	Priority   int64
	Nice       int64
	Threads    int
	StartTime  uint64
	VSizeBytes uint64
	RSSPages   int64
}

// ParsePIDStat parses /proc/[pid]/stat content. The comm field is wrapped
// in parens and may contain spaces/parens, so split on the LAST ')'.
func ParsePIDStat(data string) (StatFields, error) {
	open := strings.IndexByte(data, '(')
	closing := strings.LastIndexByte(data, ')')
	if open < 0 || closing < 0 || closing < open {
		return StatFields{}, fmt.Errorf("malformed pid stat: %q", data)
	}
	var f StatFields
	pid, err := strconv.Atoi(strings.TrimSpace(data[:open]))
	if err != nil {
		return StatFields{}, fmt.Errorf("bad pid in stat: %w", err)
	}
	f.PID = pid
	f.Comm = data[open+1 : closing]
	rest := strings.Fields(data[closing+1:])
	// rest[i] is field (i+3) of proc(5): rest[0]=state(3), rest[1]=ppid(4),
	// rest[11]=utime(14), rest[12]=stime(15), rest[15]=priority(18),
	// rest[16]=nice(19), rest[17]=num_threads(20), rest[19]=starttime(22),
	// rest[20]=vsize(23), rest[21]=rss(24).
	if len(rest) < 22 {
		return StatFields{}, fmt.Errorf("short pid stat: %d fields after comm", len(rest))
	}
	f.State = rest[0][0]
	if f.PPID, err = strconv.Atoi(rest[1]); err != nil {
		return StatFields{}, err
	}
	if f.UTime, err = strconv.ParseUint(rest[11], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.STime, err = strconv.ParseUint(rest[12], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.Priority, err = strconv.ParseInt(rest[15], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.Nice, err = strconv.ParseInt(rest[16], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.Threads, err = strconv.Atoi(rest[17]); err != nil {
		return StatFields{}, err
	}
	if f.StartTime, err = strconv.ParseUint(rest[19], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.VSizeBytes, err = strconv.ParseUint(rest[20], 10, 64); err != nil {
		return StatFields{}, err
	}
	if f.RSSPages, err = strconv.ParseInt(rest[21], 10, 64); err != nil {
		return StatFields{}, err
	}
	return f, nil
}

// ParseStatmShared returns the "shared" field (in pages) of /proc/[pid]/statm.
func ParseStatmShared(data string) (uint64, error) {
	fields := strings.Fields(data)
	if len(fields) < 3 {
		return 0, fmt.Errorf("short statm data: %q", data)
	}
	return strconv.ParseUint(fields[2], 10, 64)
}

// ParseStatusUID returns the real UID from /proc/[pid]/status content.
func ParseStatusUID(data string) (int, error) {
	for _, line := range strings.Split(data, "\n") {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}
		fields := strings.Fields(line[4:])
		if len(fields) < 1 {
			break
		}
		return strconv.Atoi(fields[0])
	}
	return 0, fmt.Errorf("no Uid line in status data")
}

// ParseCmdline converts raw /proc/[pid]/cmdline bytes (NUL-separated) to a
// display string. Kernel threads have no cmdline: show "[comm]" like htop.
func ParseCmdline(raw []byte, comm string) string {
	s := strings.TrimRight(string(raw), "\x00")
	if s == "" {
		return "[" + comm + "]"
	}
	return strings.ReplaceAll(s, "\x00", " ")
}
