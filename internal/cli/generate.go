package cli

// facto too here

func newGenerateCommand() *Command {
	generateCmd := &Command{
		Name:        "generate",
		Description: "Generate code using templates",
		Usage:       "goryu generate <type> [name] [flags]",
		Subcommands: make(map[string]*Command),
	}

	generateCmd.Subcommands["handler"] = &Command{
		Name:        "handler",
		Description: "Generate HTTP handler",
		Usage:       "goryu generate handler <name> [flags]",
		Flags: []Flag{
			{Name: "type", Shorthand: "t", Description: "Handler type (basic, crud, api, websocket)", Default: "basic"},
			{Name: "path", Shorthand: "p", Description: "Output path", Default: "internal/handlers"},
			{Name: "model", Description: "Associated model name"},
			{Name: "middleware", Description: "Middleware to apply (comma-separated)"},
			{Name: "route", Description: "Route pattern", Default: "/{name}"},
		},
		Action: cmdGenerateHandler,
	}

	generateCmd.Subcommands["middleware"] = &Command{
		Name:        "middleware",
		Description: "Generate middleware",
		Usage:       "goryu generate middleware <name> [flags]",
		Flags: []Flag{
			{Name: "type", Shorthand: "t", Description: "Middleware type (standard, builder, plugin)", Default: "builder"},
			{Name: "path", Shorthand: "p", Description: "Output path", Default: "internal/middleware"},
			{Name: "global", Description: "Make middleware global", Default: false},
		},
		Action: cmdGenerateMiddleware,
	}

	generateCmd.Subcommands["model"] = &Command{
		Name:        "model",
		Description: "Generate data model",
		Usage:       "goryu generate model <name> [flags]",
		Flags: []Flag{
			{Name: "type", Shorthand: "t", Description: "Model type (basic, db)", Default: "basic"},
			{Name: "db-tool", Description: "Database tool (gorm, sqlc, ent)", Default: "gorm"},
			{Name: "fields", Shorthand: "f", Description: "Model fields (comma-separated)"},
			{Name: "path", Shorthand: "p", Description: "Output path", Default: "internal/models"},
		},
		Action: cmdGenerateModel,
	}

	generateCmd.Subcommands["route"] = &Command{
		Name:        "route",
		Description: "Generate route configuration",
		Usage:       "goryu generate route <name> [flags]",
		Flags: []Flag{
			{Name: "builder", Shorthand: "b", Description: "Use route builder pattern", Default: true},
			{Name: "group", Shorthand: "g", Description: "Route group prefix"},
			{Name: "middleware", Shorthand: "m", Description: "Route middleware (comma-separated)"},
			{Name: "methods", Description: "HTTP methods (comma-separated)", Default: "GET,POST,PUT,DELETE"},
		},
		Action: cmdGenerateRoute,
	}

	generateCmd.Subcommands["config"] = &Command{
		Name:        "config",
		Description: "Generate configuration code",
		Usage:       "goryu generate config <name> [flags]",
		Flags: []Flag{
			{Name: "builder", Shorthand: "b", Description: "Use config builder pattern", Default: true},
			{Name: "type", Shorthand: "t", Description: "Config type (server, database, cache, etc.)", Default: "server"},
			{Name: "format", Shorthand: "f", Description: "Config format (json, yaml, toml, env)", Default: "json"},
		},
		Action: cmdGenerateConfig,
	}

	return generateCmd
}
