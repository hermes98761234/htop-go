# htop-go Implementation Plan

> **For agentic workers:** This plan is executed via Hermes kanban tasks (board `htop-go`), one task per section below. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `htop-go` — htop (the interactive Linux process viewer) rewritten in Go: live header meters (per-core CPU bars, Mem, Swp, load, uptime, task counts) plus a sortable, searchable, navigable process table with tree view, kill, and renice.

**Architecture:** Single static binary, Linux-only. `internal/proc` parses `/proc` with pure, fixture-testable functions and a `Scanner` that computes CPU%/MEM% from tick deltas between scans. `internal/ui` renders with tcell (cell-grid drawing, like ncurses) and owns the event loop: a 1.5 s ticker triggers rescans; keys drive sorting, tree mode, search/filter, signal menu, renice. The UI never reads `/proc` directly.

**Tech Stack:** Go 1.26 (preinstalled at `/usr/local/bin/go`), `github.com/gdamore/tcell/v2`. No other dependencies.

**Spec:** `docs/superpowers/specs/2026-06-12-htop-go-design.md` (in this repo).

**Rules for every task:**
- Work dir: `/home/user/projects/htop-go`. Never work elsewhere.
- Before claiming done: `gofmt -w .` then verify `gofmt -l .` prints nothing, then `go vet ./...` (zero findings), then `go test ./...` (all green), then `go build ./...`.
- Every task ends with `git add -A && git commit -m "<given message>" && git push origin main`.
- Match package, struct, function, and field names EXACTLY as written here — later tasks depend on them.
- Copy code blocks verbatim. Where a step says "replace function X", replace the whole function with the given code.
- CPU tick values are `uint64`; sizes in kB are `uint64`; PIDs are `int`.

---

### Task 1: Scaffold Go module + GitHub repo

**Files:**
- Create: `go.mod`, `main.go`, `.gitignore`

- [ ] **Step 1: Scaffold the module**

In `/home/user/projects/htop-go` (git repo already initialized, branch `main`, contains `docs/`):

```bash
cd /home/user/projects/htop-go
go mod init github.com/hermes98761234/htop-go
```

Create `main.go`:

```go
package main

import (
	"flag"
	"fmt"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	delayTenths := flag.Int("d", 15, "delay between updates, in tenths of seconds")
	flag.Parse()
	if *showVersion {
		fmt.Printf("htop-go %s\n", version)
		return
	}
	_ = delayTenths
	fmt.Println("htop-go: TUI not implemented yet")
}
```

Create `.gitignore`:

```
/htop-go
/dist
```

- [ ] **Step 2: Verify build**

```bash
go build ./... && go run . --version
```

Expected output: `htop-go 0.1.0`

- [ ] **Step 3: Format, vet, commit**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: scaffold htop-go module"
```

- [ ] **Step 4: Create the GitHub repo and push**

```bash
gh repo create htop-go --public --description '📊 htop rewritten in Go — interactive process viewer for Linux' --source . --remote origin
gh repo edit hermes98761234/htop-go --add-topic go --add-topic htop --add-topic tui --add-topic linux --add-topic tcell
git push -u origin main
gh repo view hermes98761234/htop-go --json url -q .url
```

Expected: prints `https://github.com/hermes98761234/htop-go`. Report this URL.

---

### Task 2: /proc system stats — CPU ticks, memory, load, uptime

**Files:**
- Create: `internal/proc/system.go`
- Test: `internal/proc/system_test.go`

All parsers are pure functions taking the file content as a string, so tests use fixture strings. Thin `Read*` wrappers read the real files.

- [ ] **Step 1: Write the failing tests**

Create `internal/proc/system_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/proc/
```

Expected: FAIL (compile error — `ParseStat` undefined).

- [ ] **Step 3: Implement system.go**

Create `internal/proc/system.go`:

```go
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
	MemTotal, MemFree, MemAvailable          uint64
	Buffers, Cached, SReclaimable, Shmem     uint64
	SwapTotal, SwapFree                      uint64
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/proc/ -v
```

Expected: all 5 tests PASS.

- [ ] **Step 5: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./... && go test ./... && go build ./...
git add -A && git commit -m "feat: parse /proc system stats (cpu, mem, load, uptime)" && git push origin main
```

---

### Task 3: /proc per-process parsing + user names

**Files:**
- Create: `internal/proc/process.go`, `internal/proc/users.go`
- Test: `internal/proc/process_test.go`

The tricky part: in `/proc/[pid]/stat` the command name is in parentheses and may itself contain spaces and parentheses (e.g. `(tmux: server)`). Parse by finding the LAST `)` in the line; fields after it are split on whitespace. With `rest := strings.Fields(data[afterParen:])`, field N of proc(5) is `rest[N-3]`.

- [ ] **Step 1: Write the failing tests**

Create `internal/proc/process_test.go`:

```go
package proc

import "testing"

const pidStatFixture = `1234 (tmux: server) S 1 1234 1234 0 -1 4194304 1000 0 0 0 250 150 0 0 20 0 3 0 100000 123456789 4321 18446744073709551615 1 1 0 0 0 0 0 3670020 1216 0 0 0 17 0 0 0 0 0 0 0 0 0 0 0 0 0`

func TestParsePIDStat(t *testing.T) {
	f, err := ParsePIDStat(pidStatFixture)
	if err != nil {
		t.Fatalf("ParsePIDStat error: %v", err)
	}
	if f.PID != 1234 {
		t.Errorf("PID = %d, want 1234", f.PID)
	}
	if f.Comm != "tmux: server" {
		t.Errorf("Comm = %q, want \"tmux: server\"", f.Comm)
	}
	if f.State != 'S' {
		t.Errorf("State = %c, want S", f.State)
	}
	if f.PPID != 1 {
		t.Errorf("PPID = %d, want 1", f.PPID)
	}
	if f.UTime != 250 || f.STime != 150 {
		t.Errorf("UTime,STime = %d,%d want 250,150", f.UTime, f.STime)
	}
	if f.Priority != 20 || f.Nice != 0 {
		t.Errorf("Priority,Nice = %d,%d want 20,0", f.Priority, f.Nice)
	}
	if f.Threads != 3 {
		t.Errorf("Threads = %d, want 3", f.Threads)
	}
	if f.StartTime != 100000 {
		t.Errorf("StartTime = %d, want 100000", f.StartTime)
	}
	if f.VSizeBytes != 123456789 {
		t.Errorf("VSizeBytes = %d, want 123456789", f.VSizeBytes)
	}
	if f.RSSPages != 4321 {
		t.Errorf("RSSPages = %d, want 4321", f.RSSPages)
	}
}

func TestParseStatmShared(t *testing.T) {
	shared, err := ParseStatmShared("12345 4321 999 100 0 5000 0\n")
	if err != nil {
		t.Fatalf("ParseStatmShared error: %v", err)
	}
	if shared != 999 {
		t.Errorf("shared = %d, want 999", shared)
	}
}

const statusFixture = `Name:	bash
Umask:	0022
State:	S (sleeping)
Tgid:	1234
Pid:	1234
PPid:	1
Uid:	1000	1000	1000	1000
Gid:	1000	1000	1000	1000
`

func TestParseStatusUID(t *testing.T) {
	uid, err := ParseStatusUID(statusFixture)
	if err != nil {
		t.Fatalf("ParseStatusUID error: %v", err)
	}
	if uid != 1000 {
		t.Errorf("uid = %d, want 1000", uid)
	}
}

func TestParseCmdline(t *testing.T) {
	got := ParseCmdline([]byte("/usr/bin/foo\x00--bar\x00baz\x00"), "foo")
	if got != "/usr/bin/foo --bar baz" {
		t.Errorf("got %q", got)
	}
	// Kernel threads have an empty cmdline: fall back to [comm].
	got = ParseCmdline(nil, "kthreadd")
	if got != "[kthreadd]" {
		t.Errorf("got %q, want [kthreadd]", got)
	}
}

const passwdFixture = `root:x:0:0:root:/root:/bin/bash
daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin
user:x:1000:1000::/home/user:/bin/bash
`

