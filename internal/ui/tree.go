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
