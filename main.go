package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/hermes98761234/htop-go/internal/proc"
	"github.com/hermes98761234/htop-go/internal/ui"
)

var version = "0.1.0"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	delayTenths := flag.Int("d", 15, "delay between updates, in tenths of seconds")
	flag.Parse()
	if *showVersion {
		fmt.Printf("htop-go %s\n", version)
		return
	}
	if *delayTenths < 1 {
		*delayTenths = 1
	}
	screen, err := tcell.NewScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
	if err := screen.Init(); err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
	delay := time.Duration(*delayTenths) * 100 * time.Millisecond
	app := ui.NewApp(screen, proc.NewScanner(), delay)
	if err := app.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "htop-go:", err)
		os.Exit(1)
	}
}
