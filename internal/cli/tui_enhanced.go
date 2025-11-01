package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// it was wakuwaku and I wanted to implement it

type category struct {
	name  string
	emoji string
	items []menuItem
}

type menuItem struct {
	title       string
	description string
	action      string
	params      []string
}

type enhancedMenuModel struct {
	categories    []category
	currentCat    int
	currentItem   int
	showingItems  bool
	loading       bool
	spinner       spinner.Model
	notification  string
	notifyTimeout time.Time
	width         int
	height        int
}

func newEnhancedMenu() enhancedMenuModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(primaryColor)

	return enhancedMenuModel{
		categories: []category{
			{
				name:  "Project",
				emoji: "🚀",
				items: []menuItem{
					{
						title:       "Initialize New Project",
						description: "Create a new Goryu application with templates",
						action:      "init",
						params:      []string{},
					},
					{
						title:       "Validate Project",
						description: "Check project structure and configuration",
						action:      "validate", 
						params:      []string{},
					},
				},
			},
			{
				name:  "Code Generation",
				emoji: "🏗️",
				items: []menuItem{
					{
						title:       "Generate Handler",
						description: "Create HTTP handlers with various patterns",
						action:      "generate-handler",
						params:      []string{},
					},
					{
						title:       "Generate Middleware",
						description: "Create custom middleware with builder pattern",
						action:      "generate-middleware",
						params:      []string{},
					},
					{
						title:       "Generate Model",
						description: "Create data models with DB support",
						action:      "generate-model",
						params:      []string{},
					},
					{
						title:       "Generate Routes",
						description: "Create route configurations with builders",
						action:      "generate-route",
						params:      []string{},
					},
					{
						title:       "Generate Config",
						description: "Create configuration with builder pattern",
						action:      "generate-config",
						params:      []string{},
					},
				},
			},
			{
				name:  "Scaffolding",
				emoji: "🎯",
				items: []menuItem{
					{
						title:       "Scaffold REST API",
						description: "Generate complete CRUD API with tests",
						action:      "scaffold-api",
						params:      []string{},
					},
					{
						title:       "Scaffold Microservice",
						description: "Create microservice with HTTP/gRPC",
						action:      "scaffold-service",
						params:      []string{},
					},
					{
						title:       "Scaffold Admin Panel",
						description: "Generate admin interface",
						action:      "scaffold-admin",
						params:      []string{},
					},
				},
			},
			{
				name:  "Development",
				emoji: "🔧",
				items: []menuItem{
					{
						title:       "Start Dev Server",
						description: "Run development server with hot reload",
						action:      "dev",
						params:      []string{},
					},
					{
						title:       "Build Application",
						description: "Build for production",
						action:      "build",
						params:      []string{},
					},
					{
						title:       "Run Tests",
						description: "Execute test suite",
						action:      "test",
						params:      []string{},
					},
					{
						title:       "List Routes",
						description: "Display all registered routes",
						action:      "routes-list",
						params:      []string{},
					},
				},
			},
			{
				name:  "Middleware & Plugins",
				emoji: "🛡️",
				items: []menuItem{
					{
						title:       "List Middleware",
						description: "Show available middleware",
						action:      "middleware-list",
						params:      []string{},
					},
					{
						title:       "Middleware Info",
						description: "Get details about specific middleware",
						action:      "middleware-info",
						params:      []string{},
					},
					{
						title:       "Install Plugin",
						description: "Add community plugins",
						action:      "plugin-install",
						params:      []string{},
					},
				},
			},
			{
				name:  "Configuration",
				emoji: "⚙️",
				items: []menuItem{
					{
						title:       "Initialize Config",
						description: "Create configuration files",
						action:      "config-init",
						params:      []string{},
					},
					{
						title:       "Validate Config",
						description: "Check configuration validity",
						action:      "config-validate",
						params:      []string{},
					},
					{
						title:       "Migrate Config",
						description: "Convert between config formats",
						action:      "config-migrate",
						params:      []string{},
					},
				},
			},
		},
		spinner: s,
	}
}

