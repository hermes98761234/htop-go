# htop-go Design

**Goal:** `htop-go` — a rewrite of htop (https://github.com/htop-dev/htop) in Go. An interactive terminal process viewer for Linux: live header meters plus a sortable, navigable process table with kill/nice actions.

**Execution model:** Implemented entirely by Hermes kanban workers (board `htop-go`) from the plan in `docs/superpowers/plans/2026-06-12-htop-go-plan.md`. Workers are weaker models, so every task carries exact file paths, signatures, code for tricky logic, commands, and expected output.

## Decisions (made autonomously; alternatives noted)

| Decision | Choice | Rejected alternative & why |
|---|---|---|
| Language | Go (user-specified) | — |
| TUI library | `github.com/gdamore/tcell/v2` | Bubble Tea: Elm-style framework adds indirection; htop's dense per-cell bar/table layout maps directly onto tcell's `SetContent` grid, closer to the original's ncurses model and easier for weak workers to follow literal drawing code. |
| Process data | Parse `/proc` directly, Linux-only | `gopsutil`: convenient but hides the parsing, adds a large dependency, and makes CPU% sampling semantics opaque. Direct parsing gives pure, fixture-testable functions — and htop itself is /proc-native. |
| Repo name / binary | `htop-go` | Matches sibling convention (tig-rs, wrk-rs, svgo-rs). |
| Testing | Pure parser functions tested table-driven on fixture strings; format helpers unit-tested; UI verified by build + manual smoke (workers run in headless env) | tcell SimulationScreen tests: extra moving part for weak workers; parser coverage is where the real logic lives. |
| CPU% convention | htop default: percent of a single core (total may reach `ncpu × 100`) | — |
| Refresh rate | 1.5 s default, `-d` flag in tenths of a second like htop | — |

## Scope v0.1.0

**In:**
- Header: per-core CPU bar meters (two columns), Mem bar, Swp bar; right side: Tasks/threads/running counts, load average, uptime
- Process table: PID, USER, PRI, NI, VIRT, RES, SHR, S, CPU%, MEM%, TIME+, Command — htop colors, selection bar, scrolling (arrows, PgUp/PgDn, Home/End)
- Sorting: CPU% desc default; keys `P` (CPU), `M` (MEM%), `T` (TIME), `N` (PID), `I` invert
- `F5` tree view (parent/child with branch glyphs)
- `F3` incremental search, `F4` filter
- `F9` kill with signal menu, `F7`/`F8` nice down/up
- `F1` help screen, `F10`/`q` quit, bottom function-key bar
- `--version`, `-d <tenths>` delay flag

**Out (v0.1):** F2 setup/customization, mouse, non-Linux platforms, cgroup/container/GPU meters, strace/lsof integration, user filtering (`u`), per-process environment screen.

## Architecture

Single static binary. Two packages under `internal/`:

```
htop-go/
├── main.go                     # flags, terminal init, run app
├── internal/proc/              # ALL /proc parsing — pure functions + thin file readers
│   ├── system.go               # /proc/stat CPU ticks, /proc/meminfo, loadavg, uptime
│   ├── process.go              # /proc/[pid]/stat (comm-in-parens!), status, cmdline
│   ├── users.go                # uid → username cache (parses /etc/passwd)
│   └── scan.go                 # Scanner: full scan + CPU%/MEM% via tick deltas
└── internal/ui/                # tcell rendering + event loop
    ├── app.go                  # App struct, event loop, 1.5s ticker, key dispatch, modes
    ├── style.go                # color palette constants
    ├── draw.go                 # drawString/bar-meter helpers
    ├── header.go               # CPU/Mem/Swp meters + right column
    ├── table.go                # process table, selection, scrolling
    ├── format.go               # column formatting (sizes, TIME+, etc.) — unit-tested
    ├── sort.go                 # sort orders
    ├── tree.go                 # tree build + flatten
    ├── search.go               # F3/F4 input-line modes
    ├── menu.go                 # F9 signal menu
    └── help.go                 # F1 help screen
```

**Data flow:** ticker fires → `Scanner.Scan()` reads `/proc` → snapshot `[]Process` + `SystemStats` → UI sorts/filters/trees the slice → draws. The scanner owns previous-tick state for CPU% deltas. UI never reads `/proc` itself.

**CPU% math:** per process, `Δproc_ticks / Δtotal_ticks_per_cpu × 100` where `Δtotal_ticks_per_cpu = Δ(total system ticks) / ncpu`. First scan shows 0%.

**Error handling:** processes vanish mid-scan constantly — every per-PID read error means "skip PID silently". Kill/renice failures (EPERM) surface in a status line, never crash.

**Versioning/CI:** GitHub repo `hermes98761234/htop-go` created in Task 1; every task pushes to `main`. CI (gofmt/vet/test) + tag-driven release matrix (linux & darwin, amd64 & arm64, CGO disabled) before README; v0.1.0 tagged with verified artifacts.