func TestUserTable(t *testing.T) {
	u := ParsePasswd(passwdFixture)
	if got := u.Name(0); got != "root" {
		t.Errorf("Name(0) = %q, want root", got)
	}
	if got := u.Name(1000); got != "user" {
		t.Errorf("Name(1000) = %q, want user", got)
	}
	// Unknown uid falls back to the number itself.
	if got := u.Name(4242); got != "4242" {
		t.Errorf("Name(4242) = %q, want 4242", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/proc/
```

Expected: FAIL (compile error — `ParsePIDStat` undefined).

- [ ] **Step 3: Implement process.go**

Create `internal/proc/process.go`:

```go
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
```

- [ ] **Step 4: Implement users.go**

Create `internal/proc/users.go`:

```go
package proc

import (
	"os"
	"strconv"
	"strings"
)

// UserTable maps numeric UIDs to user names.
type UserTable struct {
	names map[int]string
}

// ParsePasswd builds a UserTable from /etc/passwd content.
func ParsePasswd(data string) *UserTable {
	names := make(map[int]string)
	for _, line := range strings.Split(data, "\n") {
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		uid, err := strconv.Atoi(parts[2])
		if err != nil {
			continue
		}
		names[uid] = parts[0]
	}
	return &UserTable{names: names}
}

// LoadUsers reads /etc/passwd; on error returns an empty table (Name then
// falls back to numeric UIDs).
func LoadUsers() *UserTable {
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return &UserTable{names: map[int]string{}}
	}
	return ParsePasswd(string(data))
}

// Name returns the user name for uid, or the uid as a string if unknown.
func (u *UserTable) Name(uid int) string {
	if name, ok := u.names[uid]; ok {
		return name
	}
	return strconv.Itoa(uid)
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/proc/ -v
```

Expected: all tests PASS (the 5 from Task 2 plus the 5 new ones).

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./... && go test ./... && go build ./...
git add -A && git commit -m "feat: parse per-process /proc data and user names" && git push origin main
```

---

### Task 4: Scanner — full /proc scan with CPU%/MEM% deltas

**Files:**
- Create: `internal/proc/scan.go`
- Test: `internal/proc/scan_test.go`

The Scanner keeps the previous scan's tick counters and computes, for each process, `CPU% = Δ(utime+stime) / (Δtotal_ticks / ncpu) × 100` (htop convention: percent of ONE core; the sum over all processes can reach `ncpu × 100`). On the very first scan all CPU percentages are 0. Processes can vanish between listing and reading — ALL per-PID read errors mean "skip that PID", never fail the scan.

- [ ] **Step 1: Write the failing test**

Create `internal/proc/scan_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/proc/ -run TestScannerScan
```

Expected: FAIL (compile error — `NewScanner` undefined).

- [ ] **Step 3: Implement scan.go**

Create `internal/proc/scan.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/proc/ -v
```

Expected: all tests PASS, including `TestScannerScan`.

- [ ] **Step 5: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./... && go test ./... && go build ./...
git add -A && git commit -m "feat: add Scanner with CPU%/MEM% tick deltas" && git push origin main
```

---

### Task 5: tcell UI skeleton — event loop, ticker, quit, function-key bar

**Files:**
- Create: `internal/ui/app.go`, `internal/ui/style.go`, `internal/ui/draw.go`
- Modify: `main.go` (replace entirely)
- Test: `internal/ui/app_test.go`

After this task `./htop-go` starts a full-screen TUI showing a placeholder line and the bottom function-key bar, refreshes every 1.5 s, and quits on `q` or `F10`. Tests use tcell's `SimulationScreen` (no real terminal needed).

**IMPORTANT:** Define the App struct and Mode constants EXACTLY as below — every later task adds methods around them. Fields like `input`, `filter`, `sigSel`, `treeMode` are intentionally unused for now (unused struct fields are legal Go; unused IMPORTS are not).

- [ ] **Step 1: Add the tcell dependency**

```bash
go get github.com/gdamore/tcell/v2@latest
```

Expected: `go.mod` gains `github.com/gdamore/tcell/v2`.

- [ ] **Step 2: Create style.go**

Create `internal/ui/style.go`:

```go
// Package ui renders htop-go with tcell.
package ui

import "github.com/gdamore/tcell/v2"

var (
	styleDefault      = tcell.StyleDefault
	styleMeterLabel   = tcell.StyleDefault.Foreground(tcell.ColorTeal).Bold(true)
	styleBracket      = tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	styleBarUser      = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	styleBarSystem    = tcell.StyleDefault.Foreground(tcell.ColorRed)
	styleBarText      = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styleHeaderText   = tcell.StyleDefault.Foreground(tcell.ColorTeal)
	styleHeaderValue  = tcell.StyleDefault.Bold(true)
	styleTableHeader  = tcell.StyleDefault.Background(tcell.ColorGreen).Foreground(tcell.ColorBlack)
	styleSortedHeader = tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack)
	styleSelected     = tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack)
	styleRunning      = tcell.StyleDefault.Foreground(tcell.ColorGreen)
	styleZombie       = tcell.StyleDefault.Foreground(tcell.ColorGray)
	styleFnKey        = tcell.StyleDefault
	styleFnLabel      = tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack)
	styleStatus       = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	styleMenuBox      = tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)
	styleMenuSel      = tcell.StyleDefault.Background(tcell.ColorTeal).Foreground(tcell.ColorBlack)
)
```

- [ ] **Step 3: Create draw.go**

Create `internal/ui/draw.go`:

```go
package ui

import "github.com/gdamore/tcell/v2"

// drawString draws text at (x, y) and returns the x after the last rune.
func drawString(s tcell.Screen, x, y int, style tcell.Style, text string) int {
	for _, r := range text {
		s.SetContent(x, y, r, nil, style)
		x++
	}
	return x
}
```

- [ ] **Step 4: Create app.go**

Create `internal/ui/app.go`:

```go
package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

// Mode is the current input mode of the app.
type Mode int

const (
	ModeNormal Mode = iota
	ModeSearch
	ModeFilter
	ModeSignals
	ModeHelp
)

// Row is one display row: a process plus its tree prefix (empty when flat).
type Row struct {
	Proc   proc.Process
	Prefix string
}

// App owns the screen, the scanner, and all UI state.
type App struct {
	screen  tcell.Screen
	scanner *proc.Scanner
	delay   time.Duration
	snap    *proc.Snapshot
	rows    []Row
	mode    Mode
	input   string // text being typed in search/filter mode
	filter  string // committed filter (empty = none)
	sigSel  int    // selected entry in the F9 signal menu
	status  string // transient error/status line
	quit    bool
}

// NewApp wires a ready-to-run App. The screen must already be Init()ed.
func NewApp(screen tcell.Screen, scanner *proc.Scanner, delay time.Duration) *App {
	return &App{screen: screen, scanner: scanner, delay: delay}
}

// Run drives the event loop until quit. It calls screen.Fini on exit.
func (a *App) Run() error {
	defer a.screen.Fini()
	a.refresh()
	events := make(chan tcell.Event, 16)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				close(events)
				return
			}
			events <- ev
		}
	}()
	ticker := time.NewTicker(a.delay)
	defer ticker.Stop()
	a.draw()
	for !a.quit {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			a.handleEvent(ev)
		case <-ticker.C:
			a.refresh()
		}
		a.draw()
	}
	return nil
}

// refresh rescans /proc and rebuilds the display rows.
func (a *App) refresh() {
	snap, err := a.scanner.Scan()
	if err != nil {
		a.status = err.Error()
		return
	}
	a.snap = snap
	a.rebuild()
}

// rebuild recomputes a.rows from a.snap. (Sorting, filtering and tree mode
// are added in later tasks.)
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	rows := make([]Row, 0, len(a.snap.Procs))
	for _, p := range a.snap.Procs {
		rows = append(rows, Row{Proc: p})
	}
	a.rows = rows
}

func (a *App) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		a.screen.Sync()
	case *tcell.EventKey:
		a.handleKey(ev)
	}
}

func (a *App) handleKey(ev *tcell.EventKey) {
	a.status = ""
	switch a.mode {
	case ModeNormal:
		a.handleNormalKey(ev)
	}
}

func (a *App) handleNormalKey(ev *tcell.EventKey) {
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	}
}

// draw paints the whole screen.
func (a *App) draw() {
	a.screen.Clear()
	w, h := a.screen.Size()
	a.drawMain(w, h)
	a.drawBottom(w, h)
	a.screen.Show()
}

// drawMain paints everything above the bottom bar.
// (Replaced by the header in Task 6 and the process table in Task 7.)
func (a *App) drawMain(w, h int) {
	n := 0
	if a.snap != nil {
		n = a.snap.Tasks
	}
	drawString(a.screen, 0, 0, styleDefault, fmt.Sprintf("htop-go — %d processes", n))
}

// fnBarItems is the bottom function-key bar, htop style.
var fnBarItems = []struct{ Key, Label string }{
	{"F1", "Help"}, {"F3", "Search"}, {"F4", "Filter"}, {"F5", "Tree"},
	{"F7", "Nice-"}, {"F8", "Nice+"}, {"F9", "Kill"}, {"F10", "Quit"},
}

