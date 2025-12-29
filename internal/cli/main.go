package cli

import (
	"fmt"
	"os"
)

const VERSION = "0.1.0"

func Run() {
	cli := NewCLI(VERSION)
	InitializeCommands(cli)

	args := os.Args[1:]
	if err := cli.Run(args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
