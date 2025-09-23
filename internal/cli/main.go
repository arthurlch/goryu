package cli

import (
	"fmt"
	"os"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Usage       string
	Action      func([]string) error
	Subcommands []*Command
}

// Run executes the CLI
func Run() {
	if len(os.Args) < 2 {
		// Try to run TUI, it will fallback to help if needed
		RunTUI()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		if err := runInit(args); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		if err := runGenerate(args); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "config":
		if err := runConfigCommand(args); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "validate":
		if err := runValidate(args); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "version":
		showVersion()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
}

func runGenerate(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: goryu generate <type> <name> [options]")
		fmt.Println("\nTypes:")
		fmt.Println("  handler     Generate HTTP handler [--type=basic|crud|api]")
		fmt.Println("  middleware  Generate middleware")
		fmt.Println("  model       Generate data model [--type=basic|db] [--db-tool=sqlc|ent|gorm]")
		fmt.Println("\nExamples:")
		fmt.Println("  goryu generate handler users --type=crud")
		fmt.Println("  goryu generate handler auth --type=api")
		fmt.Println("  goryu generate middleware cors")
		fmt.Println("  goryu generate model user --type=basic")
		fmt.Println("  goryu generate model user --type=db --db-tool=sqlc")
		fmt.Println("  goryu generate model user --type=db --db-tool=ent")
		fmt.Println("  goryu generate model user --type=db --db-tool=gorm")
		return fmt.Errorf("generate type required")
	}

	generateType := args[0]
	generateArgs := args[1:]

	switch generateType {
	case "handler":
		return runGenerateHandler(generateArgs)
	case "middleware":
		return runGenerateMiddleware(generateArgs)
	case "model":
		return runGenerateModel(generateArgs)
	default:
		return fmt.Errorf("unknown generate type: %s", generateType)
	}
}

func runConfigCommand(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: goryu config <subcommand>")
		fmt.Println("Subcommands: init, show, validate, set, get")
		return fmt.Errorf("config subcommand required")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "init":
		return runConfigInit(subargs)
	case "show":
		return runConfigShow(subargs)
	case "validate":
		return runConfigValidate(subargs)
	case "set":
		return runConfigSet(subargs)
	case "get":
		return runConfigGet(subargs)
	default:
		return fmt.Errorf("unknown config subcommand: %s", subcommand)
	}
}

func showHelp() {
	fmt.Printf("Goryu CLI v1.0.0 - A powerful web framework for Go\n\n")
	fmt.Println("Usage: goryu <command> [arguments]")
	fmt.Println("       goryu             (interactive TUI mode)")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  init      Initialize a new Goryu project")
	fmt.Println("  generate  Generate boilerplate code")
	fmt.Println("  config    Manage application configuration")
	fmt.Println("  validate  Validate project setup and configuration")
	fmt.Println("  version   Show version information")
	fmt.Println("  help      Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  goryu                    # Interactive TUI mode")
	fmt.Println("  goryu init my-app --template=api")
	fmt.Println("  goryu generate handler users")
	fmt.Println("  goryu config init --type=web")
	fmt.Println("  goryu validate")
}

func showVersion() {
	fmt.Println("Goryu CLI v1.0.0")
	fmt.Println("A GOated web framework")
}
