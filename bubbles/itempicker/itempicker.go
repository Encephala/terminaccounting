package itempicker

import (
	"fmt"
	"strings"

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

// A message that's sent to the bubbletea app to inform that the user selected an item
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
		switch message.String() {
		case "ctrl+n", "j":
			m.activeItem++

			if m.activeItem >= len(m.items) {
				m.activeItem = 0
			}

		case "ctrl+p", "k":
			m.activeItem--

			if m.activeItem < 0 {
				m.activeItem = len(m.items) - 1
			}

		case "enter":
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

	var result strings.Builder
	result.WriteString("> ")
	result.WriteString(m.items[m.activeItem].String())

	return result.String()
}

// Allows to manually retrieve the currently selected value.
func (m Model) Value() Item {
	return m.items[m.activeItem]
}

// Sets the currently selected item to the given value.
// Panics if the value isn't in the set of selectable items.
func (m *Model) SetValue(value Item) {
	var index int
	found := false

	for i, item := range m.items {
		if item == value {
			index = i
			found = true

			break
		}
	}

	if !found {
		panic(fmt.Sprintf("Setting itempicker value to %v but only valid choices are %v", value, m.items))
	}

	m.activeItem = index
}

func (m *Model) SetItems(items []Item) {
	m.items = items
}

func (m Model) MaxViewLength() int {
	emptyWidth := len("No items")
	maxItemWidth := 0

	for _, item := range m.items {
		maxItemWidth = max(maxItemWidth, len(item.String()))
	}

	return max(emptyWidth, maxItemWidth+2)
}
