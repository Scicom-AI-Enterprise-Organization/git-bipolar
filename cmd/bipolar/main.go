package main

import (
	"fmt"
	"os"

	"bipolar/internal/shell"
	"bipolar/internal/ui"
)

func main() {
	if err := shell.EnsureInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Setup error: %v\n", err)
		os.Exit(1)
	}

	if err := ui.RunMainMenu(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
