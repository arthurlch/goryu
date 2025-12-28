package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type GenerateMenuModel struct {
	step          int
	cursor        int
	generatorType string
	name          string
	handlerType   string
	modelType     string
	dbTool        string
	path          string
	generating    bool
	done          bool
	err           error
}

const (
	genStepType = iota
	genStepName
	genStepOptions
	genStepConfirm
	genStepGenerating
	genStepDone
)

var generators = []string{
	"handler",
	"middleware",
	"model",
}

var handlerTypes = []string{"basic", "crud", "api"}
var modelTypes = []string{"basic", "db"}
var dbToolOptions = []string{"sqlc", "ent", "gorm"}

func NewGenerateMenuModel() GenerateMenuModel {
	return GenerateMenuModel{
		step: genStepType,
	}
}

func (m GenerateMenuModel) Init() tea.Cmd {
	return nil
}

func (m GenerateMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.step > genStepType {
				m.step--
				m.cursor = 0
				return m, nil
			}
			return newEnhancedMenu(), nil
		case "enter":
			return m.handleGenEnter()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			switch m.step {
			case genStepType:
				if m.cursor < len(generators)-1 {
					m.cursor++
				}
			case genStepOptions:
				if m.generatorType == "handler" && m.cursor < len(handlerTypes)-1 {
					m.cursor++
				} else if m.generatorType == "model" && m.modelType == "" && m.cursor < len(modelTypes)-1 {
					m.cursor++
				} else if m.generatorType == "model" && m.modelType == "db" && m.cursor < len(dbToolOptions)-1 {
					m.cursor++
				}
			}
		default:
			if m.step == genStepName {
				if msg.String() == "backspace" {
					if len(m.name) > 0 {
						m.name = m.name[:len(m.name)-1]
					}
				} else if len(msg.String()) == 1 {
					m.name += msg.String()
				}
			}
		}

	case generateCodeMsg:
		m.generating = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.done = true
		}
		return m, nil
	}

	return m, nil
}

func (m GenerateMenuModel) handleGenEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case genStepType:
		m.generatorType = generators[m.cursor]
		m.step = genStepName
		m.cursor = 0
		// Set default paths
		switch m.generatorType {
		case "handler":
			m.path = "internal/handlers"
		case "middleware":
			m.path = "internal/middleware"
		case "model":
			m.path = "internal/models"
		}
	case genStepName:
		if strings.TrimSpace(m.name) != "" {
			if m.generatorType == "middleware" {
				m.step = genStepConfirm
			} else {
				m.step = genStepOptions
				m.cursor = 0
			}
		}
	case genStepOptions:
		if m.generatorType == "handler" { // lets ignore warning for that block
			m.handlerType = handlerTypes[m.cursor]
			m.step = genStepConfirm
		} else if m.generatorType == "model" {
			if m.modelType == "" {
				m.modelType = modelTypes[m.cursor]
				if m.modelType == "db" {
					m.cursor = 0
				} else {
					m.step = genStepConfirm
				}
			} else if m.modelType == "db" {
				m.dbTool = dbToolOptions[m.cursor]
				m.step = genStepConfirm
			}
		}
	case genStepConfirm:
		m.step = genStepGenerating
		m.generating = true
		return m, m.generateCode()
	case genStepDone:
		return NewGenerateMenuModel(), nil
	}
	return m, nil
}

type generateCodeMsg struct {
	err error
}

func (m GenerateMenuModel) generateCode() tea.Cmd {
	return func() tea.Msg {
		var err error

		switch m.generatorType {
		case "handler":
			// Create args for handler generation
			args := []string{m.name}
			if m.handlerType != "" {
				args = append(args, "--type="+m.handlerType)
			}
			if m.path != "internal/handlers" {
				args = append(args, "--path="+m.path)
			}
			err = runGenerateHandler(args)
		case "middleware":
			args := []string{m.name}
			if m.path != "internal/middleware" {
				args = append(args, "--path="+m.path)
			}
			err = runGenerateMiddleware(args)
		case "model":
			args := []string{m.name}
			if m.modelType != "" {
				args = append(args, "--type="+m.modelType)
			}
			if m.dbTool != "" {
				args = append(args, "--db-tool="+m.dbTool)
			}
			if m.path != "internal/models" {
				args = append(args, "--path="+m.path)
			}
			err = runGenerateModel(args)
		}

		return generateCodeMsg{err: err}
	}
}