// drawBottom paints the bottom row: status line if set, else the fn-key bar.
// (Replaced in Task 10 to also render search/filter input.)
func (a *App) drawBottom(w, h int) {
	y := h - 1
	if a.status != "" {
		drawString(a.screen, 0, y, styleStatus, a.status)
		return
	}
	x := 0
	for _, item := range fnBarItems {
		x = drawString(a.screen, x, y, styleFnKey, item.Key)
		x = drawString(a.screen, x, y, styleFnLabel, item.Label+" ")
	}
}
```

- [ ] **Step 5: Replace main.go entirely**

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
	"github.com/hermes98761234/htop-go/internal/ui"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	delayTenths := flag.Int("d", 15, "delay between updates, in tenths of seconds")
	flag.Parse()
	if *showVersion {
		fmt.Printf("htop-go %s\n", version)
		return
	}
	if *delayTenths < 1 {
		*delayTenths = 1
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
	delay := time.Duration(*delayTenths) * 100 * time.Millisecond
	app := ui.NewApp(screen, proc.NewScanner(), delay)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Write the smoke test**

Create `internal/ui/app_test.go`:

```go
package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
)

// newTestApp builds an App on a 120x40 simulation screen with a real
// scan already loaded (tests run on Linux, /proc is available).
func newTestApp(t *testing.T) (*App, tcell.SimulationScreen) {
	t.Helper()
	sim := tcell.NewSimulationScreen("UTF-8")
	if err := sim.Init(); err != nil {
		t.Fatal(err)
	}
	sim.SetSize(120, 40)
	app := NewApp(sim, proc.NewScanner(), 1500*time.Millisecond)
	app.refresh()
	return app, sim
}

// screenText flattens the simulation screen into one string for assertions.
func screenText(sim tcell.SimulationScreen) string {
	cells, w, h := sim.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) > 0 {
				b.WriteRune(c.Runes[0])
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func TestDrawSmoke(t *testing.T) {
	app, sim := newTestApp(t)
	app.draw()
	text := screenText(sim)
	if !strings.Contains(text, "Quit") {
		t.Errorf("function bar not drawn; screen:\n%s", text)
	}
	if len(app.rows) == 0 {
		t.Error("no rows after refresh")
	}
}

func TestQuitKeys(t *testing.T) {
	app, _ := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))
	if !app.quit {
		t.Error("q did not quit")
	}
	app.quit = false
	app.handleKey(tcell.NewEventKey(tcell.KeyF10, 0, tcell.ModNone))
	if !app.quit {
		t.Error("F10 did not quit")
	}
}
```

- [ ] **Step 7: Run tests**

```bash
go test ./... && go build ./...
```

Expected: all packages PASS, build succeeds.

- [ ] **Step 8: Manual smoke check (no real TTY available)**

```bash
go vet ./...
./htop-go --version 2>/dev/null || go run . --version
```

Expected: `htop-go 0.1.0`. (Running the TUI itself needs a terminal; the SimulationScreen test covers drawing.)

- [ ] **Step 9: Format, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
git add -A && git commit -m "feat: tcell app skeleton with event loop and fn-key bar" && git push origin main
```

---

### Task 6: Header meters — CPU bars, Mem, Swp, Tasks/Load/Uptime

**Files:**
- Create: `internal/ui/header.go`, `internal/ui/format.go`
- Modify: `internal/ui/draw.go` (add `BarSeg` + `drawBar`), `internal/ui/app.go` (replace `drawMain` only)
- Test: `internal/ui/format_test.go`, `internal/ui/header_test.go`

Layout (htop default): two columns. Left column: first half of the CPU bars, then `Mem` and `Swp` bars. Right column: remaining CPU bars, then `Tasks`, `Load average`, `Uptime` text lines. Header height is `ceil(ncpu/2) + 3` rows. CPU labels are 1-based like htop.

- [ ] **Step 1: Write the failing format tests**

Create `internal/ui/format_test.go`:

```go
package ui

import "testing"

func TestFormatMeter(t *testing.T) {
	cases := []struct {
		kb   uint64
		want string
	}{
		{800, "800K"},
		{512000, "500M"},
		{3355443, "3.20G"},
		{15728640, "15.0G"},
	}
	for _, c := range cases {
		if got := FormatMeter(c.kb); got != c.want {
			t.Errorf("FormatMeter(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		sec  float64
		want string
	}{
		{3784, "01:03:04"},
		{93784, "1 day, 02:03:04"},
		{200000, "2 days, 07:33:20"},
	}
	for _, c := range cases {
		if got := FormatUptime(c.sec); got != c.want {
			t.Errorf("FormatUptime(%f) = %q, want %q", c.sec, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/
```

Expected: FAIL (compile error — `FormatMeter` undefined).

- [ ] **Step 3: Create format.go**

Create `internal/ui/format.go`:

```go
package ui

import "fmt"

// FormatMeter renders a kB amount for meter texts: "800K", "500M", "3.20G".
func FormatMeter(kb uint64) string {
	g := float64(kb) / (1024 * 1024)
	switch {
	case g >= 10:
		return fmt.Sprintf("%.1fG", g)
	case g >= 1:
		return fmt.Sprintf("%.2fG", g)
	case kb >= 1024:
		return fmt.Sprintf("%dM", kb/1024)
	default:
		return fmt.Sprintf("%dK", kb)
	}
}

// FormatUptime renders seconds as "hh:mm:ss" with an optional day prefix.
func FormatUptime(sec float64) string {
	t := int64(sec)
	days := t / 86400
	h := t % 86400 / 3600
	m := t % 3600 / 60
	s := t % 60
	switch {
	case days == 1:
		return fmt.Sprintf("1 day, %02d:%02d:%02d", h, m, s)
	case days > 1:
		return fmt.Sprintf("%d days, %02d:%02d:%02d", days, h, m, s)
	default:
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
}
```

Run `go test ./internal/ui/` — the two format tests must now PASS.

- [ ] **Step 4: Add drawBar to draw.go**

Append to `internal/ui/draw.go`:

```go
// BarSeg is one colored segment of a meter bar.
type BarSeg struct {
	Frac  float64
	Style tcell.Style
}

// drawBar renders an htop-style meter: `LBL[|||||||       text]`.
// width is the TOTAL width including label and brackets. Segment fills are
// drawn left to right as '|' runes; text is right-aligned inside the bar.
func drawBar(s tcell.Screen, x, y, width int, label string, segs []BarSeg, text string) {
	x = drawString(s, x, y, styleMeterLabel, label)
	x = drawString(s, x, y, styleBracket, "[")
	inner := width - len(label) - 2
	if inner < 1 {
		return
	}
	runes := make([]rune, inner)
	styles := make([]tcell.Style, inner)
	for i := range runes {
		runes[i] = ' '
		styles[i] = styleBarText
	}
	pos := 0
	for _, seg := range segs {
		n := int(seg.Frac*float64(inner) + 0.5)
		for i := 0; i < n && pos < inner; i++ {
			runes[pos] = '|'
			styles[pos] = seg.Style
			pos++
		}
	}
	if len(text) > inner {
		text = text[:inner]
	}
	off := inner - len(text)
	for i, r := range text {
		runes[off+i] = r
	}
	for i := 0; i < inner; i++ {
		s.SetContent(x+i, y, runes[i], nil, styles[i])
	}
	s.SetContent(x+inner, y, ']', nil, styleBracket)
}
```

- [ ] **Step 5: Create header.go**

Create `internal/ui/header.go`:

```go
package ui

import "fmt"

// drawHeader paints the meter area at the top; returns rows used.
func (a *App) drawHeader(w int) int {
	if a.snap == nil {
		return 0
	}
	snap := a.snap
	ncpu := len(snap.CPUs)
	rowsPerCol := (ncpu + 1) / 2
	colW := w/2 - 2
	rightX := w / 2

	// CPU bars: left column first half, right column the rest. 1-based labels.
	for i, u := range snap.CPUs {
		x, y := 0, i
		if i >= rowsPerCol {
			x, y = rightX, i-rowsPerCol
		}
		label := fmt.Sprintf("%3d", i+1)
		text := fmt.Sprintf("%.1f%%", (u.User+u.System)*100)
		drawBar(a.screen, x, y, colW, label, []BarSeg{
			{Frac: u.User, Style: styleBarUser},
			{Frac: u.System, Style: styleBarSystem},
		}, text)
	}

	// Left column below CPUs: Mem and Swp bars.
	memFrac := 0.0
	if snap.Mem.MemTotal > 0 {
		memFrac = float64(snap.Mem.MemUsed()) / float64(snap.Mem.MemTotal)
	}
	memText := FormatMeter(snap.Mem.MemUsed()) + "/" + FormatMeter(snap.Mem.MemTotal)
	drawBar(a.screen, 0, rowsPerCol, colW, "Mem", []BarSeg{{Frac: memFrac, Style: styleBarUser}}, memText)

	swpFrac := 0.0
	if snap.Mem.SwapTotal > 0 {
		swpFrac = float64(snap.Mem.SwapUsed()) / float64(snap.Mem.SwapTotal)
	}
	swpText := FormatMeter(snap.Mem.SwapUsed()) + "/" + FormatMeter(snap.Mem.SwapTotal)
	drawBar(a.screen, 0, rowsPerCol+1, colW, "Swp", []BarSeg{{Frac: swpFrac, Style: styleBarUser}}, swpText)

	// Right column below CPUs: Tasks, Load average, Uptime.
	x := drawString(a.screen, rightX, rowsPerCol, styleHeaderText, "Tasks: ")
	drawString(a.screen, x, rowsPerCol, styleHeaderValue,
		fmt.Sprintf("%d, %d thr; %d running", snap.Tasks, snap.Threads, snap.Running))
	x = drawString(a.screen, rightX, rowsPerCol+1, styleHeaderText, "Load average: ")
	drawString(a.screen, x, rowsPerCol+1, styleHeaderValue,
		fmt.Sprintf("%.2f %.2f %.2f", snap.Load.One, snap.Load.Five, snap.Load.Fifteen))
	x = drawString(a.screen, rightX, rowsPerCol+2, styleHeaderText, "Uptime: ")
	drawString(a.screen, x, rowsPerCol+2, styleHeaderValue, FormatUptime(snap.UptimeSec))

	return rowsPerCol + 3
}
```

