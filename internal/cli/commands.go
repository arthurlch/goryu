package cli

// So I can have all commands in one place here, easy to understand
func InitializeCommands(cli *CLI) {
	// Project commands
	cli.RegisterCommand(newInitCommand())
	cli.RegisterCommand(newGenerateCommand())
	cli.RegisterCommand(newScaffoldCommand())
	
	// Development commands
	cli.RegisterCommand(newDevCommand())
	cli.RegisterCommand(newBuildCommand())
	
	// Management commands
	cli.RegisterCommand(newMiddlewareCommand())
	cli.RegisterCommand(newConfigCommand())
	cli.RegisterCommand(newRoutesCommand())
	
	// Utility commands
	cli.RegisterCommand(newVersionCommand())
	cli.RegisterCommand(newValidateCommand())
	
	// Aliases
	cli.RegisterCommand(newGenerateAliasCommand())
}