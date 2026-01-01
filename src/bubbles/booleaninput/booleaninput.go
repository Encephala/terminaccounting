package booleaninput

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	state bool
}

func New() Model {
	return Model{}
}

func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "enter":
			m.state = !m.state
		}

		return m, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (m Model) View() string {
	return renderBoolean(m.state)
}

func (m Model) Value() bool {
	return m.state
}

func (m *Model) SetValue(value bool) {
	m.state = value
}

func renderBoolean(reconciled bool) string {
	if reconciled {
		// Font Awesome checkbox because it's monospace, standard emoji character is too wide
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("")
	} else {
		return "□"
	}
}
