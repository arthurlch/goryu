package cli

// Package cli provides command-line interface functionality

func newGenerateCommand() *Command {
	return &Command{
		Name:        "generate",
		Description: "Generate boilerplate code",
		Usage:       "goryu generate <type> <name> [options]",
		Subcommands: []*Command{
			{
				Name:        "handler",
				Description: "Generate a new HTTP handler",
				Usage:       "goryu generate handler <name> [--type=basic|crud|api] [--path=internal/handlers]",
				Action:      runGenerateHandler,
			},
			{
				Name:        "middleware",
				Description: "Generate a new middleware",
				Usage:       "goryu generate middleware <name> [--path=internal/middleware]",
				Action:      runGenerateMiddleware,
			},
			{
				Name:        "model",
				Description: "Generate a new model",
				Usage:       "goryu generate model <name> [--type=basic|db] [--db-tool=sqlc|ent|gorm] [--path=internal/models]",
				Action:      runGenerateModel,
			},
		},
	}
}
