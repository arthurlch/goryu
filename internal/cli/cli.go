package cli

import (
	"fmt"
	"strings"
)

// Any good framework needs a good CLI.
// There is improvement to be made here, but for now it works well enough for my (our) needs.

type CLI struct {
	commands map[string]*Command
	version  string
	commit   string
	date     string
}

type Command struct {
	Name        string
	Description string
	Usage       string
	Flags       []Flag
	Action      func(*Context) error
	Subcommands map[string]*Command
}

type Flag struct {
	Name        string
	Shorthand   string
	Description string
	Default     interface{}
	Required    bool
}

type Context struct {
	Args   []string
	Flags  map[string]string
	Global map[string]interface{}
}

func NewCLI(version string) *CLI {
	return &CLI{
		commands: make(map[string]*Command),
		version:  version,
	}
}

func (c *CLI) RegisterCommand(cmd *Command) {
	c.commands[cmd.Name] = cmd
}

func (c *CLI) Run(args []string) error {
	if len(args) < 1 {
		// TUI !
		return RunEnhancedTUI()
	}

	globalFlags := make(map[string]string)
	filteredArgs := []string{}

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			parts := strings.SplitN(args[i], "=", 2)
			key := strings.TrimPrefix(parts[0], "--")
			value := "true"
			if len(parts) > 1 {
				value = parts[1]
			}
			globalFlags[key] = value
		} else if strings.HasPrefix(args[i], "-") && len(args[i]) == 2 {
			// Short flag
			key := string(args[i][1])
			value := "true"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
			globalFlags[key] = value
		} else {
			filteredArgs = append(filteredArgs, args[i])
		}
	}

	if _, ok := globalFlags["help"]; ok {
		c.showHelp()
		return nil
	}
	if _, ok := globalFlags["h"]; ok {
		c.showHelp()
		return nil
	}
	if _, ok := globalFlags["version"]; ok {
		c.showVersion()
		return nil
	}
	if _, ok := globalFlags["v"]; ok {
		c.showVersion()
		return nil
	}

	if len(filteredArgs) == 0 {
		c.showHelp()
		return nil
	}

	cmdName := filteredArgs[0]
	cmd, exists := c.commands[cmdName]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	ctx := &Context{
		Args:   filteredArgs[1:],
		Flags:  globalFlags,
		Global: make(map[string]interface{}),
	}

	return c.executeCommand(cmd, ctx)
}

func (c *CLI) executeCommand(cmd *Command, ctx *Context) error {
	if len(ctx.Args) > 0 && len(cmd.Subcommands) > 0 {
		subCmdName := ctx.Args[0]
		if subCmd, exists := cmd.Subcommands[subCmdName]; exists {
			ctx.Args = ctx.Args[1:]
			return c.executeCommand(subCmd, ctx)
		}
	}

	cmdArgs, cmdFlags := parseCommandArgs(ctx.Args)
	for k, v := range cmdFlags {
		ctx.Flags[k] = v
	}
	ctx.Args = cmdArgs

	if cmd.Action != nil {
		return cmd.Action(ctx)
	}

	c.showCommandHelp(cmd)
	return nil
}

func parseCommandArgs(args []string) ([]string, map[string]string) {
	flags := make(map[string]string)
	cmdArgs := []string{}

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			parts := strings.SplitN(args[i], "=", 2)
			key := strings.TrimPrefix(parts[0], "--")
			value := "true"
			if len(parts) > 1 {
				value = parts[1]
			}
			flags[key] = value
		} else if strings.HasPrefix(args[i], "-") && len(args[i]) == 2 {
			key := string(args[i][1])
			value := "true"
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++
			}
			flags[key] = value
		} else {
			cmdArgs = append(cmdArgs, args[i])
		}
	}

	return cmdArgs, flags
}

func (c *CLI) showHelp() {
	fmt.Printf("Goryu CLI v%s - A powerful web framework for Go\n\n", c.version)
	fmt.Println("Usage: goryu <command> [arguments]")
	fmt.Println("       goryu             (interactive TUI mode)")
	fmt.Println("\nAvailable commands:")

	groups := map[string][]*Command{
		"Project":         {},
		"Code Generation": {},
		"Development":     {},
		"Other":           {},
	}

	for _, cmd := range c.commands {
		switch cmd.Name {
		case "init", "new":
			groups["Project"] = append(groups["Project"], cmd)
		case "generate", "g":
			groups["Code Generation"] = append(groups["Code Generation"], cmd)
		case "dev", "serve", "build", "test":
			groups["Development"] = append(groups["Development"], cmd)
		default:
			groups["Other"] = append(groups["Other"], cmd)
		}
	}

	for group, cmds := range groups {
		if len(cmds) > 0 {
			fmt.Printf("\n%s:\n", group)
			for _, cmd := range cmds {
				fmt.Printf("  %-12s %s\n", cmd.Name, cmd.Description)
			}
		}
	}

	fmt.Println("\nUse 'goryu <command> --help' for more information about a command.")
}

func (c *CLI) showCommandHelp(cmd *Command) {
	fmt.Printf("%s - %s\n\n", cmd.Name, cmd.Description)
	if cmd.Usage != "" {
		fmt.Printf("Usage: %s\n\n", cmd.Usage)
	}

	if len(cmd.Subcommands) > 0 {
		fmt.Println("Available subcommands:")
		for _, sub := range cmd.Subcommands {
			fmt.Printf("  %-12s %s\n", sub.Name, sub.Description)
		}
		fmt.Println()
	}

	if len(cmd.Flags) > 0 {
		fmt.Println("Flags:")
		for _, flag := range cmd.Flags {
			flagStr := fmt.Sprintf("--%s", flag.Name)
			if flag.Shorthand != "" {
				flagStr = fmt.Sprintf("-%s, --%s", flag.Shorthand, flag.Name)
			}
			fmt.Printf("  %-20s %s", flagStr, flag.Description)
			if flag.Default != nil {
				fmt.Printf(" (default: %v)", flag.Default)
			}
			if flag.Required {
				fmt.Printf(" [required]")
			}
			fmt.Println()
		}
	}
}

func (c *CLI) showVersion() {
	fmt.Printf("Goryu CLI v%s\n", c.version)
	if c.commit != "" && c.commit != "none" {
		fmt.Printf("commit: %s\n", c.commit)
	}
	if c.date != "" && c.date != "unknown" {
		fmt.Printf("built:  %s\n", c.date)
	}
	fmt.Println("A GOated web framework")
}
