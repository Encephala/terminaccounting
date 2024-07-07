package journals

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
	return "TODO journals"
}

func (m *model) Name() string {
	return "Journals"
}

func (m *model) AccentColour() lipgloss.Color {
	return lipgloss.Color("#F6D6D6D0")
}
