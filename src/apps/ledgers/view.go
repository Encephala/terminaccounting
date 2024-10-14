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

	idInput     textinput.Model
	nameInput   textinput.Model
	typeInput   itempicker.Model
	noteInput   textarea.Model
	activeInput int

	styles styles.CreateViewStyles
}

func NewCreateView(app meta.App, colours styles.AppColours) *CreateView {
	styles := styles.CreateViewStyles{
		Title: lipgloss.NewStyle().Background(colours.Background).Padding(0, 1),

		Table: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(colours.Foreground),
	}

	tableColumns := []tableBubble.Column{
		{
			Title: "ID",
			Width: 6,
		},
		{
			Title: "Name",
			Width: 16,
		},
		{
			Title: "Type",
			Width: 10,
		},
		{
			Title: "Notes",
			Width: 20,
		},
	}
	table := tableBubble.New(
		tableBubble.WithColumns(tableColumns),
		tableBubble.WithHeight(6),
	)

	types := []itempicker.Item{
		Income,
		Expense,
		Asset,
		Liability,
		Equity,
	}

	idInput := textinput.New()
	idInput.Focus()
	nameInput := textinput.New()
	nameInput.Focus()
	typeInput := itempicker.New(types)
	noteInput := textarea.New()
	noteInput.Focus()

	result := &CreateView{
		table: table,

		idInput:     idInput,
		nameInput:   nameInput,
		typeInput:   typeInput,
		noteInput:   noteInput,
		activeInput: 0,

		styles: styles,
	}

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
			switch message.Data.(vim.Direction) {
			case vim.LEFT:
				cv.activeInput--
				if cv.activeInput < 0 {
					cv.activeInput += 4
				}

			case vim.RIGHT:
				cv.activeInput++
				cv.activeInput %= 4
			}
		}

		return cv, nil
	}

	var cmd tea.Cmd

	switch cv.activeInput {
	case 0:
		cv.idInput, cmd = cv.idInput.Update(message)
	case 1:
		cv.nameInput, cmd = cv.nameInput.Update(message)
	case 2:
		cv.typeInput, cmd = cv.typeInput.Update(message)
	case 3:
		cv.noteInput, cmd = cv.noteInput.Update(message)

	default:
		panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
	}

	return cv, cmd
}

func (cv *CreateView) View() string {
	var result strings.Builder

	result.WriteString("\n")
	result.WriteString(fmt.Sprintf(" %s", cv.styles.Title.Render("Create new Ledgers")))
	result.WriteString("\n\n")

	cv.table.SetRows([]tableBubble.Row{{
		cv.idInput.View(),
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
