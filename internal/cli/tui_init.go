package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ProjectInitModel struct {
	step        int
	projectName string
	template    string
	dbTool      string
	cursor      int
	templates   []string
	dbTools     []string
	creating    bool
	done        bool
	err         error
}

const (
	stepProjectName = iota
	stepTemplate
	stepDBTool
	stepConfirm
	stepCreating
	stepDone
)

func NewProjectInitModel() ProjectInitModel {
	return ProjectInitModel{
		step:      stepProjectName,
		templates: []string{"basic", "api", "web"},
		dbTools:   []string{"sqlc", "ent", "gorm"},
	}
}

func (m ProjectInitModel) Init() tea.Cmd {
	return nil
}

func (m ProjectInitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step > stepProjectName {
				m.step--
				m.cursor = 0
				return m, nil
			}
			return initialMainMenuModel(), nil
		case "enter":
			return m.handleEnter()
		case "up", "k":
			if m.step == stepTemplate || m.step == stepDBTool {
				if m.cursor > 0 {
					m.cursor--
				}
			}
		case "down", "j":
			if m.step == stepTemplate {
				if m.cursor < len(m.templates)-1 {
					m.cursor++
				}
			} else if m.step == stepDBTool {
				if m.cursor < len(m.dbTools)-1 {
					m.cursor++
				}
			}
		default:
			if m.step == stepProjectName {
				if msg.String() == "backspace" {
					if len(m.projectName) > 0 {
						m.projectName = m.projectName[:len(m.projectName)-1]
					}
				} else if len(msg.String()) == 1 {
					m.projectName += msg.String()
				}
			}
		}

	case createProjectMsg:
		m.creating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.done = true
		}
		return m, nil
	}

	return m, nil
}

func (m ProjectInitModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepProjectName:
		if strings.TrimSpace(m.projectName) != "" {
			m.step = stepTemplate
			m.cursor = 0
		}
	case stepTemplate:
		m.template = m.templates[m.cursor]
		if m.template == "db" {
			m.step = stepDBTool
			m.cursor = 0
		} else {
			m.step = stepConfirm
		}
	case stepDBTool:
		m.dbTool = m.dbTools[m.cursor]
		m.step = stepConfirm
	case stepConfirm:
		m.step = stepCreating
		m.creating = true
		return m, m.createProject()
	case stepDone:
		return initialMainMenuModel(), nil
	}
	return m, nil
}

type createProjectMsg struct {
	err error
}

func (m ProjectInitModel) createProject() tea.Cmd {
	return func() tea.Msg {
		var err error
		fa := NewFlexibleArchitecture()
		if loadErr := fa.LoadArchitectures(); loadErr != nil {
			return createProjectMsg{err: loadErr}
		}
		err = fa.GenerateProject(m.projectName, m.template, nil)
		return createProjectMsg{err: err}
	}
}

func (m ProjectInitModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("🚀 Initialize New Project"))
	b.WriteString("\n\n")

	switch m.step {
	case stepProjectName:
		b.WriteString("Project name:")
		b.WriteString("\n")

		nameStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1).
			Width(40)

		projectName := m.projectName
		if projectName == "" {
			projectName = "my-goryu-app"
		}
		b.WriteString(nameStyle.Render(projectName))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Type project name • enter: next • esc: back"))

	case stepTemplate:
		b.WriteString("Choose project template:")
		b.WriteString("\n\n")

		for i, template := range m.templates {
			cursor := " "
			description := getTemplateDescription(template)

			if m.cursor == i {
				cursor = accentStyle.Render("❯")
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					selectedStyle.Render(template),
					selectedStyle.Render(description)))
			} else {
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					unselectedStyle.Render(template),
					unselectedStyle.Render(description)))
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	case stepDBTool:
		b.WriteString("Choose database tool:")
		b.WriteString("\n\n")

		for i, tool := range m.dbTools {
			cursor := " "
			description := getDBToolDescription(tool)

			if m.cursor == i {
				cursor = accentStyle.Render("❯")
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					selectedStyle.Render(tool),
					selectedStyle.Render(description)))
			} else {
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					unselectedStyle.Render(tool),
					unselectedStyle.Render(description)))
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	case stepConfirm:
		b.WriteString("Confirm project settings:")
		b.WriteString("\n\n")

		b.WriteString(fmt.Sprintf("Project name: %s\n", accentStyle.Render(m.projectName)))
		b.WriteString(fmt.Sprintf("Template:     %s\n", accentStyle.Render(m.template)))
		if m.template == "db" {
			b.WriteString(fmt.Sprintf("Database:     %s\n", accentStyle.Render(m.dbTool)))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter: create project • esc: back"))

	case stepCreating:
		b.WriteString(fmt.Sprintf("Creating project '%s'...\n\n", m.projectName))
		b.WriteString("⏳ Setting up project structure...")

	case stepDone:
		if m.err != nil {
			b.WriteString(errorStyle.Render("❌ Error creating project:"))
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(m.err.Error()))
		} else {
			b.WriteString(successStyle.Render("✅ Project created successfully!"))
			b.WriteString("\n\n")
			b.WriteString("Next steps:\n")
			b.WriteString(fmt.Sprintf("  cd %s\n", m.projectName))
			b.WriteString("  go mod tidy\n")
			if m.template == "db" {
				switch m.dbTool {
				case "sqlc":
					b.WriteString("  make install-tools\n")
					b.WriteString("  make sqlc-generate\n")
				case "ent":
					b.WriteString("  go generate ./ent\n")
				case "gorm":
					b.WriteString("  # Configure database and auto-migrate\n")
				}
			}
			b.WriteString("  go run cmd/server/main.go\n")
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter: back to main menu"))
	}

	return b.String()
}

func getTemplateDescription(template string) string {
	switch template {
	case "basic":
		return "- Simple web application"
	case "api":
		return "- REST API with enhanced features"
	case "web":
		return "- Web app with static file serving"
	default:
		return ""
	}
}

func getDBToolDescription(tool string) string {
	switch tool {
	case "sqlc":
		return "- Type-safe SQL with code generation"
	case "ent":
		return "- Entity framework with relationships"
	case "gorm":
		return "- Traditional ORM with auto-migration"
	default:
		return ""
	}
}
