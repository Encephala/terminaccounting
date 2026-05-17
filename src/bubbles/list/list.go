package list

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

type Item interface {
	FilterValue() string

	Render(isActive bool) string
}

type Model struct {
	width, height int

	viewport  viewport.Model
	activeIdx int

	filterQuery string

	items      []Item
	shownItems []Item
}

func New(width, height int) Model {
	return Model{
		viewport: viewport.New(width, height),
	}
}

func (m Model) Update(message tea.Msg) (Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height

		m.viewport.Width = message.Width
		m.viewport.Height = message.Height

		return m, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (m Model) View() string {
	var result strings.Builder

	result.WriteString("Query: ")
	if m.filterQuery == "" {
		result.WriteString(lipgloss.NewStyle().Italic(true).Render("None"))
	} else {
		result.WriteString(m.filterQuery)
	}
	result.WriteString("\n\n")

	if len(m.shownItems) > m.viewport.Height {
		m.viewport.Width = m.width - 2

		scrollState := float64(m.viewport.YOffset) / float64(len(m.shownItems)-m.viewport.Height)

		result.WriteString(lipgloss.JoinHorizontal(lipgloss.Position(scrollState), m.viewport.View(), " ", "█"))
	} else {
		m.viewport.Width = m.width

		result.WriteString(m.viewport.View())
	}

	return result.String()
}

func (m *Model) SetItems(items []Item) {
	m.items = items

	m.updateShownItems()
}

func (m *Model) Items() []Item {
	return m.items
}

func (m Model) ActiveIndex() int {
	return m.activeIdx
}

func (m *Model) SetActiveIndex(index int) {
	m.activeIdx = index

	m.updateViewportContent()
}

func (m Model) ActiveItem() *Item {
	if len(m.shownItems) == 0 {
		return nil
	}

	return &m.shownItems[m.activeIdx]
}

func (m *Model) SetFilter(query string) {
	m.filterQuery = query

	m.updateShownItems()
}

// Scroll one line up/down
func (m *Model) Navigate(down bool) {
	if down {
		m.activeIdx = min(m.activeIdx+1, len(m.shownItems)-1)
	} else {
		m.activeIdx = max(m.activeIdx-1, 0)
	}

	m.updateViewportContent()
	m.scrollViewport()
}

// Scroll all the way to the top/bottom
func (m *Model) Jump(down bool) {
	if down {
		m.activeIdx = len(m.shownItems) - 1
	} else {
		m.activeIdx = 0
	}

	m.updateViewportContent()
	m.scrollViewport()
}

func (m *Model) scrollViewport() {
	// Use ScrollDown/ScrollUp to handle clipping
	if m.activeIdx >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.ScrollDown(m.activeIdx - m.viewport.YOffset - m.viewport.Height + 1)
	}

	if m.activeIdx < m.viewport.YOffset {
		m.viewport.ScrollUp(m.viewport.YOffset - m.activeIdx)
	}
}

// Applies the filter
func (m *Model) updateShownItems() {
	if m.filterQuery == "" {
		m.shownItems = m.items
	} else {
		var filterValues []string
		for _, item := range m.items {
			filterValues = append(filterValues, item.FilterValue())
		}

		matches := fuzzy.Find(m.filterQuery, filterValues)

		m.shownItems = make([]Item, len(matches))

		for i, match := range matches {
			m.shownItems[i] = m.items[match.Index]
		}
	}

	m.updateViewportContent()

	m.activeIdx = 0
	m.scrollViewport()
}

func (m *Model) updateViewportContent() {
	var result []string
	for i, item := range m.shownItems {
		result = append(result, item.Render(m.activeIdx == i))
	}

	m.viewport.SetContent(strings.Join(result, "\n"))
}