func (m GenerateMenuModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("⚡ Generate Code"))
	b.WriteString("\n\n")

	switch m.step {
	case genStepType:
		b.WriteString("What would you like to generate?")
		b.WriteString("\n\n")

		for i, gen := range generators {
			cursor := " "
			description := getGeneratorDescription(gen)

			if m.cursor == i {
				cursor = accentStyle.Render("❯")
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					selectedStyle.Render(gen),
					selectedStyle.Render(description)))
			} else {
				b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
					unselectedStyle.Render(gen),
					unselectedStyle.Render(description)))
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	case genStepName:
		b.WriteString(fmt.Sprintf("Enter %s name:", m.generatorType))
		b.WriteString("\n")

		nameStyle := lipgloss.NewStyle().
			Foreground(textColor).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1).
			Width(30)

		name := m.name
		if name == "" {
			name = "my" + strings.Title(m.generatorType)
		}
		b.WriteString(nameStyle.Render(name))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Type name • enter: next • esc: back"))

	case genStepOptions:
		if m.generatorType == "handler" { // for that block too
			b.WriteString("Choose handler type:")
			b.WriteString("\n\n")

			for i, hType := range handlerTypes {
				cursor := " "
				description := getHandlerTypeDescription(hType)

				if m.cursor == i {
					cursor = accentStyle.Render("❯")
					b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
						selectedStyle.Render(hType),
						selectedStyle.Render(description)))
				} else {
					b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
						unselectedStyle.Render(hType),
						unselectedStyle.Render(description)))
				}
			}
		} else if m.generatorType == "model" {
			if m.modelType == "" {
				b.WriteString("Choose model type:")
				b.WriteString("\n\n")

				for i, mType := range modelTypes {
					cursor := " "
					description := getModelTypeDescription(mType)

					if m.cursor == i {
						cursor = accentStyle.Render("❯")
						b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
							selectedStyle.Render(mType),
							selectedStyle.Render(description)))
					} else {
						b.WriteString(fmt.Sprintf("%s %s %s\n", cursor,
							unselectedStyle.Render(mType),
							unselectedStyle.Render(description)))
					}
				}
			} else if m.modelType == "db" {
				b.WriteString("Choose database tool:")
				b.WriteString("\n\n")

				for i, tool := range dbToolOptions {
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
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • esc: back"))

	case genStepConfirm:
		b.WriteString("Confirm generation settings:")
		b.WriteString("\n\n")

		b.WriteString(fmt.Sprintf("Type:     %s\n", accentStyle.Render(m.generatorType)))
		b.WriteString(fmt.Sprintf("Name:     %s\n", accentStyle.Render(m.name)))
		if m.handlerType != "" {
			b.WriteString(fmt.Sprintf("Handler:  %s\n", accentStyle.Render(m.handlerType)))
		}
		if m.modelType != "" {
			b.WriteString(fmt.Sprintf("Model:    %s\n", accentStyle.Render(m.modelType)))
		}
		if m.dbTool != "" {
			b.WriteString(fmt.Sprintf("DB Tool:  %s\n", accentStyle.Render(m.dbTool)))
		}
		b.WriteString(fmt.Sprintf("Path:     %s\n", accentStyle.Render(m.path)))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter: generate • esc: back"))

	case genStepGenerating:
		b.WriteString(fmt.Sprintf("Generating %s '%s'...\n\n", m.generatorType, m.name))
		b.WriteString("⏳ Creating files...")

	case genStepDone:
		if m.err != nil {
			b.WriteString(errorStyle.Render("❌ Error generating code:"))
			b.WriteString("\n")
			b.WriteString(errorStyle.Render(m.err.Error()))
		} else {
			b.WriteString(successStyle.Render("✅ Code generated successfully!"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("Created %s: %s/%s.go\n", m.generatorType, m.path, strings.ToLower(m.name)))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter: generate another • esc: back to main menu"))
	}

	return b.String()
}

func getGeneratorDescription(gen string) string {
	switch gen {
	case "handler":
		return "- HTTP request handlers"
	case "middleware":
		return "- Request/response middleware"
	case "model":
		return "- Data models & repositories"
	default:
		return ""
	}
}

func getHandlerTypeDescription(hType string) string {
	switch hType {
	case "basic":
		return "- Simple endpoint handler"
	case "crud":
		return "- Full CRUD operations"
	case "api":
		return "- Structured API handler"
	default:
		return ""
	}
}

func getModelTypeDescription(mType string) string {
	switch mType {
	case "basic":
		return "- Simple data struct"
	case "db":
		return "- Database-integrated model"
	default:
		return ""
	}
}
