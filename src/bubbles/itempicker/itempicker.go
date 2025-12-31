package itempicker

import (
	"fmt"
	"slices"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Item interface {
	fmt.Stringer

	CompareId() int
}

type Model struct {
	Items      []Item
	activeItem int
}

func New(items []Item) Model {
	return Model{
		Items: items,
	}
}

func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.String() {
		case "ctrl+n", "j":
			m.activeItem++

			m.activeItem %= len(m.Items)

		case "ctrl+p", "k":
			m.activeItem--

			if m.activeItem < 0 {
				m.activeItem = len(m.Items) - 1
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if len(m.Items) == 0 {
		return lipgloss.NewStyle().Italic(true).Render("No items")
	}

	return fmt.Sprintf("> %s", m.Items[m.activeItem].String())
}

// Allows to manually retrieve the currently selected value.
func (m Model) Value() Item {
	if len(m.Items) == 0 {
		return nil
	}

	return m.Items[m.activeItem]
}

// Sets the currently selected item to the given value.
// Panics if the value isn't in the set of selectable items.
func (m *Model) SetValue(value Item) error {
	index := slices.IndexFunc(m.Items, func(item Item) bool {
		return item.CompareId() == value.CompareId()
	})

	if index == -1 {
		return fmt.Errorf("setting itempicker value to %#v but only valid choices are %#v", value, m.Items)
	}

	m.activeItem = index

	return nil
}

func (m Model) MaxViewLength() int {
	emptyWidth := len("No items")
	maxItemWidth := 0

	for _, item := range m.Items {
		maxItemWidth = max(maxItemWidth, len(item.String()))
	}

	return max(emptyWidth, maxItemWidth+2)
}
