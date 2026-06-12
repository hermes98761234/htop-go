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
