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
