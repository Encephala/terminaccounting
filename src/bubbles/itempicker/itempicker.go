package itempicker

import (
	"fmt"
	"slices"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

type Item interface {
	fmt.Stringer

	CompareId() int
}

type FuzzySelectMsg struct {
	Query string
}

type Model struct {
	MaxWidth int
	Colour   lipgloss.Color

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
		if len(m.Items) == 0 {
			return m, nil
		}

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

		return m, nil

	case FuzzySelectMsg:
		if message.Query == "" {
			return m, nil
		}

		var stringReprs []string
		for _, item := range m.Items {
			stringReprs = append(stringReprs, item.String())
		}

		matches := fuzzy.Find(message.Query, stringReprs)

		if len(matches) == 0 {
			return m, meta.MessageCmd(fmt.Errorf("%q not found in items", message.Query))
		}

		m.activeItem = matches[0].Index

		return m, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (m Model) View() string {
	style := lipgloss.NewStyle().Foreground(m.Colour)

	var result string
	if len(m.Items) == 0 {
		result = style.Italic(true).Render("No items")
	} else {
		result = style.Render(fmt.Sprintf("> %s", m.Items[m.activeItem].String()))
	}

	if m.MaxWidth == 0 {
		return result
	}

	return ansi.Truncate(result, m.MaxWidth, "…")
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
	if len(m.Items) == 0 {
		return len("No items")
	}

	result := 0
	for _, item := range m.Items {
		// +2 for prompt "> "
		result = max(result, len(item.String())+2)
	}

	return result
}
