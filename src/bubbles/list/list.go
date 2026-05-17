package list

import (
	"fmt"
	"log/slog"
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

type FuzzyFilterMsg struct {
	Query string
}

type Model struct {
	width, height int

	viewport   viewport.Model
	activeItem int

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

	case FuzzyFilterMsg:
		m.filterQuery = message.Query

		m.updateShownItems()

		return m, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (m Model) View() string {
	m.updateViewportContent()

	var result strings.Builder

	result.WriteString("Query: " + m.filterQuery)
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

func (m *Model) ActiveItem() *Item {
	if len(m.shownItems) == 0 {
		return nil
	}

	return &m.shownItems[m.activeItem]
}

// Scroll one line up/down
func (m *Model) Navigate(down bool) {
	if down {
		m.activeItem = min(m.activeItem+1, len(m.shownItems)-1)
	} else {
		m.activeItem = max(m.activeItem-1, 0)
	}

	m.scrollViewport()
}

// Scroll all the way to the top/bottom
func (m *Model) Jump(down bool) {
	if down {
		m.activeItem = len(m.shownItems) - 1
	} else {
		m.activeItem = 0
	}

	m.scrollViewport()
}

func (m *Model) scrollViewport() {
	// Use ScrollDown/ScrollUp to handle clipping
	if m.activeItem >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.ScrollDown(m.activeItem - m.viewport.YOffset - m.viewport.Height + 1)
	}

	if m.activeItem < m.viewport.YOffset {
		m.viewport.ScrollUp(m.viewport.YOffset - m.activeItem)
	}
}

// Applies the filter
func (m *Model) updateShownItems() {
	if m.filterQuery == "" {
		m.shownItems = m.items
		return
	}

	var filterValues []string
	for _, item := range m.items {
		filterValues = append(filterValues, item.FilterValue())
	}

	matches := fuzzy.Find(m.filterQuery, filterValues)

	m.shownItems = make([]Item, len(matches))

	for i, match := range matches {
		m.shownItems[i] = m.items[match.Index]
	}

	slog.Debug("updating shown items", "filterQuery", m.filterQuery, "n items", len(m.items), "n shown items", len(m.shownItems))

	m.activeItem = 0
	m.scrollViewport()
}

func (m *Model) updateViewportContent() {
	var result []string
	for i, item := range m.shownItems {
		result = append(result, item.Render(m.activeItem == i))
	}

	m.viewport.SetContent(strings.Join(result, "\n"))
}
