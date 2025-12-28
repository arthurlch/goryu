package cli

// facto pattern is good here to keep the main CLI setup clean and modular

func newScaffoldCommand() *Command {
	scaffoldCmd := &Command{
		Name:        "scaffold",
		Description: "Scaffold complete features",
		Usage:       "goryu scaffold <feature> [name] [flags]",
		Subcommands: make(map[string]*Command),
	}

	scaffoldCmd.Subcommands["api"] = &Command{
		Name:        "api",
		Description: "Scaffold complete REST API",
		Usage:       "goryu scaffold api <resource> [flags]",
		Flags: []Flag{
			{Name: "fields", Shorthand: "f", Description: "Resource fields", Required: true},
			{Name: "db", Description: "Include database layer", Default: true},
			{Name: "auth", Description: "Add authentication", Default: false},
			{Name: "validation", Description: "Add validation", Default: true},
			{Name: "tests", Description: "Generate tests", Default: true},
		},
		Action: cmdScaffoldAPI,
	}

	scaffoldCmd.Subcommands["service"] = &Command{
		Name:        "service",
		Description: "Scaffold microservice",
		Usage:       "goryu scaffold service <name> [flags]",
		Flags: []Flag{
			{Name: "grpc", Description: "Include gRPC support", Default: false},
			{Name: "http", Description: "Include HTTP support", Default: true},
			{Name: "kafka", Description: "Include Kafka support", Default: false},
			{Name: "monitoring", Description: "Include monitoring", Default: true},
		},
		Action: cmdScaffoldService,
	}

	return scaffoldCmd
}

func newDevCommand() *Command {
	return &Command{
		Name:        "dev",
		Description: "Development tools",
		Usage:       "goryu dev [flags]",
		Flags: []Flag{
			{Name: "port", Shorthand: "p", Description: "Server port", Default: 3000},
			{Name: "hot-reload", Description: "Enable hot reload", Default: true},
			{Name: "debug", Shorthand: "d", Description: "Enable debug mode", Default: false},
		},
		Action: cmdDev,
	}
}

func newBuildCommand() *Command {
	return &Command{
		Name:        "build",
		Description: "Build the application",
		Usage:       "goryu build [flags]",
		Flags: []Flag{
			{Name: "output", Shorthand: "o", Description: "Output binary name", Default: "server"},
			{Name: "target", Shorthand: "t", Description: "Build target (development, production)", Default: "production"},
			{Name: "compress", Description: "Compress binary", Default: false},
			{Name: "static", Description: "Build static binary", Default: true},
		},
		Action: cmdBuild,
	}
}

func newMiddlewareCommand() *Command {
	middlewareCmd := &Command{
		Name:        "middleware",
		Description: "Manage middleware",
		Usage:       "goryu middleware <subcommand>",
		Subcommands: make(map[string]*Command),
	}

	middlewareCmd.Subcommands["list"] = &Command{
		Name:        "list",
		Description: "List available middleware",
		Action:      cmdMiddlewareList,
	}

	middlewareCmd.Subcommands["info"] = &Command{
		Name:        "info",
		Description: "Show middleware information",
		Usage:       "goryu middleware info <name>",
		Action:      cmdMiddlewareInfo,
	}

	return middlewareCmd
}

func newConfigCommand() *Command {
	configCmd := &Command{
		Name:        "config",
		Description: "Manage configuration",
		Usage:       "goryu config <subcommand>",
		Subcommands: make(map[string]*Command),
	}

	configCmd.Subcommands["init"] = &Command{
		Name:        "init",
		Description: "Initialize configuration",
		Flags: []Flag{
			{Name: "type", Shorthand: "t", Description: "Config type", Default: "server"},
			{Name: "format", Shorthand: "f", Description: "Config format", Default: "json"},
		},
		Action: cmdConfigInit,
	}

	configCmd.Subcommands["validate"] = &Command{
		Name:        "validate",
		Description: "Validate configuration",
		Flags: []Flag{
			{Name: "file", Shorthand: "f", Description: "Config file", Default: "config.json"},
		},
		Action: cmdConfigValidate,
	}

	configCmd.Subcommands["migrate"] = &Command{
		Name:        "migrate",
		Description: "Migrate configuration format",
		Flags: []Flag{
			{Name: "from", Description: "Source format", Required: true},
			{Name: "to", Description: "Target format", Required: true},
		},
		Action: cmdConfigMigrate,
	}

	return configCmd
}

func newRoutesCommand() *Command {
	routeCmd := &Command{
		Name:        "routes",
		Description: "Manage routes",
		Usage:       "goryu routes <subcommand>",
		Subcommands: make(map[string]*Command),
	}

	routeCmd.Subcommands["list"] = &Command{
		Name:        "list",
		Description: "List all routes",
		Flags: []Flag{
			{Name: "format", Shorthand: "f", Description: "Output format (table, json)", Default: "table"},
			{Name: "filter", Description: "Filter routes by pattern"},
		},
		Action: cmdRoutesList,
	}

	routeCmd.Subcommands["test"] = &Command{
		Name:        "test",
		Description: "Test route matching",
		Usage:       "goryu routes test <path> [method]",
		Action:      cmdRoutesTest,
	}

	return routeCmd
}

func newVersionCommand() *Command {
	return &Command{
		Name:        "version",
		Description: "Show version information",
		Action:      cmdVersion,
	}
}

// Special handling for the generate alias
func newGenerateAliasCommand() *Command {
	generateCmd := newGenerateCommand()
	return &Command{
		Name:        "g",
		Description: "Alias for generate",
		Usage:       "goryu g <type> [name] [flags]",
		Subcommands: generateCmd.Subcommands,
	}
}
