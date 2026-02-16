package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var runInteractiveCommand = runSelf

type menuItem struct {
	Title       string
	Description string
	Args        []string
	Exit        bool
}

type menuModel struct {
	items    []menuItem
	cursor   int
	selected []string
	exit     bool
}

func newMenuModel() menuModel {
	return menuModel{
		items: []menuItem{
			{Title: "Clean (dry run)", Description: "Preview cache and temp cleanup", Args: []string{"clean", "--dry-run"}},
			{Title: "Analyze disk", Description: "Inspect disk usage by path", Args: []string{"analyze"}},
			{Title: "Purge projects (dry run)", Description: "Preview build artifact cleanup", Args: []string{"purge", "--dry-run"}},
			{Title: "Status", Description: "Show current system metrics", Args: []string{"status"}},
			{Title: "Exit", Description: "Close interactive mode", Exit: true},
		},
	}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.exit = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			item := m.items[m.cursor]
			if item.Exit {
				m.exit = true
			} else {
				m.selected = append([]string(nil), item.Args...)
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Render("Talpa Interactive")
	hint := lipgloss.NewStyle().Faint(true).Render("Use ↑/↓ (or j/k), Enter to run, q to quit")

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	defaultStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle().Faint(true)

	lines := []string{title, hint, ""}
	for i, item := range m.items {
		cursor := "  "
		style := defaultStyle
		if i == m.cursor {
			cursor = "> "
			style = selectedStyle
		}
		lines = append(lines, style.Render(cursor+item.Title))
		lines = append(lines, descStyle.Render("   "+item.Description))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func runInteractiveMenu() error {
	for {
		p := tea.NewProgram(newMenuModel())
		result, err := p.Run()
		if err != nil {
			return err
		}

		m, ok := result.(menuModel)
		if !ok || m.exit {
			return nil
		}
		if len(m.selected) == 0 {
			continue
		}

		fmt.Println()
		if err := runInteractiveCommand(m.selected...); err != nil {
			return fmt.Errorf("interactive command failed: %w", err)
		}
		fmt.Println()
	}
}
