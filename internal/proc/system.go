// Package proc parses Linux /proc into plain Go values.
package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// CPUTimes holds one cpu line of /proc/stat, in USER_HZ ticks.
type CPUTimes struct {
	User, Nice, System, Idle, IOWait, IRQ, SoftIRQ, Steal uint64
}

// Total is the sum of all tick counters.
func (c CPUTimes) Total() uint64 {
	return c.User + c.Nice + c.System + c.Idle + c.IOWait + c.IRQ + c.SoftIRQ + c.Steal
}

// Busy is Total minus idle and iowait ticks.
func (c CPUTimes) Busy() uint64 {
	return c.Total() - c.Idle - c.IOWait
}

func parseCPULine(fields []string) (CPUTimes, error) {
	// fields[0] is the "cpu"/"cpuN" label; up to 8 counters follow
	// (guest/guest_nice are already included in user/nice, so ignore them).
	var vals [8]uint64
	for i := 0; i < 8; i++ {
		if i+1 >= len(fields) {
			break
		}
		v, err := strconv.ParseUint(fields[i+1], 10, 64)
		if err != nil {
			return CPUTimes{}, fmt.Errorf("bad cpu field %q: %w", fields[i+1], err)
		}
		vals[i] = v
	}
	return CPUTimes{
		User: vals[0], Nice: vals[1], System: vals[2], Idle: vals[3],
		IOWait: vals[4], IRQ: vals[5], SoftIRQ: vals[6], Steal: vals[7],
	}, nil
}

// ParseStat parses /proc/stat content. Returns the aggregate "cpu" line
// and one CPUTimes per "cpuN" line, in order.
func ParseStat(data string) (CPUTimes, []CPUTimes, error) {
	var total CPUTimes
	var perCPU []CPUTimes
	seenTotal := false
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "cpu") {
			continue
		}
		t, err := parseCPULine(fields)
		if err != nil {
			return CPUTimes{}, nil, err
		}
		if fields[0] == "cpu" {
			total = t
			seenTotal = true
		} else {
			perCPU = append(perCPU, t)
		}
	}
	if !seenTotal {
		return CPUTimes{}, nil, fmt.Errorf("no cpu line in /proc/stat data")
	}
	return total, perCPU, nil
}

// ReadStat reads and parses /proc/stat.
func ReadStat() (CPUTimes, []CPUTimes, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUTimes{}, nil, err
	}
	return ParseStat(string(data))
}

// MemInfo holds /proc/meminfo values, all in kB.
type MemInfo struct {
	MemTotal, MemFree, MemAvailable      uint64
	Buffers, Cached, SReclaimable, Shmem uint64
	SwapTotal, SwapFree                  uint64
}

// MemUsed follows htop: total - free - buffers - (cached + sreclaimable - shmem).
func (m MemInfo) MemUsed() uint64 {
	cached := m.Cached + m.SReclaimable - m.Shmem
	used := m.MemTotal - m.MemFree - m.Buffers - cached
	if used > m.MemTotal { // underflow guard
		return 0
	}
	return used
}

// SwapUsed is SwapTotal - SwapFree.
func (m MemInfo) SwapUsed() uint64 {
	if m.SwapFree > m.SwapTotal {
		return 0
	}
	return m.SwapTotal - m.SwapFree
}

// ParseMemInfo parses /proc/meminfo content.
func ParseMemInfo(data string) (MemInfo, error) {
	want := map[string]*uint64{}
	var m MemInfo
	want["MemTotal"] = &m.MemTotal
	want["MemFree"] = &m.MemFree
	want["MemAvailable"] = &m.MemAvailable
	want["Buffers"] = &m.Buffers
	want["Cached"] = &m.Cached
	want["SReclaimable"] = &m.SReclaimable
	want["Shmem"] = &m.Shmem
	want["SwapTotal"] = &m.SwapTotal
	want["SwapFree"] = &m.SwapFree
	for _, line := range strings.Split(data, "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		dst, wanted := want[name]
		if !wanted {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		v, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return MemInfo{}, fmt.Errorf("bad meminfo line %q: %w", line, err)
		}
		*dst = v
	}
	if m.MemTotal == 0 {
		return MemInfo{}, fmt.Errorf("no MemTotal in /proc/meminfo data")
	}
	return m, nil
}

// ReadMemInfo reads and parses /proc/meminfo.
func ReadMemInfo() (MemInfo, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return MemInfo{}, err
	}
	return ParseMemInfo(string(data))
}

// LoadAvg holds the three load averages from /proc/loadavg.
type LoadAvg struct {
	One, Five, Fifteen float64
}

// ParseLoadAvg parses /proc/loadavg content.
func ParseLoadAvg(data string) (LoadAvg, error) {
	fields := strings.Fields(data)
	if len(fields) < 3 {
		return LoadAvg{}, fmt.Errorf("short /proc/loadavg data: %q", data)
	}
	var l LoadAvg
	var err error
	if l.One, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return LoadAvg{}, err
	}
	if l.Five, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return LoadAvg{}, err
	}
	if l.Fifteen, err = strconv.ParseFloat(fields[2], 64); err != nil {
		return LoadAvg{}, err
	}
	return l, nil
}

// ReadLoadAvg reads and parses /proc/loadavg.
func ReadLoadAvg() (LoadAvg, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadAvg{}, err
	}
	return ParseLoadAvg(string(data))
}

// ParseUptime parses /proc/uptime content; returns uptime in seconds.
func ParseUptime(data string) (float64, error) {
	fields := strings.Fields(data)
	if len(fields) < 1 {
		return 0, fmt.Errorf("empty /proc/uptime data")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// ReadUptime reads and parses /proc/uptime.
func ReadUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	return ParseUptime(string(data))
}