func (m enhancedMenuModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m enhancedMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.showingItems {
				if m.currentItem > 0 {
					m.currentItem--
				}
			} else {
				if m.currentCat > 0 {
					m.currentCat--
				}
			}

		case "down", "j":
			if m.showingItems {
				if m.currentItem < len(m.categories[m.currentCat].items)-1 {
					m.currentItem++
				}
			} else {
				if m.currentCat < len(m.categories)-1 {
					m.currentCat++
				}
			}

		case "enter", " ":
			if !m.showingItems {
				m.showingItems = true
				m.currentItem = 0
			} else {
				// Execute the selected action
				item := m.categories[m.currentCat].items[m.currentItem]
				
				switch item.action {
				case "init":
					return NewProjectInitModel(), nil
				case "generate-handler", "generate-middleware", "generate-model":
					return NewGenerateMenuModel(), nil
				case "validate":
					if err := runValidate([]string{}); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
					return m, tea.Quit
				case "config-init":
					if err := runConfigInit([]string{}); err != nil {
						fmt.Printf("Error: %v\n", err)
					}
					return m, tea.Quit
				case "middleware-list":
					cmdMiddlewareList(&Context{})
					return m, tea.Quit
				default:
					fmt.Printf("❌ %s is not implemented yet\n", item.action)
					return m, tea.Quit
				}
			}

		case "esc", "backspace":
			if m.showingItems {
				m.showingItems = false
			} else {
				return m, tea.Quit
			}

		case "h", "?":
			m.notification = "Navigation: ↑↓/jk • Select: Enter/Space • Back: Esc • Quit: q"
			m.notifyTimeout = time.Now().Add(5 * time.Second)
		}

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case notificationMsg:
		if time.Now().After(m.notifyTimeout) {
			m.notification = ""
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return notificationMsg{}
		})
	}

	return m, nil
}

func (m enhancedMenuModel) View() string {
	if m.loading {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(
				lipgloss.Center,
				m.spinner.View(),
				loadingStyle.Render("Loading..."),
			),
		)
	}

	var content string

	header := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("🎯 Goryu CLI v"+VERSION),
		subtitleStyle.Render("A GOated Web Framework for Go"),
	)

	if !m.showingItems {
		var categoryList []string
		for i, cat := range m.categories {
			style := unselectedStyle
			if i == m.currentCat {
				style = selectedStyle
			}
			categoryList = append(categoryList,
				style.Render(fmt.Sprintf("%s %s", cat.emoji, cat.name)),
			)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			categoryList...,
		)
	} else {
		cat := m.categories[m.currentCat]
		
		categoryHeader := categoryStyle.Render(
			fmt.Sprintf("%s %s", cat.emoji, cat.name),
		)

		var itemList []string
		for i, item := range cat.items {
			style := itemStyle
			if i == m.currentItem {
				style = selectedItemStyle
			}

			itemContent := lipgloss.JoinVertical(
				lipgloss.Left,
				itemTitleStyle.Copy().Inherit(style).Render(item.title),
				itemDescStyle.Copy().Inherit(style).Render(item.description),
			)

			itemList = append(itemList, itemContent)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			categoryHeader,
			"",
			strings.Join(itemList, "\n"),
		)
	}

	help := helpStyle.Render("Navigate: ↑↓ • Select: Enter • Back: Esc • Help: h • Quit: q")

	if m.notification != "" {
		help = notificationStyle.Render(m.notification)
	}

	fullContent := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		help,
	)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		borderStyle.Render(fullContent),
	)
}

var (
	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(2, 4)

	categoryStyle = lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
		Padding(1, 2).
		MarginBottom(1)

	selectedItemStyle = itemStyle.Copy().
		Background(lipgloss.Color("#1F2937")).
		Foreground(textColor)

	itemTitleStyle = lipgloss.NewStyle().
		Bold(true)

	itemDescStyle = lipgloss.NewStyle().
		Foreground(mutedColor)

	loadingStyle = lipgloss.NewStyle().
		Foreground(mutedColor).
		MarginTop(1)

	notificationStyle = lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true)
)

type notificationMsg struct{}

func RunEnhancedTUI() error {
	p := tea.NewProgram(newEnhancedMenu(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}