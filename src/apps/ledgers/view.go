package ledgers

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/vim"

	"local/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/cursor"
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

	types := []itempicker.Item{
		INCOME,
		EXPENSE,
		ASSET,
		LIABILITY,
		EQUITY,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	typeInput := itempicker.New(types)
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &CreateView{
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
			// If currently on a textinput, blur it
			// Shouldn't matter too much because we only send the update to the right input, but FWIW
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

			// If now on a textinput, focus it
			switch cv.activeInput {
			case 0:
				cv.nameInput.Focus()
			case 2:
				cv.noteInput.Focus()
			}

		default:
			panic(fmt.Sprintf("unexpected vim.completedMotionType: %#v", message.Type))
		}

		return cv, nil

	case tea.WindowSizeMsg:
		// TODO

		return cv, nil

	case tea.KeyMsg:
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

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (cv *CreateView) View() string {
	var result strings.Builder

	result.WriteString(fmt.Sprintf("  %s", cv.styles.Title.Render("Create new Ledger")))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	const inputWidth = 26
	cv.nameInput.Width = inputWidth - 2 // -2 because of the prompt
	cv.noteInput.SetWidth(inputWidth)

	// TODO: Render active input with a different colour
	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Name"),
		" ",
		style.Render(cv.nameInput.View()),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Type"),
		" ",
		style.Render(cv.typeInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(cv.noteInput.View()),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			nameRow,
			typeRow,
			notesRow,
		),
	))

	return result.String()
}

func (cv *CreateView) Type() meta.ViewType {
	return meta.CreateViewType
}

func (cv *CreateView) MotionSet() *vim.MotionSet {
	var normalMotions vim.Trie[vim.CompletedMotionMsg]

	normalMotions.Insert(vim.Motion{"ctrl+o"}, vim.CompletedMotionMsg{Type: vim.SWITCHVIEW, Data: vim.LISTVIEW})

	return &vim.MotionSet{Normal: normalMotions}
}

func (cv *CreateView) CommandSet() *vim.CommandSet {
	var commands vim.Trie[vim.CompletedCommandMsg]

	commands.Insert(vim.Command{"w"}, vim.CompletedCommandMsg{Type: vim.WRITE})

	return &vim.CommandSet{Commands: commands}
}
