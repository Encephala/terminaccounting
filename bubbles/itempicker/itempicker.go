package itempicker

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Item interface {
	fmt.Stringer
}

type Model struct {
	Items      []Item
	activeItem int
}

type ItemSelectedMsg struct {
	Item
}

func New() Model {
	return Model{
		Items: make([]Item, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlN:
			m.activeItem++

			if m.activeItem >= len(m.Items) {
				m.activeItem = 0
			}

		case tea.KeyCtrlP:
			m.activeItem--

			if m.activeItem < 0 {
				m.activeItem = len(m.Items) - 1
			}

		case tea.KeyEnter:
			return m, func() tea.Msg {
				return ItemSelectedMsg{
					Item: m.Items[m.activeItem],
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if len(m.Items) == 0 {
		return lipgloss.NewStyle().Italic(true).Render("No items")
	}

	return m.Items[m.activeItem].String()
}