- [ ] **Step 6: Replace drawMain in app.go**

Replace the whole `drawMain` function (and ONLY it) in `internal/ui/app.go` with:

```go
// drawMain paints everything above the bottom bar.
// (The process table is added here in Task 7.)
func (a *App) drawMain(w, h int) {
	a.drawHeader(w)
}
```

The `"fmt"` import in app.go becomes unused — remove it from the import block.

- [ ] **Step 7: Write the header smoke test**

Create `internal/ui/header_test.go`:

```go
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
```

- [ ] **Step 8: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 9: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: header meters (CPU bars, Mem, Swp, tasks/load/uptime)" && git push origin main
```

---

### Task 7: Process table — columns, colors, selection, scrolling

**Files:**
- Create: `internal/ui/table.go`
- Modify: `internal/ui/format.go` (append two functions), `internal/ui/app.go` (add `table` field; replace `rebuild`, `drawMain`, `handleNormalKey`)
- Test: `internal/ui/table_test.go`, `internal/ui/format_test.go` (append)

Columns (separated by one space; Command stretches to the right edge):

```
PID(7,right) USER(9,left) PRI(3,right) NI(3,right) VIRT(6,right) RES(6,right) SHR(6,right) S(1) CPU%(5,right) MEM%(5,right) TIME+(9,right) Command(stretch)
```

- [ ] **Step 1: Append failing format tests**

Append to `internal/ui/format_test.go`:

```go
func TestFormatSize(t *testing.T) {
	cases := []struct {
		kb   uint64
		want string
	}{
		{0, "0"},
		{99999, "99999"},
		{100000, "97M"},
		{2097152, "2048M"},
		{20971520, "20.0G"},
	}
	for _, c := range cases {
		if got := FormatSize(c.kb); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}

func TestFormatTimePlus(t *testing.T) {
	if got := FormatTimePlus(12345, 100); got != "2:03.45" {
		t.Errorf("got %q, want 2:03.45", got)
	}
	if got := FormatTimePlus(372300, 100); got != "1:02:03" {
		t.Errorf("got %q, want 1:02:03", got)
	}
}
```

Run `go test ./internal/ui/` — expected: FAIL (`FormatSize` undefined).

- [ ] **Step 2: Append to format.go**

Append to `internal/ui/format.go` (and add `"strconv"` to its imports):

```go
// FormatSize renders a kB amount for table columns: plain kB below 100000,
// then integer MB below 10 GB, then one-decimal GB.
func FormatSize(kb uint64) string {
	switch {
	case kb < 100000:
		return strconv.FormatUint(kb, 10)
	case kb < 10*1024*1024:
		return fmt.Sprintf("%dM", kb/1024)
	default:
		return fmt.Sprintf("%.1fG", float64(kb)/(1024*1024))
	}
}

// FormatTimePlus renders CPU ticks as htop's TIME+ column:
// "m:ss.cc" below one hour, else "h:mm:ss". USER_HZ is 100 on Linux.
func FormatTimePlus(ticks uint64, hz uint64) string {
	cs := ticks * 100 / hz
	h := cs / 360000
	m := cs % 360000 / 6000
	s := cs % 6000 / 100
	c := cs % 100
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d.%02d", m, s, c)
}
```

Run `go test ./internal/ui/` — the format tests must now PASS.

- [ ] **Step 3: Create table.go**

Create `internal/ui/table.go`:

```go
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
)

// Table tracks selection and scroll state for the process list.
type Table struct {
	Sel    int // selected index into the rows slice
	Scroll int // first visible row index
}

// ClampTo keeps Sel and Scroll inside [0, n).
func (t *Table) ClampTo(n int) {
	if t.Sel >= n {
		t.Sel = n - 1
	}
	if t.Sel < 0 {
		t.Sel = 0
	}
	if t.Scroll > t.Sel {
		t.Scroll = t.Sel
	}
	if t.Scroll < 0 {
		t.Scroll = 0
	}
}

// HandleKey processes navigation keys; returns true if the key was consumed.
func (t *Table) HandleKey(ev *tcell.EventKey, n, pageRows int) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		t.Sel--
	case tcell.KeyDown:
		t.Sel++
	case tcell.KeyPgUp:
		t.Sel -= pageRows
	case tcell.KeyPgDn:
		t.Sel += pageRows
	case tcell.KeyHome:
		t.Sel = 0
	case tcell.KeyEnd:
		t.Sel = n - 1
	default:
		return false
	}
	if t.Sel >= n {
		t.Sel = n - 1
	}
	if t.Sel < 0 {
		t.Sel = 0
	}
	return true
}

type column struct {
	Title string
	Width int // 0 = stretch to the remaining width (Command)
	Right bool
}

var columns = []column{
	{"PID", 7, true}, {"USER", 9, false}, {"PRI", 3, true}, {"NI", 3, true},
	{"VIRT", 6, true}, {"RES", 6, true}, {"SHR", 6, true}, {"S", 1, false},
	{"CPU%", 5, true}, {"MEM%", 5, true}, {"TIME+", 9, true}, {"Command", 0, false},
}

// pad truncates or pads s to exactly w runes.
func pad(s string, w int, right bool) string {
	if len(s) > w {
		return s[:w]
	}
	if right {
		return strings.Repeat(" ", w-len(s)) + s
	}
	return s + strings.Repeat(" ", w-len(s))
}

// formatPri shows "RT" for real-time priorities like htop.
func formatPri(pri int64) string {
	if pri < -99 {
		return "RT"
	}
	return strconv.FormatInt(pri, 10)
}

// formatCell renders column i of row r.
func formatCell(r Row, i int) string {
	p := r.Proc
	switch i {
	case 0:
		return strconv.Itoa(p.PID)
	case 1:
		return p.User
	case 2:
		return formatPri(p.Priority)
	case 3:
		return strconv.FormatInt(p.Nice, 10)
	case 4:
		return FormatSize(p.VirtKB)
	case 5:
		return FormatSize(p.ResKB)
	case 6:
		return FormatSize(p.ShrKB)
	case 7:
		return string(p.State)
	case 8:
		return fmt.Sprintf("%.1f", p.CPUPercent)
	case 9:
		return fmt.Sprintf("%.1f", p.MemPercent)
	case 10:
		return FormatTimePlus(p.UTime+p.STime, 100)
	default:
		return r.Prefix + p.Cmdline
	}
}

