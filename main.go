package main

import (
	"flag"
	"fmt"
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
	_ = delayTenths
	fmt.Println("htop-go: TUI not implemented yet")
}
