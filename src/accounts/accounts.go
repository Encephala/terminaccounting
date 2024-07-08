package accounts

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct{}

func New() meta.App {
	return &model{}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *model) View() string {
	return "TODO accounts"
}

func (m *model) Name() string {
	return "Accounts"
}

func (m *model) AccentColour() lipgloss.Color {
	return lipgloss.Color("#7BD4EAD0")
}
func (m *model) BackgroundColour() lipgloss.Color {
	return lipgloss.Color("#7BD4EA50")
}
func (m *model) HoverColour() lipgloss.Color {
	return lipgloss.Color("#7BD4EAFF")
}
