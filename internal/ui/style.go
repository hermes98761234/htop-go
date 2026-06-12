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
