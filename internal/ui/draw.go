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
