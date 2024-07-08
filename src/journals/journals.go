package journals

import (
	"fmt"
	"log/slog"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	viewWidth, viewHeight int
}

func New() meta.App {
	return &model{}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

	case meta.SetupSchemaMsg:
		changed, err := setupSchema(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `journals` TABLE: %v", err)
			return m, func() tea.Msg { return meta.ErrorMsg{Error: message} }
		}

		if changed != 0 {
			return m, func() tea.Msg {
				slog.Info("Set up `Journals` schema")
				return nil
			}
		}

		return m, nil
	}

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
func (m *model) BackgroundColour() lipgloss.Color {
	return lipgloss.Color("#F6D6D680")
}
func (m *model) HoverColour() lipgloss.Color {
	return lipgloss.Color("#F6D6D6FF")
}
