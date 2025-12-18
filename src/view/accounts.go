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

type accountsActiveInput int

func (input *accountsActiveInput) previous() {
	*input--

	if *input < 0 {
		*input += accountsActiveInput(NUMACCOUNTSINPUTS)
	}
}

func (input *accountsActiveInput) next() {
	*input++

	*input %= accountsActiveInput(NUMACCOUNTSINPUTS)
}

const (
	ACCOUNTSNAMEINPUT accountsActiveInput = iota
	ACCOUNTSTYPEINPUT
	ACCOUNTSBANKNUMBERSINPUT
	ACCOUNTSNOTESINPUT
)

const NUMACCOUNTSINPUTS int = 4

type AccountsCreateView struct {
	nameInput        textinput.Model
	typeInput        itempicker.Model
	bankNumbersInput textarea.Model
	notesInput       textarea.Model
	activeInput      accountsActiveInput

	colours meta.AppColours
}

func NewAccountsCreateView() *AccountsCreateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	return &AccountsCreateView{
		nameInput:        nameInput,
		typeInput:        itempicker.New(accountTypes),
		notesInput:       notesInput,
		bankNumbersInput: bankNumbersInput,
		activeInput:      ACCOUNTSNAMEINPUT,

		colours: meta.ACCOUNTSCOLOURS,
	}
}

func (cv *AccountsCreateView) Init() tea.Cmd {
	return nil
}

func (cv *AccountsCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		accountType := cv.typeInput.Value().(database.AccountType)

		newAccount := database.Account{
			Name:        cv.nameInput.Value(),
			Type:        accountType,
			BankNumbers: meta.CompileNotes(cv.bankNumbersInput.Value()),
			Notes:       meta.CompileNotes(cv.notesInput.Value()),
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
		case ACCOUNTSNAMEINPUT:
			cv.nameInput.Blur()
		case ACCOUNTSBANKNUMBERSINPUT:
			cv.bankNumbersInput.Blur()
		case ACCOUNTSNOTESINPUT:
			cv.notesInput.Blur()
		}

		switch message.Direction {
		case meta.PREVIOUS:
			cv.activeInput.previous()

		case meta.NEXT:
			cv.activeInput.next()
		}

		// If now on a textinput, focus it
		switch cv.activeInput {
		case ACCOUNTSNAMEINPUT:
			cv.nameInput.Focus()
		case ACCOUNTSBANKNUMBERSINPUT:
			cv.bankNumbersInput.Focus()
		case ACCOUNTSNOTESINPUT:
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
		case ACCOUNTSNAMEINPUT:
			cv.nameInput, cmd = cv.nameInput.Update(message)
		case ACCOUNTSTYPEINPUT:
			cv.typeInput, cmd = cv.typeInput.Update(message)
		case ACCOUNTSBANKNUMBERSINPUT:
			cv.bankNumbersInput, cmd = cv.bankNumbersInput.Update(message)
		case ACCOUNTSNOTESINPUT:
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

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(cv.colours.Foreground)

	const inputWidth = 26
	cv.nameInput.Width = inputWidth - 2
	cv.bankNumbersInput.SetWidth(inputWidth)
	cv.notesInput.SetWidth(inputWidth)

	notesFocusStyle := lipgloss.NewStyle().Foreground(cv.colours.Foreground)
	cv.bankNumbersInput.FocusedStyle.Prompt = notesFocusStyle
	cv.bankNumbersInput.FocusedStyle.Text = notesFocusStyle
	cv.bankNumbersInput.FocusedStyle.CursorLine = notesFocusStyle
	cv.bankNumbersInput.FocusedStyle.LineNumber = notesFocusStyle
	cv.notesInput.FocusedStyle.Prompt = notesFocusStyle
	cv.notesInput.FocusedStyle.Text = notesFocusStyle
	cv.notesInput.FocusedStyle.CursorLine = notesFocusStyle
	cv.notesInput.FocusedStyle.LineNumber = notesFocusStyle

	nameStyle := sectionStyle
	typeStyle := sectionStyle

	switch cv.activeInput {
	case ACCOUNTSNAMEINPUT:
		nameStyle = highlightStyle
	case ACCOUNTSTYPEINPUT:
		typeStyle = highlightStyle
	// textareas have FocusedStyle set, don't manually render with highlightStyle
	case ACCOUNTSBANKNUMBERSINPUT:
	case ACCOUNTSNOTESINPUT:
	default:
		panic(fmt.Sprintf("unexpected view.accountsActiveInput: %#v", cv.activeInput))
	}

	// +2 for padding
	maxNameColWidth := len("Bank numbers") + 2

	nameRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Name"),
		" ",
		nameStyle.Render(cv.nameInput.View()),
	)

	typeRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Type"),
		" ",
		typeStyle.Render(cv.typeInput.View()),
	)

	bankNumbersRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Bank numbers"),
		" ",
		sectionStyle.Render(cv.bankNumbersInput.View()),
	)

	notesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Notes"),
		" ",
		sectionStyle.Render(cv.notesInput.View()),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			nameRow,
			typeRow,
			bankNumbersRow,
			notesRow,
		),
	))

	return result.String()
}

