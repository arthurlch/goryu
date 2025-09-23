package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Colors and styles
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

// MainMenuModel represents the main menu state
type MainMenuModel struct {
	choices  []string
	cursor   int
	selected map[int]struct{}
}

type menuChoice int

const (
	menuInit menuChoice = iota
	menuGenerate
	menuConfig
	menuValidate
	menuExit
)

func initialMainMenuModel() MainMenuModel {
	return MainMenuModel{
		choices: []string{
			"🚀 Initialize new project",
			"⚡ Generate code",
			"⚙️  Manage configuration",
			"✅ Validate project",
			"👋 Exit",
		},
		selected: make(map[int]struct{}),
	}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter", " ":
			switch menuChoice(m.cursor) {
			case menuInit:
				initModel := NewProjectInitModel()
				return initModel, nil
			case menuGenerate:
				genModel := NewGenerateMenuModel()
				return genModel, nil
			case menuConfig:
				// TODO: Implement config TUI
				return m, tea.Quit
			case menuValidate:
				// TODO: Implement validate TUI
				return m, tea.Quit
			case menuExit:
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m MainMenuModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("🐉 Goryu CLI"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("A powerful web framework for Go"))
	b.WriteString("\n")

	// Menu options
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = accentStyle.Render("❯")
			choice = selectedStyle.Render(choice)
		} else {
			choice = unselectedStyle.Render(choice)
		}
		b.WriteString(fmt.Sprintf("%s %s\n", cursor, choice))
	}

	// Help text
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q/esc: quit"))

	return b.String()
}

// RunTUI starts the Bubble Tea TUI
func RunTUI() {
	p := tea.NewProgram(initialMainMenuModel())
	if _, err := p.Run(); err != nil {
		// If TUI fails, fall back to help
		showHelp()
		return
	}
}
