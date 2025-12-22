package cli

import (
	"github.com/charmbracelet/lipgloss"
)

// Colors and styles shared across TUI components
var (
	primaryColor   = lipgloss.Color("#7C3AED") // Purple
	secondaryColor = lipgloss.Color("#10B981") // Green
	accentColor    = lipgloss.Color("#F59E0B") // Amber
	textColor      = lipgloss.Color("#F9FAFB") // Light gray
	mutedColor     = lipgloss.Color("#9CA3AF") // Gray
	dangerColor    = lipgloss.Color("#EF4444") // Red

	// Base styles
	titleStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			MarginTop(1).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginBottom(2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 2).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Padding(0, 2)

	helpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	accentStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)
