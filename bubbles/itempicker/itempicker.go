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
	items      []Item
	activeItem int
}

type ItemSelectedMsg struct {
	Item
}

func New(items []Item) Model {
	return Model{
		items: items,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlN:
			m.activeItem++

			if m.activeItem >= len(m.items) {
				m.activeItem = 0
			}

		case tea.KeyCtrlP:
			m.activeItem--

			if m.activeItem < 0 {
				m.activeItem = len(m.items) - 1
			}

		case tea.KeyEnter:
			return m, func() tea.Msg {
				return ItemSelectedMsg{
					Item: m.items[m.activeItem],
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if len(m.items) == 0 {
		return lipgloss.NewStyle().Italic(true).Render("No items")
	}

	return m.items[m.activeItem].String()
}
