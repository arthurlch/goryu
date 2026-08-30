package cli

import (
	"fmt"
	"os"
)

// Build metadata, overridden at release time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func Run() {
	cli := NewCLI(version)
	cli.commit = commit
	cli.date = date
	InitializeCommands(cli)

	args := os.Args[1:]
	if err := cli.Run(args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
