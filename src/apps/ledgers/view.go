package ledgers

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/vim"

	"local/bubbles/itempicker"

	tableBubble "github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (l Ledger) FilterValue() string {
	var result strings.Builder
	result.WriteString(l.Name)
	result.WriteString(strings.Join(l.Notes, ";"))
	return result.String()
}

func (l Ledger) Title() string {
	return l.Name
}

func (l Ledger) Description() string {
	return l.Name
}

type CreateView struct {
	table tableBubble.Model

	nameInput   textinput.Model
	typeInput   itempicker.Model
	noteInput   textarea.Model
	activeInput int

	styles styles.CreateViewStyles
}

func NewCreateView(app meta.App, colours styles.AppColours, width, height int) *CreateView {
	styles := styles.CreateViewStyles{
		Title: lipgloss.NewStyle().Background(colours.Background).Padding(0, 1),

		Table: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(colours.Foreground),
	}

	table := tableBubble.New()

	types := []itempicker.Item{
		Income,
		Expense,
		Asset,
		Liability,
		Equity,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	typeInput := itempicker.New(types)
	noteInput := textarea.New()

	result := &CreateView{
		table: table,

		nameInput:   nameInput,
		typeInput:   typeInput,
		noteInput:   noteInput,
		activeInput: 0,

		styles: styles,
	}

	result.updateTableDimensions(width, height)

	return result
}

func (cv *CreateView) Init() tea.Cmd {
	return nil
}

func (cv *CreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case vim.CompletedMotionMsg:
		switch message.Type {
		case vim.SWITCHFOCUS:
			switch cv.activeInput {
			case 0:
				cv.nameInput.Blur()
			case 2:
				cv.noteInput.Blur()
			}

			switch message.Data.(vim.Direction) {
			case vim.LEFT:
				cv.activeInput--
				if cv.activeInput < 0 {
					cv.activeInput += 3
				}

			case vim.RIGHT:
				cv.activeInput++
				cv.activeInput %= 3
			}

			switch cv.activeInput {
			case 0:
				cv.nameInput.Focus()
			case 2:
				cv.noteInput.Focus()
			}
		}

		return cv, nil

	case tea.WindowSizeMsg:
		cv.updateTableDimensions(message.Width, message.Height)

		return cv, nil
	}

	var cmd tea.Cmd
	switch cv.activeInput {
	case 0:
		cv.nameInput, cmd = cv.nameInput.Update(message)
	case 1:
		cv.typeInput, cmd = cv.typeInput.Update(message)
	case 2:
		cv.noteInput, cmd = cv.noteInput.Update(message)

	default:
		panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
	}

	return cv, cmd
}

func (cv *CreateView) View() string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("  %s", cv.styles.Title.Render("Create new Ledger")))
	result.WriteString("\n\n")

	cv.table.SetRows([]tableBubble.Row{{
		cv.nameInput.View(),
		cv.typeInput.View(),
		cv.noteInput.View(),
	}})

	result.WriteString(
		cv.styles.Table.Render(cv.table.View()),
	)

	return result.String()
}

func (cv *CreateView) Type() meta.ViewType {
	return meta.CreateViewType
}

func (cv *CreateView) MotionSet() *vim.MotionSet {
	var normalMotions vim.Trie
	normalMotions.Insert(vim.Motion{"ctrl+o"}, vim.CompletedMotionMsg{Type: vim.SWITCHVIEW, Data: vim.LISTVIEW})

	return &vim.MotionSet{Normal: normalMotions}
}

func (cv *CreateView) updateTableDimensions(width, height int) {
	tableWidth, tableHeight := viewDimensionsToTableDimensions(width, height)

	cv.table.SetWidth(tableWidth)
	cv.table.SetHeight(tableHeight)

	// -8 because each column has 1-wide padding on either side
	totalColumnWidth := width - 8

	typeInputWidth := 9 // Hardcoded, maximum length of a ledger type is 9 ('LIABILITY')
	nameInputWidth := min((totalColumnWidth-typeInputWidth)/2, 20)
	noteInputWidth := totalColumnWidth - typeInputWidth - nameInputWidth

	cv.table.SetColumns([]tableBubble.Column{
		{
			Title: "Name",
			Width: nameInputWidth,
		},
		{
			Title: "Type",
			Width: typeInputWidth,
		},
		{
			Title: "Notes",
			Width: noteInputWidth,
		},
	})

	cv.nameInput.Width = nameInputWidth
	cv.noteInput.SetWidth(noteInputWidth)
}

func viewDimensionsToTableDimensions(width, height int) (int, int) {
	// -2 for the table borders
	width = width - 2

	// -2 for the title
	// -2 for the table borders
	// -1 for the table header (I think? either way it looks good this way)
	height = height - 2 - 2 - 1

	return width, height
}
