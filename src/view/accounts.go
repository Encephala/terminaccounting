package view

import (
	"fmt"
	"local/bubbles/itempicker"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AccountsCreateView struct {
	nameInput  textinput.Model
	typeInput  itempicker.Model
	notesInput textarea.Model
	activeInput

	colours meta.AppColours
}

func NewAccountsCreateView(colours meta.AppColours) *AccountsCreateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	return &AccountsCreateView{
		nameInput:   nameInput,
		typeInput:   itempicker.New(accountTypes),
		notesInput:  notesInput,
		activeInput: NAMEINPUT,

		colours: colours,
	}
}

func (cv *AccountsCreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(cv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(cv.CommandSet())))

	return tea.Batch(cmds...)
}

func (cv *AccountsCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		name := cv.nameInput.Value()
		accountType := cv.typeInput.Value().(database.AccountType)
		notes := cv.notesInput.Value()

		newAccount := database.Account{
			Name:  name,
			Type:  accountType,
			Notes: meta.CompileNotes(notes),
		}

		id, err := newAccount.Insert()

		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		return cv, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		})

	case meta.SwitchFocusMsg:
		// If currently on a textinput, blur it
		// Shouldn't matter too much because we only send the update to the right input, but FWIW
		// Note from later me: might actually delete this as an implicit check that only the right input
		// gets the update message.
		switch cv.activeInput {
		case NAMEINPUT:
			cv.nameInput.Blur()
		case NOTEINPUT:
			cv.notesInput.Blur()
		}

		switch message.Direction {
		case meta.PREVIOUS:
			cv.activeInput--
			if cv.activeInput < 0 {
				cv.activeInput += 3
			}

		case meta.NEXT:
			cv.activeInput++
			cv.activeInput %= 3
		}

		// If now on a textinput, focus it
		switch cv.activeInput {
		case NAMEINPUT:
			cv.nameInput.Focus()
		case NOTEINPUT:
			cv.notesInput.Focus()
		}

		return cv, nil

	case meta.NavigateMsg:
		return cv, nil

	case tea.WindowSizeMsg:
		// TODO

		return cv, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch cv.activeInput {
		case NAMEINPUT:
			cv.nameInput, cmd = cv.nameInput.Update(message)
		case TYPEINPUT:
			cv.typeInput, cmd = cv.typeInput.Update(message)
		case NOTEINPUT:
			cv.notesInput, cmd = cv.notesInput.Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", cv.activeInput))
		}

		return cv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (cv *AccountsCreateView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(cv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render("Creating new Account"))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	const inputWidth = 26
	cv.nameInput.Width = inputWidth - 2
	cv.notesInput.SetWidth(inputWidth)

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
		style.Width(cv.typeInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(cv.typeInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(cv.notesInput.View()),
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

func (cv *AccountsCreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{Normal: normalMotions}
}

func (cv *AccountsCreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

type AccountsUpdateView struct {
	nameInput  textinput.Model
	typeInput  itempicker.Model
	notesInput textarea.Model
	activeInput

	modelId       int
	startingValue database.Account

	colours meta.AppColours
}

func NewAccountsUpdateView(modelId int, colours meta.AppColours) *AccountsUpdateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	return &AccountsUpdateView{
		nameInput:   nameInput,
		typeInput:   itempicker.New(accountTypes),
		notesInput:  notesInput,
		activeInput: NAMEINPUT,

		modelId: modelId,

		colours: colours,
	}
}

func (uv *AccountsUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(uv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(uv.CommandSet())))

	cmds = append(cmds, database.MakeLoadAccountsDetailCmd(uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *AccountsUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		account := message.Data.(database.Account)

		uv.startingValue = account

		uv.nameInput.SetValue(account.Name)
		uv.typeInput.SetValue(account.Type)
		uv.notesInput.SetValue(account.Notes.Collapse())

		return uv, nil

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput.SetValue(uv.startingValue.Name)
		case TYPEINPUT:
			uv.typeInput.SetValue(uv.startingValue.Type)
		case NOTEINPUT:
			uv.notesInput.SetValue(uv.startingValue.Notes.Collapse())
		}

		return uv, nil

	case meta.CommitMsg:
		account := database.Account{
			Id:    uv.modelId,
			Name:  uv.nameInput.Value(),
			Type:  uv.typeInput.Value().(database.AccountType),
			Notes: meta.CompileNotes(uv.notesInput.Value()),
		}

		account.Update()

		return uv, nil

	case meta.SwitchFocusMsg:
		// If currently on a textinput, blur it
		// Shouldn't matter too much because we only send the update to the right input, but FWIW
		// Note from later me: might actually delete this as an implicit check that only the right input
		// gets the update message.
		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput.Blur()
		case NOTEINPUT:
			uv.notesInput.Blur()
		}

		switch message.Direction {
		case meta.PREVIOUS:
			uv.activeInput--
			if uv.activeInput < 0 {
				uv.activeInput += 3
			}

		case meta.NEXT:
			uv.activeInput++
			uv.activeInput %= 3
		}

		// If now on a textinput, focus it
		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput.Focus()
		case NOTEINPUT:
			uv.notesInput.Focus()
		}

		return uv, nil

	case meta.NavigateMsg:
		return uv, nil

	case tea.WindowSizeMsg:
		// TODO

		return uv, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput, cmd = uv.nameInput.Update(message)
		case TYPEINPUT:
			uv.typeInput, cmd = uv.typeInput.Update(message)
		case NOTEINPUT:
			uv.notesInput, cmd = uv.notesInput.Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", uv.activeInput))
		}

		return uv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (uv *AccountsUpdateView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(uv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render("Creating new Account"))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	const inputWidth = 26
	uv.nameInput.Width = inputWidth - 2
	uv.notesInput.SetWidth(inputWidth)

	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Name"),
		" ",
		style.Render(uv.nameInput.View()),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Type"),
		" ",
		style.Width(uv.typeInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(uv.typeInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(uv.notesInput.View()),
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

func (uv *AccountsUpdateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (uv *AccountsUpdateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func (uv *AccountsUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}
