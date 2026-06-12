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
