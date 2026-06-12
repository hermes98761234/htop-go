# htop-go

htop rewritten in Go: an interactive process viewer for Linux. A rewrite of [htop](https://github.com/htop-dev/htop) using [tcell](https://github.com/gdamore/tcell).

## Features

- **Header meters**: per-core CPU bars with user/system split, Mem/Swp meters, load average, uptime, and task counts
- **Process table**: sortable by CPU%, MEM%, TIME+, and PID; invert sort order support
- **Tree view**: hierarchical process display with Unicode branch prefixes
- **Incremental search**: find processes by name with live highlighting
- **Live filter**: narrow the process list to matching entries
- **Kill**: send signals to the selected process via a signal menu (F9)
- **Renice**: adjust process priority with F7/F8
- **Refresh**: 1.5 s default, configurable with `-d`

## Install

Download a pre-built binary from the [GitHub release](https://github.com/hermes98761234/htop-go/releases/latest):

- `htop-go-linux-amd64`
- `htop-go-linux-arm64`

Or install with `go install`:

```
go install github.com/hermes98761234/htop-go@latest
```

## Usage

Run `htop-go` with no arguments to start the interactive viewer.

Flags:

| Flag | Description |
|------|-------------|
| `-d <tenths>` | Refresh delay in tenths of a second (default `15` = 1.5 s) |
| `--version` | Print version and exit |

## Key bindings

| Key | Action |
|-----|--------|
| ↑ ↓ ← →, PgUp/PgDn, Home/End | Navigate the process list |
| P | Sort by CPU% |
| M | Sort by MEM% |
| T | Sort by TIME+ |
| N | Sort by PID |
| I | Invert sort order |
| F1 or h | Help screen |
| F3 | Incremental search (Enter/Esc to leave) |
| F4 | Filter the list (Enter keeps it, Esc clears it) |
| F5 | Toggle tree view |
| F7 | Decrease nice value (lower priority) |
| F8 | Increase nice value (higher priority) |
| F9 | Send a signal to the selected process |
| F10 or q | Quit |

## How it works

htop-go reads process data directly from `/proc` — no cgo, no external dependencies beyond tcell. CPU usage is reported as the share of a single core, so total CPU% can reach `ncpu × 100%`. Linux-only by design.

## Development

```
go test ./...
go vet ./...
gofmt -w .
```

Project layout:

- `internal/proc` — `/proc` parsing (process listing, system info, user resolution)
- `internal/ui` — tcell-based terminal UI (header, table, tree, search, help, menus)

## License

GPL-2.0 — same spirit as the original htop.
