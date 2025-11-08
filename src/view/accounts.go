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
	return nil
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

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Account %q", cv.nameInput.Value(),
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)

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
	rightStyle := style.Margin(0, 0, 0, 1)

	const inputWidth = 26
	cv.nameInput.Width = inputWidth - 2
	cv.notesInput.SetWidth(inputWidth)

	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Name"),
		rightStyle.Render(cv.nameInput.View()),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Type"),
		rightStyle.Width(cv.typeInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(cv.typeInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Notes"),
		rightStyle.Render(cv.notesInput.View()),
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
	return database.MakeLoadAccountsDetailCmd(uv.modelId)
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

		err := account.Update()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Account %q", uv.nameInput.Value(),
		)})

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

	result.WriteString(titleStyle.Render(fmt.Sprintf("Updating Account %q", uv.startingValue.Name)))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	rightStyle := style.Margin(0, 0, 0, 1)

	const inputWidth = 26
	uv.nameInput.Width = inputWidth - 2
	uv.notesInput.SetWidth(inputWidth)

	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Name"),
		rightStyle.Render(uv.nameInput.View()),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Type"),
		rightStyle.Width(uv.typeInput.MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(uv.typeInput.View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Notes"),
		rightStyle.Render(uv.notesInput.View()),
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

type AccountsDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Account

	colours meta.AppColours
}

func NewAccountsDeleteView(modelId int, colours meta.AppColours) *AccountsDeleteView {
	return &AccountsDeleteView{
		modelId: modelId,

		colours: colours,
	}
}

func (dv *AccountsDeleteView) Init() tea.Cmd {
	return database.MakeLoadAccountsDetailCmd(dv.modelId)
}

func (dv *AccountsDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(database.Account)

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteAccount(dv.modelId)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Account %q", dv.model.Name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *AccountsDeleteView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Delete Account: %s", dv.model.Name)))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	rightStyle := style.Margin(0, 0, 0, 1)

	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Name"),
		rightStyle.Render(dv.model.Name),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Type"),
		rightStyle.Render(dv.model.Type.String()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Render("Notes"),
		rightStyle.AlignHorizontal(lipgloss.Left).Render(dv.model.Notes.Collapse()),
	)

	var confirmRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			nameRow,
			typeRow,
			notesRow,
			"",
			confirmRow,
		),
	))

	return result.String()
}

func (dv *AccountsDeleteView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *AccountsDeleteView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func (dv *AccountsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