// Draw renders the column header at row y0 and processes below it, using
// height rows total. sortCol is the column index to highlight.
func (t *Table) Draw(s tcell.Screen, w, y0, height int, rows []Row, sortCol int) {
	visible := height - 1
	if visible < 1 {
		return
	}
	t.ClampTo(len(rows))
	if t.Sel >= t.Scroll+visible {
		t.Scroll = t.Sel - visible + 1
	}
	if t.Sel < t.Scroll {
		t.Scroll = t.Sel
	}

	// Column header row: green background across the full width.
	for x := 0; x < w; x++ {
		s.SetContent(x, y0, ' ', nil, styleTableHeader)
	}
	x := 0
	for i, c := range columns {
		style := styleTableHeader
		if i == sortCol {
			style = styleSortedHeader
		}
		cw := c.Width
		if cw == 0 {
			cw = w - x
		}
		if cw < 1 {
			break
		}
		x = drawString(s, x, y0, style, pad(c.Title, cw, c.Right))
		x = drawString(s, x, y0, styleTableHeader, " ")
	}

	// Process rows.
	for line := 0; line < visible; line++ {
		idx := t.Scroll + line
		if idx >= len(rows) {
			break
		}
		y := y0 + 1 + line
		r := rows[idx]
		base := styleDefault
		if r.Proc.State == 'Z' {
			base = styleZombie
		}
		selected := idx == t.Sel
		if selected {
			base = styleSelected
			for fx := 0; fx < w; fx++ {
				s.SetContent(fx, y, ' ', nil, base)
			}
		}
		x := 0
		for i, c := range columns {
			cw := c.Width
			if cw == 0 {
				cw = w - x
			}
			if cw < 1 {
				break
			}
			style := base
			if i == 7 && r.Proc.State == 'R' && !selected {
				style = styleRunning
			}
			x = drawString(s, x, y, style, pad(formatCell(r, i), cw, c.Right))
			x = drawString(s, x, y, base, " ")
		}
	}
}
```

- [ ] **Step 4: Wire the table into app.go**

In `internal/ui/app.go`:

4a. Add the field `table Table` to the `App` struct, directly after the line `rows    []Row`:

```go
	rows    []Row
	table   Table
```

4b. Replace the whole `rebuild` function with (keeps the selection on the same PID across refreshes):

```go
// rebuild recomputes a.rows from a.snap, keeping the selection on the same
// PID when possible. (Sorting/filter/tree are added in Tasks 8-10.)
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	selPID := 0
	if len(a.rows) > 0 && a.table.Sel < len(a.rows) {
		selPID = a.rows[a.table.Sel].Proc.PID
	}
	rows := make([]Row, 0, len(a.snap.Procs))
	for _, p := range a.snap.Procs {
		rows = append(rows, Row{Proc: p})
	}
	a.rows = rows
	a.table.ClampTo(len(a.rows))
	if selPID != 0 {
		for i := range a.rows {
			if a.rows[i].Proc.PID == selPID {
				a.table.Sel = i
				break
			}
		}
	}
}
```

4c. Replace the whole `drawMain` function with:

```go
// drawMain paints the header and the process table.
func (a *App) drawMain(w, h int) {
	headerH := a.drawHeader(w)
	tableH := h - headerH - 1 // one row reserved for the bottom bar
	if tableH < 2 {
		return
	}
	a.table.Draw(a.screen, w, headerH, tableH, a.rows, 8) // 8 = CPU% (sort keys come in Task 8)
}
```

4d. Replace the whole `handleNormalKey` function with:

```go
func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	}
}
```

- [ ] **Step 5: Write the table tests**

Create `internal/ui/table_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPad(t *testing.T) {
	if got := pad("ab", 5, true); got != "   ab" {
		t.Errorf("right pad: %q", got)
	}
	if got := pad("ab", 5, false); got != "ab   " {
		t.Errorf("left pad: %q", got)
	}
	if got := pad("abcdef", 3, false); got != "abc" {
		t.Errorf("truncate: %q", got)
	}
}

func TestTableDrawAndNavigate(t *testing.T) {
	app, sim := newTestApp(t)
	app.draw()
	text := screenText(sim)
	for _, want := range []string{"PID", "USER", "CPU%", "Command"} {
		if !strings.Contains(text, want) {
			t.Errorf("table header missing %q", want)
		}
	}
	sel := app.table.Sel
	app.handleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	if app.table.Sel != sel+1 {
		t.Errorf("KeyDown: Sel = %d, want %d", app.table.Sel, sel+1)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone))
	if app.table.Sel != 0 {
		t.Errorf("KeyHome: Sel = %d, want 0", app.table.Sel)
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone))
	if app.table.Sel != len(app.rows)-1 {
		t.Errorf("KeyEnd: Sel = %d, want %d", app.table.Sel, len(app.rows)-1)
	}
}
```

- [ ] **Step 6: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 7: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: process table with columns, selection, scrolling" && git push origin main
```

---

### Task 8: Sorting — P/M/T/N keys, invert with I

**Files:**
- Create: `internal/ui/sort.go`
- Modify: `internal/ui/app.go` (add 2 fields; edit `NewApp`; replace `rebuild`, `drawMain`, `handleNormalKey`; add `setSort`)
- Test: `internal/ui/sort_test.go`

Default sort: CPU% descending. `P`=CPU%, `M`=MEM% (by resident size), `T`=TIME+, `N`=PID (ascending by default), `I` inverts the current direction. The sorted column header is highlighted cyan.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/sort_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/
```

Expected: FAIL (compile error — `sortProcs`, `SortCPU` undefined).

- [ ] **Step 3: Create sort.go**

Create `internal/ui/sort.go`:

```go
package ui

import (
	"sort"

	"github.com/hermes98761234/htop-go/internal/proc"
)

// SortBy selects the process table sort key.
type SortBy int

const (
	SortCPU SortBy = iota
	SortMem
	SortTime
	SortPID
)

// columnIndex maps a SortBy to its table column for header highlighting.
func (s SortBy) columnIndex() int {
	switch s {
	case SortCPU:
		return 8
	case SortMem:
		return 9
	case SortTime:
		return 10
	default:
		return 0
	}
}

// defaultDesc is the natural direction when a sort key is first chosen.
func (s SortBy) defaultDesc() bool {
	return s != SortPID
}

// sortProcs orders procs by the given key; desc flips the direction.
// Ties always break by ascending PID.
func sortProcs(procs []proc.Process, by SortBy, desc bool) {
	less := func(a, b proc.Process) bool {
		switch by {
		case SortCPU:
			if a.CPUPercent != b.CPUPercent {
				return a.CPUPercent < b.CPUPercent
			}
		case SortMem:
			if a.ResKB != b.ResKB {
				return a.ResKB < b.ResKB
			}
		case SortTime:
			at, bt := a.UTime+a.STime, b.UTime+b.STime
			if at != bt {
				return at < bt
			}
		}
		return a.PID < b.PID
	}
	sort.SliceStable(procs, func(i, j int) bool {
		if desc {
			return less(procs[j], procs[i])
		}
		return less(procs[i], procs[j])
	})
}
```

- [ ] **Step 4: Wire sorting into app.go**

4a. Add two fields to the `App` struct, directly after the line `table   Table`:

```go
	table    Table
	sortBy   SortBy
	sortDesc bool
```

4b. Replace the whole `NewApp` function with:

```go
// NewApp wires a ready-to-run App. The screen must already be Init()ed.
func NewApp(screen tcell.Screen, scanner *proc.Scanner, delay time.Duration) *App {
	return &App{
		screen:   screen,
		scanner:  scanner,
		delay:    delay,
		sortBy:   SortCPU,
		sortDesc: true,
	}
}
```

4c. Add this function after `NewApp`:

```go
// setSort switches the sort key and resets to its natural direction.
func (a *App) setSort(by SortBy) {
	a.sortBy = by
	a.sortDesc = by.defaultDesc()
	a.rebuild()
}
```

4d. Replace the whole `rebuild` function with:

```go
// rebuild recomputes a.rows from a.snap: sort, then keep the selection on
// the same PID when possible. (Filter/tree are added in Tasks 9-10.)
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	selPID := 0
	if len(a.rows) > 0 && a.table.Sel < len(a.rows) {
		selPID = a.rows[a.table.Sel].Proc.PID
	}
	procs := make([]proc.Process, len(a.snap.Procs))
	copy(procs, a.snap.Procs)
	sortProcs(procs, a.sortBy, a.sortDesc)
	rows := make([]Row, 0, len(procs))
	for _, p := range procs {
		rows = append(rows, Row{Proc: p})
	}
	a.rows = rows
	a.table.ClampTo(len(a.rows))
	if selPID != 0 {
		for i := range a.rows {
			if a.rows[i].Proc.PID == selPID {
				a.table.Sel = i
				break
			}
		}
	}
}
```

4e. Replace the whole `drawMain` function with:

```go
// drawMain paints the header and the process table.
func (a *App) drawMain(w, h int) {
	headerH := a.drawHeader(w)
	tableH := h - headerH - 1 // one row reserved for the bottom bar
	if tableH < 2 {
		return
	}
	a.table.Draw(a.screen, w, headerH, tableH, a.rows, a.sortBy.columnIndex())
}
```

4f. Replace the whole `handleNormalKey` function with:

```go
func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	case ev.Rune() == 'P':
		a.setSort(SortCPU)
	case ev.Rune() == 'M':
		a.setSort(SortMem)
	case ev.Rune() == 'T':
		a.setSort(SortTime)
	case ev.Rune() == 'N':
		a.setSort(SortPID)
	case ev.Rune() == 'I':
		a.sortDesc = !a.sortDesc
		a.rebuild()
	}
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: sort keys P/M/T/N with I to invert" && git push origin main
```

---

### Task 9: Tree view (F5)

**Files:**
- Create: `internal/ui/tree.go`
- Modify: `internal/ui/app.go` (add `treeMode` field; replace `rebuild`, `handleNormalKey`)
- Test: `internal/ui/tree_test.go`

Tree mode orders processes parent-first with branch prefixes on the Command column (`├─ `, `└─ `, `│  `). Siblings are ordered by the active sort key. A process whose PPID is 0 or missing from the snapshot is a root.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/tree_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/
```

