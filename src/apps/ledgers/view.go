package ledgers

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/vim"

	"local/bubbles/itempicker"

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
	}

	types := []itempicker.Item{
		Income,
		Expense,
		Asset,
		Liability,
		Equity,
	}

	result := &CreateView{
		idInput:     textinput.New(),
		nameInput:   textinput.New(),
		typeInput:   itempicker.New(types),
		noteInput:   textarea.New(),
		activeInput: 0,

		styles: styles,
	}

	return result
}

func (cv *CreateView) Init() tea.Cmd {
	return nil
}

func (cv *CreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

	result.WriteString(cv.idInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.nameInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.typeInput.View())
	result.WriteString(" | ")
	result.WriteString(cv.noteInput.View())

	return result.String()
}

func (cv *CreateView) Type() meta.ViewType {
	return meta.CreateViewType
}

func (cv *CreateView) MotionSet() *vim.MotionSet {
	var normal vim.Trie
	normal.Insert(vim.Motion{"ctrl+o"}, vim.CompletedMotionMsg{Type: vim.SWITCHVIEW, Data: vim.LISTVIEW})

	return &vim.MotionSet{Normal: normal}
}
