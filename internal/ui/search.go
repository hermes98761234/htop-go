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
