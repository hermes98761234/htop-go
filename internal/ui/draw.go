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
