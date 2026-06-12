package proc

import "testing"

const statFixture = `cpu  263185 1925 65166 4889344 5786 0 4724 0 0 0
cpu0 131592 962 32583 2444672 2893 0 2362 0 0 0
cpu1 131593 963 32583 2444672 2893 0 2362 0 0 0
intr 8910374 9 0 0
ctxt 13123456
btime 1718000000
processes 12345
procs_running 2
procs_blocked 0
`

func TestParseStat(t *testing.T) {
	total, perCPU, err := ParseStat(statFixture)
	if err != nil {
		t.Fatalf("ParseStat error: %v", err)
	}
	if got := total.Total(); got != 5230130 {
		t.Errorf("total.Total() = %d, want 5230130", got)
	}
	if got := total.Busy(); got != 335000 {
		t.Errorf("total.Busy() = %d, want 335000", got)
	}
	if len(perCPU) != 2 {
		t.Fatalf("len(perCPU) = %d, want 2", len(perCPU))
	}
	if perCPU[0].User != 131592 || perCPU[1].User != 131593 {
		t.Errorf("perCPU User = %d,%d want 131592,131593", perCPU[0].User, perCPU[1].User)
	}
}

const meminfoFixture = `MemTotal:       16000000 kB
MemFree:         8000000 kB
MemAvailable:   11000000 kB
Buffers:          500000 kB
Cached:          3000000 kB
SwapCached:            0 kB
SReclaimable:     200000 kB
Shmem:            100000 kB
SwapTotal:       2000000 kB
SwapFree:        1500000 kB
`

func TestParseMemInfo(t *testing.T) {
	m, err := ParseMemInfo(meminfoFixture)
	if err != nil {
		t.Fatalf("ParseMemInfo error: %v", err)
	}
	if m.MemTotal != 16000000 {
		t.Errorf("MemTotal = %d, want 16000000", m.MemTotal)
	}
	if got := m.MemUsed(); got != 4400000 {
		t.Errorf("MemUsed() = %d, want 4400000", got)
	}
	if got := m.SwapUsed(); got != 500000 {
		t.Errorf("SwapUsed() = %d, want 500000", got)
	}
}

func TestParseLoadAvg(t *testing.T) {
	l, err := ParseLoadAvg("1.25 0.75 0.50 2/345 6789\n")
	if err != nil {
		t.Fatalf("ParseLoadAvg error: %v", err)
	}
	if l.One != 1.25 || l.Five != 0.75 || l.Fifteen != 0.50 {
		t.Errorf("got %+v, want 1.25 0.75 0.50", l)
	}
}

func TestParseUptime(t *testing.T) {
	up, err := ParseUptime("93784.50 180000.00\n")
	if err != nil {
		t.Fatalf("ParseUptime error: %v", err)
	}
	if up != 93784.50 {
		t.Errorf("uptime = %f, want 93784.50", up)
	}
}

func TestReadRealFiles(t *testing.T) {
	if _, _, err := ReadStat(); err != nil {
		t.Errorf("ReadStat: %v", err)
	}
	m, err := ReadMemInfo()
	if err != nil || m.MemTotal == 0 {
		t.Errorf("ReadMemInfo: %v, MemTotal=%d", err, m.MemTotal)
	}
	if _, err := ReadLoadAvg(); err != nil {
		t.Errorf("ReadLoadAvg: %v", err)
	}
	up, err := ReadUptime()
	if err != nil || up <= 0 {
		t.Errorf("ReadUptime: %v, up=%f", err, up)
	}
}
