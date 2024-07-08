package entries

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
		changedEntries, err := setupSchemaEntries(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entries` TABLE: %v", err)
			return m, func() tea.Msg { return meta.ErrorMsg{Error: message} }
		}

		changedEntryRows, err := setupSchemaEntryRows(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entryrows` TABLE: %v", err)
			return m, func() tea.Msg { return meta.ErrorMsg{Error: message} }
		}

		if changedEntries+changedEntryRows != 0 {
			return m, func() tea.Msg {
				slog.Info("Set up `Entries` schema")
				return nil
			}
		}

		return m, nil
	}

	return m, nil
}

func (m *model) View() string {
	return "TODO entries"
}

func (m *model) Name() string {
	return "Entries"
}

func (m *model) AccentColour() lipgloss.Color {
	return lipgloss.Color("#F0F1B2D0")
}
func (m *model) BackgroundColour() lipgloss.Color {
	return lipgloss.Color("#F0F1B280")
}
func (m *model) HoverColour() lipgloss.Color {
	return lipgloss.Color("#EBECABFF")
}