func (cv *AccountsCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *AccountsCreateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func (cv *AccountsCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *AccountsCreateView) Reload() View {
	return NewAccountsCreateView()
}

type AccountsUpdateView struct {
	nameInput        textinput.Model
	typeInput        itempicker.Model
	bankNumbersInput textarea.Model
	notesInput       textarea.Model
	activeInput      accountsActiveInput

	modelId       int
	startingValue database.Account

	colours meta.AppColours
}

func NewAccountsUpdateView(modelId int) *AccountsUpdateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	return &AccountsUpdateView{
		nameInput:        nameInput,
		typeInput:        itempicker.New(accountTypes),
		bankNumbersInput: bankNumbersInput,
		notesInput:       notesInput,
		activeInput:      ACCOUNTSNAMEINPUT,

		modelId: modelId,

		colours: meta.ACCOUNTSCOLOURS,
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
		err := uv.typeInput.SetValue(account.Type)
		uv.bankNumbersInput.SetValue(account.BankNumbers.Collapse())
		uv.notesInput.SetValue(account.Notes.Collapse())

		return uv, meta.MessageCmd(err)

	case meta.ResetInputFieldMsg:
		var err error
		switch uv.activeInput {
		case ACCOUNTSNAMEINPUT:
			uv.nameInput.SetValue(uv.startingValue.Name)
		case ACCOUNTSTYPEINPUT:
			err = uv.typeInput.SetValue(uv.startingValue.Type)
		case ACCOUNTSBANKNUMBERSINPUT:
			uv.bankNumbersInput.SetValue(uv.startingValue.BankNumbers.Collapse())
		case ACCOUNTSNOTESINPUT:
			uv.notesInput.SetValue(uv.startingValue.Notes.Collapse())
		}

		return uv, meta.MessageCmd(err)

	case meta.CommitMsg:
		typeInput := uv.typeInput.Value()

		account := database.Account{
			Id:          uv.modelId,
			Name:        uv.nameInput.Value(),
			Type:        typeInput.(database.AccountType),
			BankNumbers: meta.CompileNotes(uv.bankNumbersInput.Value()),
			Notes:       meta.CompileNotes(uv.notesInput.Value()),
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
		case ACCOUNTSNAMEINPUT:
			uv.nameInput.Blur()
		case ACCOUNTSBANKNUMBERSINPUT:
			uv.bankNumbersInput.Blur()
		case ACCOUNTSNOTESINPUT:
			uv.notesInput.Blur()
		}

		switch message.Direction {
		case meta.PREVIOUS:
			uv.activeInput.previous()

		case meta.NEXT:
			uv.activeInput.next()
		}

		// If now on a textinput, focus it
		switch uv.activeInput {
		case ACCOUNTSNAMEINPUT:
			uv.nameInput.Focus()
		case ACCOUNTSBANKNUMBERSINPUT:
			uv.bankNumbersInput.Focus()
		case ACCOUNTSNOTESINPUT:
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
		case ACCOUNTSNAMEINPUT:
			uv.nameInput, cmd = uv.nameInput.Update(message)
		case ACCOUNTSTYPEINPUT:
			uv.typeInput, cmd = uv.typeInput.Update(message)
		case ACCOUNTSBANKNUMBERSINPUT:
			uv.bankNumbersInput, cmd = uv.bankNumbersInput.Update(message)
		case ACCOUNTSNOTESINPUT:
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

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(uv.colours.Foreground)

	const inputWidth = 26
	uv.nameInput.Width = inputWidth - 2
	uv.bankNumbersInput.SetWidth(inputWidth)
	uv.notesInput.SetWidth(inputWidth)

	notesFocusStyle := lipgloss.NewStyle().Foreground(uv.colours.Foreground)
	uv.bankNumbersInput.FocusedStyle.Prompt = notesFocusStyle
	uv.bankNumbersInput.FocusedStyle.Text = notesFocusStyle
	uv.bankNumbersInput.FocusedStyle.CursorLine = notesFocusStyle
	uv.bankNumbersInput.FocusedStyle.LineNumber = notesFocusStyle
	uv.notesInput.FocusedStyle.Prompt = notesFocusStyle
	uv.notesInput.FocusedStyle.Text = notesFocusStyle
	uv.notesInput.FocusedStyle.CursorLine = notesFocusStyle
	uv.notesInput.FocusedStyle.LineNumber = notesFocusStyle

	nameStyle := sectionStyle
	typeStyle := sectionStyle

	switch uv.activeInput {
	case ACCOUNTSNAMEINPUT:
		nameStyle = highlightStyle
	case ACCOUNTSTYPEINPUT:
		typeStyle = highlightStyle
	// textareas have FocusedStyle set, don't manually render with highlightStyle
	case ACCOUNTSBANKNUMBERSINPUT:
	case ACCOUNTSNOTESINPUT:
	default:
		panic(fmt.Sprintf("unexpected view.accountsActiveInput: %#v", uv.activeInput))
	}

	// +2 for padding
	maxNameColWidth := len("Bank numbers") + 2

	nameRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Name"),
		" ",
		nameStyle.Render(uv.nameInput.View()),
	)

	typeRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Type"),
		" ",
		typeStyle.Render(uv.typeInput.View()),
	)

	bankNumbersRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Bank numbers"),
		" ",
		sectionStyle.Render(uv.bankNumbersInput.View()),
	)

	notesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sectionStyle.Width(maxNameColWidth).Render("Notes"),
		" ",
		sectionStyle.Render(uv.notesInput.View()),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Top,
			nameRow,
			typeRow,
			bankNumbersRow,
			notesRow,
		),
	))

	return result.String()
}