Expected: FAIL (compile error — `buildTreeRows` undefined).

- [ ] **Step 3: Create tree.go**

Create `internal/ui/tree.go`:

```go
package ui

import "github.com/hermes98761234/htop-go/internal/proc"

// buildTreeRows orders processes parent-first with htop-style branch
// prefixes. Siblings are sorted with the active sort key.
func buildTreeRows(procs []proc.Process, by SortBy, desc bool) []Row {
	present := make(map[int]bool, len(procs))
	for _, p := range procs {
		present[p.PID] = true
	}
	children := make(map[int][]proc.Process)
	var roots []proc.Process
	for _, p := range procs {
		if p.PPID != 0 && p.PPID != p.PID && present[p.PPID] {
			children[p.PPID] = append(children[p.PPID], p)
		} else {
			roots = append(roots, p)
		}
	}
	sortProcs(roots, by, desc)
	for pid := range children {
		sortProcs(children[pid], by, desc)
	}
	rows := make([]Row, 0, len(procs))
	var walk func(p proc.Process, prefix, childPrefix string)
	walk = func(p proc.Process, prefix, childPrefix string) {
		rows = append(rows, Row{Proc: p, Prefix: prefix})
		kids := children[p.PID]
		for i, k := range kids {
			if i == len(kids)-1 {
				walk(k, childPrefix+"└─ ", childPrefix+"   ")
			} else {
				walk(k, childPrefix+"├─ ", childPrefix+"│  ")
			}
		}
	}
	for _, r := range roots {
		walk(r, "", "")
	}
	return rows
}
```

- [ ] **Step 4: Wire tree mode into app.go**

4a. Add the field `treeMode bool` to the `App` struct, directly after the line `sortDesc bool`:

```go
	sortDesc bool
	treeMode bool
```

4b. Replace the whole `rebuild` function with:

```go
// rebuild recomputes a.rows from a.snap: sort (or tree), then keep the
// selection on the same PID when possible. (Filter is added in Task 10.)
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	selPID := 0
	if len(a.rows) > 0 && a.table.Sel < len(a.rows) {
		selPID = a.rows[a.table.Sel].Proc.PID
	}
	procs := make([]proc.Process, len(a.snap.Procs))
	copy(procs, a.snap.Procs)
	if a.treeMode {
		a.rows = buildTreeRows(procs, a.sortBy, a.sortDesc)
	} else {
		sortProcs(procs, a.sortBy, a.sortDesc)
		rows := make([]Row, 0, len(procs))
		for _, p := range procs {
			rows = append(rows, Row{Proc: p})
		}
		a.rows = rows
	}
	a.table.ClampTo(len(a.rows))
	if selPID != 0 {
		for i := range a.rows {
			if a.rows[i].Proc.PID == selPID {
				a.table.Sel = i
				break
			}
		}
	}
}
```

4c. Replace the whole `handleNormalKey` function with:

```go
func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	case ev.Key() == tcell.KeyF5:
		a.treeMode = !a.treeMode
		a.rebuild()
	case ev.Rune() == 'P':
		a.setSort(SortCPU)
	case ev.Rune() == 'M':
		a.setSort(SortMem)
	case ev.Rune() == 'T':
		a.setSort(SortTime)
	case ev.Rune() == 'N':
		a.setSort(SortPID)
	case ev.Rune() == 'I':
		a.sortDesc = !a.sortDesc
		a.rebuild()
	}
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: tree view toggle on F5" && git push origin main
```

---

### Task 10: Incremental search (F3) and filter (F4)

**Files:**
- Create: `internal/ui/search.go`
- Modify: `internal/ui/app.go` (replace `rebuild`, `handleKey`, `handleNormalKey`, `drawBottom`)
- Test: `internal/ui/search_test.go`

Search (`F3`): typing jumps the selection to the next matching command line (case-insensitive, wraps around); `Enter` keeps the position, `Esc` too — both just leave search mode. Filter (`F4`): typing live-narrows the table; `Enter` commits the filter (it stays active, shown until cleared), `Esc` clears it. While a filter is active, tree mode is bypassed (flat list), like htop.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/search_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/
```

Expected: FAIL (compile error — `findNext` undefined).

- [ ] **Step 3: Create search.go**

Create `internal/ui/search.go`:

```go
package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
)

// matchRow reports whether the row's command line contains needle
// (case-insensitive).
func matchRow(r Row, needle string) bool {
	return strings.Contains(strings.ToLower(r.Proc.Cmdline), strings.ToLower(needle))
}

// findNext returns the index of the first row at or after start whose
// command line matches needle, wrapping around; -1 if none or empty needle.
func findNext(rows []Row, needle string, start int) int {
	if needle == "" || len(rows) == 0 {
		return -1
	}
	for i := 0; i < len(rows); i++ {
		idx := (start + i) % len(rows)
		if matchRow(rows[idx], needle) {
			return idx
		}
	}
	return -1
}

// handleSearchKey drives ModeSearch: incremental jump to the next match.
func (a *App) handleSearchKey(ev *tcell.EventKey) {
	switch {
	case ev.Key() == tcell.KeyEscape, ev.Key() == tcell.KeyEnter:
		a.mode = ModeNormal
		a.input = ""
		return
	case ev.Key() == tcell.KeyBackspace, ev.Key() == tcell.KeyBackspace2:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
	case ev.Rune() != 0:
		a.input += string(ev.Rune())
	}
	if idx := findNext(a.rows, a.input, a.table.Sel); idx >= 0 {
		a.table.Sel = idx
	}
}

