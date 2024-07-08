package ledgers

import (
	"fmt"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type model struct {
	db *sqlx.DB

	activeView int
	models     []tea.Model
}

func New(db *sqlx.DB) meta.App {
	result := &model{
		db: db,

		activeView: 0,
		models:     []tea.Model{},
	}

	result.models = append(result.models, newListView(db, result))

	return result
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd

	for _, model := range m.models {
		cmds = append(cmds, model.Init())
	}

	return tea.Batch(cmds...)
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	for i, model := range m.models {
		m.models[i], cmd = model.Update(message)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.activeView < 0 || m.activeView >= len(m.models) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.activeView))
	}

	return m.models[m.activeView].View()
}

func (m *model) Name() string {
	return "Ledgers"
}

func (m *model) AccentColour() lipgloss.Color {
	return lipgloss.Color("#A1EEBDD0")
}
func (m *model) BackgroundColour() lipgloss.Color {
	return lipgloss.Color("#A1EEBD60")
}
func (m *model) HoverColour() lipgloss.Color {
	return lipgloss.Color("#A1EEBDFF")
}