func (uv *AccountsUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (uv *AccountsUpdateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (uv *AccountsUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *AccountsUpdateView) Reload() View {
	return NewAccountsUpdateView(uv.modelId)
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

func NewAccountsDeleteView(modelId int) *AccountsDeleteView {
	return &AccountsDeleteView{
		modelId: modelId,

		colours: meta.ACCOUNTSCOLOURS,
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

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

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
		UnsetWidth()
	rightStyle := style.Margin(0, 0, 0, 1)

	// +2 for padding
	maxNameColWidth := len("Bank numbers") + 2

	nameRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Width(maxNameColWidth).Render("Name"),
		rightStyle.Render(dv.model.Name),
	)

	typeRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Width(maxNameColWidth).Render("Type"),
		rightStyle.Render(dv.model.Type.String()),
	)

	bankNumbersRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Width(maxNameColWidth).Render("Bank Numbers"),
		rightStyle.Render(dv.model.BankNumbers.Collapse()),
	)

	notesRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		style.Width(maxNameColWidth).Render("Notes"),
		rightStyle.AlignHorizontal(lipgloss.Left).Render(dv.model.Notes.Collapse()),
	)

	confirmRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Italic(true).Render("Run the `:w` command to confirm"),
	)

	result.WriteString(lipgloss.NewStyle().MarginLeft(2).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			nameRow,
			typeRow,
			bankNumbersRow,
			notesRow,
			"",
			confirmRow,
		),
	))

	return result.String()
}

func (dv *AccountsDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (dv *AccountsDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *AccountsDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *AccountsDeleteView) Reload() View {
	return NewAccountsDeleteView(dv.modelId)
}

func (dv *AccountsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