// handleFilterKey drives ModeFilter: live-narrow the table while typing.
func (a *App) handleFilterKey(ev *tcell.EventKey) {
	switch {
	case ev.Key() == tcell.KeyEscape:
		a.filter = ""
		a.input = ""
		a.mode = ModeNormal
	case ev.Key() == tcell.KeyEnter:
		a.filter = a.input
		a.input = ""
		a.mode = ModeNormal
	case ev.Key() == tcell.KeyBackspace, ev.Key() == tcell.KeyBackspace2:
		if len(a.input) > 0 {
			a.input = a.input[:len(a.input)-1]
		}
		a.filter = a.input
	case ev.Rune() != 0:
		a.input += string(ev.Rune())
		a.filter = a.input
	}
	a.rebuild()
}
```

- [ ] **Step 4: Wire the modes into app.go**

4a. Replace the whole `rebuild` function with (final form — filter, then tree or sort):

```go
// rebuild recomputes a.rows from a.snap: filter, then tree or sort, then
// keep the selection on the same PID when possible. An active filter
// bypasses tree mode (flat list), like htop.
func (a *App) rebuild() {
	if a.snap == nil {
		a.rows = nil
		return
	}
	selPID := 0
	if len(a.rows) > 0 && a.table.Sel < len(a.rows) {
		selPID = a.rows[a.table.Sel].Proc.PID
	}
	needle := strings.ToLower(a.filter)
	procs := make([]proc.Process, 0, len(a.snap.Procs))
	for _, p := range a.snap.Procs {
		if needle != "" && !strings.Contains(strings.ToLower(p.Cmdline), needle) {
			continue
		}
		procs = append(procs, p)
	}
	if a.treeMode && needle == "" {
		a.rows = buildTreeRows(procs, a.sortBy, a.sortDesc)
	} else {
		sortProcs(procs, a.sortBy, a.sortDesc)
		rows := make([]Row, 0, len(procs))
		for _, p := range procs {
			rows = append(rows, Row{Proc: p})
		}
		a.rows = rows
	}
	a.table.ClampTo(len(a.rows))
	if selPID != 0 {
		for i := range a.rows {
			if a.rows[i].Proc.PID == selPID {
				a.table.Sel = i
				break
			}
		}
	}
}
```

Add `"strings"` to the import block of `app.go`.

4b. Replace the whole `handleKey` function with:

```go
func (a *App) handleKey(ev *tcell.EventKey) {
	a.status = ""
	switch a.mode {
	case ModeNormal:
		a.handleNormalKey(ev)
	case ModeSearch:
		a.handleSearchKey(ev)
	case ModeFilter:
		a.handleFilterKey(ev)
	}
}
```

4c. Replace the whole `handleNormalKey` function with:

```go
func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	case ev.Key() == tcell.KeyF3:
		a.mode = ModeSearch
		a.input = ""
	case ev.Key() == tcell.KeyF4:
		a.mode = ModeFilter
		a.input = a.filter
	case ev.Key() == tcell.KeyF5:
		a.treeMode = !a.treeMode
		a.rebuild()
	case ev.Rune() == 'P':
		a.setSort(SortCPU)
	case ev.Rune() == 'M':
		a.setSort(SortMem)
	case ev.Rune() == 'T':
		a.setSort(SortTime)
	case ev.Rune() == 'N':
		a.setSort(SortPID)
	case ev.Rune() == 'I':
		a.sortDesc = !a.sortDesc
		a.rebuild()
	}
}
```

4d. Replace the whole `drawBottom` function with:

```go
// drawBottom paints the bottom row: search/filter prompt when typing,
// else the status line if set, else the fn-key bar.
func (a *App) drawBottom(w, h int) {
	y := h - 1
	switch a.mode {
	case ModeSearch:
		drawString(a.screen, 0, y, styleDefault, "Search: "+a.input)
		return
	case ModeFilter:
		drawString(a.screen, 0, y, styleDefault, "Filter: "+a.input)
		return
	}
	if a.status != "" {
		drawString(a.screen, 0, y, styleStatus, a.status)
		return
	}
	x := 0
	for _, item := range fnBarItems {
		x = drawString(a.screen, x, y, styleFnKey, item.Key)
		x = drawString(a.screen, x, y, styleFnLabel, item.Label+" ")
	}
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: incremental search (F3) and live filter (F4)" && git push origin main
```

---

### Task 11: Signal menu (F9 kill) and renice (F7/F8)

**Files:**
- Create: `internal/ui/menu.go`
- Modify: `internal/ui/app.go` (replace `draw`, `handleKey`, `handleNormalKey`; add `headerHeight`)
- Test: `internal/ui/menu_test.go`

`F9` opens a small signal menu over the table (default selection SIGTERM); `Enter` sends the signal to the selected process, `Esc` cancels. `F7`/`F8` decrease/increase the nice value by 1. Failures (e.g. EPERM on other users' processes) must NEVER crash — they go to the status line.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/menu_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/ui/
```

Expected: FAIL (compile error — `signalEntries` undefined).

- [ ] **Step 3: Create menu.go**

Create `internal/ui/menu.go`:

```go
package ui

import (
	"fmt"
	"syscall"

	"github.com/gdamore/tcell/v2"
)

// signalEntries is the F9 menu, a subset of htop's list.
// Index 3 (SIGTERM) is the default selection.
var signalEntries = []struct {
	Num  int
	Name string
}{
	{1, "SIGHUP"}, {2, "SIGINT"}, {9, "SIGKILL"},
	{15, "SIGTERM"}, {18, "SIGCONT"}, {19, "SIGSTOP"},
}

// drawSignalMenu paints the menu box at the left edge, starting at row y0.
func (a *App) drawSignalMenu(y0 int) {
	const boxW = 16
	drawString(a.screen, 0, y0, styleMenuBox, pad("Send signal:", boxW, false))
	for i, e := range signalEntries {
		style := styleMenuBox
		if i == a.sigSel {
			style = styleMenuSel
		}
		label := fmt.Sprintf("%3d %s", e.Num, e.Name)
		drawString(a.screen, 0, y0+1+i, style, pad(label, boxW, false))
	}
}

// handleSignalsKey drives the F9 menu.
func (a *App) handleSignalsKey(ev *tcell.EventKey) {
	switch ev.Key() {
	case tcell.KeyEscape:
		a.mode = ModeNormal
	case tcell.KeyUp:
		if a.sigSel > 0 {
			a.sigSel--
		}
	case tcell.KeyDown:
		if a.sigSel < len(signalEntries)-1 {
			a.sigSel++
		}
	case tcell.KeyEnter:
		a.mode = ModeNormal
		if a.table.Sel < len(a.rows) {
			p := a.rows[a.table.Sel].Proc
			sig := signalEntries[a.sigSel]
			if err := syscall.Kill(p.PID, syscall.Signal(sig.Num)); err != nil {
				a.status = fmt.Sprintf("Cannot send %s to PID %d: %v", sig.Name, p.PID, err)
				return
			}
			a.refresh()
		}
	}
}

// renice changes the selected process's nice value by delta.
// Lowering nice (delta < 0) usually requires root; errors go to the status line.
func (a *App) renice(delta int) {
	if a.table.Sel >= len(a.rows) {
		return
	}
	p := a.rows[a.table.Sel].Proc
	newNice := int(p.Nice) + delta
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, p.PID, newNice); err != nil {
		a.status = fmt.Sprintf("Cannot renice PID %d: %v", p.PID, err)
		return
	}
	a.refresh()
}
```

- [ ] **Step 4: Wire the menu into app.go**

4a. Add this method after `drawHeader` usage — place it directly below the `draw` function:

```go
// headerHeight is the row count drawHeader uses (without drawing).
func (a *App) headerHeight() int {
	if a.snap == nil {
		return 0
	}
	return (len(a.snap.CPUs)+1)/2 + 3
}
```

4b. Replace the whole `draw` function with:

```go
// draw paints the whole screen.
func (a *App) draw() {
	a.screen.Clear()
	w, h := a.screen.Size()
	a.drawMain(w, h)
	if a.mode == ModeSignals {
		a.drawSignalMenu(a.headerHeight() + 1)
	}
	a.drawBottom(w, h)
	a.screen.Show()
}
```

4c. Replace the whole `handleKey` function with:

```go
func (a *App) handleKey(ev *tcell.EventKey) {
	a.status = ""
	switch a.mode {
	case ModeNormal:
		a.handleNormalKey(ev)
	case ModeSearch:
		a.handleSearchKey(ev)
	case ModeFilter:
		a.handleFilterKey(ev)
	case ModeSignals:
		a.handleSignalsKey(ev)
	}
}
```

4d. Replace the whole `handleNormalKey` function with:

```go
func (a *App) handleNormalKey(ev *tcell.EventKey) {
	_, h := a.screen.Size()
	pageRows := h - 10
	if pageRows < 1 {
		pageRows = 1
	}
	if a.table.HandleKey(ev, len(a.rows), pageRows) {
		return
	}
	switch {
	case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':
		a.quit = true
	case ev.Key() == tcell.KeyF3:
		a.mode = ModeSearch
		a.input = ""
	case ev.Key() == tcell.KeyF4:
		a.mode = ModeFilter
		a.input = a.filter
	case ev.Key() == tcell.KeyF5:
		a.treeMode = !a.treeMode
		a.rebuild()
	case ev.Key() == tcell.KeyF7:
		a.renice(-1)
	case ev.Key() == tcell.KeyF8:
		a.renice(1)
	case ev.Key() == tcell.KeyF9:
		if len(a.rows) > 0 {
			a.sigSel = 3 // SIGTERM
			a.mode = ModeSignals
		}
	case ev.Rune() == 'P':
		a.setSort(SortCPU)
	case ev.Rune() == 'M':
		a.setSort(SortMem)
	case ev.Rune() == 'T':
		a.setSort(SortTime)
	case ev.Rune() == 'N':
		a.setSort(SortPID)
	case ev.Rune() == 'I':
		a.sortDesc = !a.sortDesc
		a.rebuild()
	}
}
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS (including `TestKillSendsSignal` and `TestRenice`).

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: F9 signal menu and F7/F8 renice" && git push origin main
```

---

### Task 12: Help screen (F1)

**Files:**
- Create: `internal/ui/help.go`
- Modify: `internal/ui/app.go` (replace `draw`, `handleKey`, `handleNormalKey`)
- Test: `internal/ui/help_test.go`

`F1` or `h` shows a full-screen key reference; any key returns to the process list.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/help_test.go`:

```go
package ui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestHelpScreen(t *testing.T) {
	app, sim := newTestApp(t)
	app.handleKey(tcell.NewEventKey(tcell.KeyF1, 0, tcell.ModNone))
	if app.mode != ModeHelp {
		t.Fatal("F1 did not open help")
	}
	app.draw()
	text := screenText(sim)
	for _, want := range []string{"tree view", "sort by", "quit"} {
		if !strings.Contains(text, want) {
			t.Errorf("help missing %q", want)
		}
	}
	app.handleKey(tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone))
	if app.mode != ModeNormal {
		t.Error("key press did not close help")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/ui/ -run TestHelpScreen
```

Expected: FAIL (`ModeHelp` is never entered — F1 is not handled yet).

- [ ] **Step 3: Create help.go**

Create `internal/ui/help.go`:

```go
package ui

import "github.com/gdamore/tcell/v2"

var helpLines = []string{
	"htop-go — interactive process viewer for Linux",
	"",
	"  Arrows, PgUp/PgDn, Home/End   navigate the process list",
	"  P M T N    sort by CPU%, MEM%, TIME+, PID",
	"  I          invert the sort order",
	"  F5         toggle tree view",
	"  F3         incremental search (Enter/Esc to leave)",
	"  F4         filter the list (Enter keeps it, Esc clears it)",
	"  F7 / F8    decrease / increase nice value",
	"  F9         send a signal to the selected process",
	"  F1 or h    this help screen",
	"  F10 or q   quit",
	"",
	"Press any key to return.",
}

// drawHelp paints the full-screen help.
func (a *App) drawHelp(w, h int) {
	for i, line := range helpLines {
		if i >= h {
			break
		}
		drawString(a.screen, 0, i, styleDefault, line)
	}
}

// handleHelpKey leaves help on any key.
func (a *App) handleHelpKey(ev *tcell.EventKey) {
	a.mode = ModeNormal
}
```

- [ ] **Step 4: Wire help into app.go**

4a. Replace the whole `draw` function with:

```go
// draw paints the whole screen.
func (a *App) draw() {
	a.screen.Clear()
	w, h := a.screen.Size()
	if a.mode == ModeHelp {
		a.drawHelp(w, h)
		a.screen.Show()
		return
	}
	a.drawMain(w, h)
	if a.mode == ModeSignals {
		a.drawSignalMenu(a.headerHeight() + 1)
	}
	a.drawBottom(w, h)
	a.screen.Show()
}
```

4b. Replace the whole `handleKey` function with:

```go
func (a *App) handleKey(ev *tcell.EventKey) {
	a.status = ""
	switch a.mode {
	case ModeNormal:
		a.handleNormalKey(ev)
	case ModeSearch:
		a.handleSearchKey(ev)
	case ModeFilter:
		a.handleFilterKey(ev)
	case ModeSignals:
		a.handleSignalsKey(ev)
	case ModeHelp:
		a.handleHelpKey(ev)
	}
}
```

4c. In `handleNormalKey`, add this case directly after the `case ev.Key() == tcell.KeyF10, ev.Rune() == 'q':` block's `a.quit = true` line:

```go
	case ev.Key() == tcell.KeyF1, ev.Rune() == 'h':
		a.mode = ModeHelp
```

- [ ] **Step 5: Run all tests**

```bash
go test ./... && go build ./...
```

Expected: all PASS.

- [ ] **Step 6: Format, vet, commit, push**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./...
git add -A && git commit -m "feat: help screen on F1" && git push origin main
```

---

### Task 13: CI + tag-driven release builds, tag v0.1.0

**Files:**
- Create: `.github/workflows/ci.yml`, `.github/workflows/release.yml`

- [ ] **Step 1: Create ci.yml**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - name: gofmt
        run: test -z "$(gofmt -l .)"
      - name: vet
        run: go vet ./...
      - name: test
        run: go test ./...
      - name: build
        run: go build ./...
```

- [ ] **Step 2: Create release.yml**

Create `.github/workflows/release.yml` (Linux-only binaries — htop-go reads /proc):

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        include:
          - goos: linux
            goarch: amd64
          - goos: linux
            goarch: arm64
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: 'stable'
      - name: Build
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: '0'
        run: |
          mkdir -p dist
          go build -trimpath -ldflags "-s -w -X main.version=${GITHUB_REF_NAME#v}" -o "dist/htop-go-${{ matrix.goos }}-${{ matrix.goarch }}" .
      - uses: actions/upload-artifact@v4
        with:
          name: binaries-${{ matrix.goos }}-${{ matrix.goarch }}
          path: dist/*
          if-no-files-found: error

  release:
    name: Create GitHub Release
    needs: build
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/download-artifact@v4
        with:
          path: artifacts
          merge-multiple: true
      - name: Generate SHA-256 checksums
        run: |
          cd artifacts
          sha256sum htop-go-* > SHA256SUMS.txt
          cat SHA256SUMS.txt
      - name: Generate release notes
        run: |
          PREV_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
          {
            if [ -n "$PREV_TAG" ]; then
              echo "## Changes since $PREV_TAG"
            else
              echo "## Changes"
            fi
            echo ""
            if [ -n "$PREV_TAG" ]; then
              git log --oneline --no-decorate "${PREV_TAG}..HEAD"
            else
              git log --oneline --no-decorate -20
            fi
          } > RELEASE_NOTES.md
          cat RELEASE_NOTES.md
      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ github.ref_name }}
          name: Release ${{ github.ref_name }}
          body_path: RELEASE_NOTES.md
          files: |
            artifacts/htop-go-*
            artifacts/SHA256SUMS.txt
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Commit, push, wait for CI green**

```bash
gofmt -w . && gofmt -l .   # must print nothing
go vet ./... && go test ./... && go build ./...
git add -A && git commit -m "ci: add CI and release workflows" && git push origin main
sleep 30 && gh run list --branch main --limit 3
```

Then watch the newest run until it completes:

```bash
gh run watch $(gh run list --branch main --limit 1 --json databaseId -q '.[0].databaseId') --exit-status
```

Expected: exit code 0 (CI green). If it fails, run `gh run view --log-failed`, fix the problem, commit, push, and watch again. Do NOT tag until CI is green.

- [ ] **Step 4: Tag v0.1.0 and verify the release**

```bash
git tag v0.1.0 && git push origin v0.1.0
sleep 30 && gh run watch $(gh run list --workflow Release --limit 1 --json databaseId -q '.[0].databaseId') --exit-status
gh release view v0.1.0
```

Expected: `gh release view v0.1.0` lists assets `htop-go-linux-amd64`, `htop-go-linux-arm64`, `SHA256SUMS.txt`. If a build target fails, fix it (or drop the target), delete and re-push the tag (`git tag -d v0.1.0 && git push origin :refs/tags/v0.1.0`), and re-tag. Report the release URL when done.

---

### Task 14: README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Inspect the finished project**

```bash
cd /home/user/projects/htop-go
ls -R internal/ && cat go.mod && ./htop-go --version 2>/dev/null || go run . --version
gh release view v0.1.0 --json assets -q '.assets[].name'
```

- [ ] **Step 2: Write README.md**

Write a comprehensive `README.md` covering, in this order:

1. **Title + one-liner**: `# htop-go` — htop rewritten in Go: an interactive process viewer for Linux. Mention it is a rewrite of [htop](https://github.com/htop-dev/htop) using [tcell](https://github.com/gdamore/tcell).
2. **Features**: per-core CPU bars with user/system split, Mem/Swp meters, load average, uptime, task counts; sortable process table (CPU%, MEM%, TIME+, PID); tree view; incremental search and live filter; kill via signal menu; renice; 1.5 s refresh (configurable with `-d`).
3. **Install**: download a binary from the GitHub release (linux-amd64 / linux-arm64), or `go install github.com/hermes98761234/htop-go@latest`.
4. **Usage**: run `htop-go`; flags `-d <tenths>` (refresh delay, default 15 = 1.5 s) and `--version`.
5. **Key bindings**: a table with every key from the help screen (arrows/PgUp/PgDn/Home/End, P/M/T/N, I, F1/h, F3, F4, F5, F7, F8, F9, F10/q).
6. **How it works**: short section — parses `/proc` directly (no cgo, no dependencies beyond tcell); CPU% is the share of a single core, so totals can reach `ncpu × 100%`; Linux-only by design.
7. **Development**: `go test ./...`, `go vet ./...`, `gofmt`; project layout (`internal/proc` = /proc parsing, `internal/ui` = tcell UI).
8. **License**: GPL-2.0, same spirit as the original htop.

- [ ] **Step 3: Commit and push**

```bash
git add README.md && git commit -m "docs: add README.md" && git push origin main
```

Verify with `gh repo view hermes98761234/htop-go` that the README renders. Report the repo URL.

---

## Self-review notes (done at plan time)

- Spec coverage: header meters (T6), table+nav (T7), sorting (T8), tree (T9), search/filter (T10), kill/nice (T11), help (T12), flags (T1/T5), CI/release (T13), README (T14). /proc parsing (T2-T4).
- Type consistency: `App` fields are declared once in T5 (`screen, scanner, delay, snap, rows, mode, input, filter, sigSel, status, quit`) and extended in T7 (`table`), T8 (`sortBy`, `sortDesc`), T9 (`treeMode`). `handleNormalKey`/`rebuild`/`draw`/`drawBottom`/`handleKey` are replaced WHOLE in each task that touches them, each time with the complete accumulated version.
- Workers must never run the TUI interactively — all verification is `go test` (SimulationScreen) + `go build`.

