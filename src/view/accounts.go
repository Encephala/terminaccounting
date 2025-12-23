package view

import (
	"fmt"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	ACCOUNTSNAMEINPUT int = iota
	ACCOUNTSTYPEINPUT
	ACCOUNTSBANKNUMBERSINPUT
	ACCOUNTSNOTESINPUT
)

const NUMACCOUNTSINPUTS int = 4

type accountsCreateView struct {
	nameInput        textinput.Model
	typeInput        itempicker.Model
	bankNumbersInput textarea.Model
	notesInput       textarea.Model
	activeInput      int

	colours meta.AppColours
}

func NewAccountsCreateView() *accountsCreateView {
	colours := meta.ACCOUNTSCOLOURS

	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	const baseInputWidth = 26
	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Width = baseInputWidth - 2 - 1
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(colours.Foreground)
	bankNumbersInput.FocusedStyle.Prompt = notesFocusStyle
	bankNumbersInput.FocusedStyle.Text = notesFocusStyle
	bankNumbersInput.FocusedStyle.CursorLine = notesFocusStyle
	bankNumbersInput.FocusedStyle.LineNumber = notesFocusStyle
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	return &accountsCreateView{
		nameInput:        nameInput,
		typeInput:        itempicker.New(accountTypes),
		notesInput:       notesInput,
		bankNumbersInput: bankNumbersInput,
		activeInput:      ACCOUNTSNAMEINPUT,

		colours: colours,
	}
}

func (cv *accountsCreateView) Init() tea.Cmd {
	return nil
}

func (cv *accountsCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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
			previousInput(&cv.activeInput, 4)

		case meta.NEXT:
			nextInput(&cv.activeInput, 4)
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

func (cv *accountsCreateView) View() string {
	return genericMutateViewView(cv)
}

func (cv *accountsCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *accountsCreateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func (cv *accountsCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *accountsCreateView) Reload() View {
	return NewAccountsCreateView()
}

func (cv *accountsCreateView) title() string {
	return "Creating new account"
}

func (cv *accountsCreateView) inputNames() []string {
	return []string{"Name", "Type", "Bank numbers", "Notes"}
}

func (cv *accountsCreateView) inputs() []viewable {
	return []viewable{cv.nameInput, cv.typeInput, cv.bankNumbersInput, cv.notesInput}
}

func (cv *accountsCreateView) getActiveInput() *int {
	return (*int)(&cv.activeInput)
}

func (cv *accountsCreateView) getColours() meta.AppColours {
	return cv.colours
}

type accountsUpdateView struct {
	nameInput        textinput.Model
	typeInput        itempicker.Model
	bankNumbersInput textarea.Model
	notesInput       textarea.Model
	activeInput      int

	modelId       int
	startingValue database.Account

	colours meta.AppColours
}

func NewAccountsUpdateView(modelId int) *accountsUpdateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	const baseInputWidth = 26
	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Width = baseInputWidth - 2 - 1
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	return &accountsUpdateView{
		nameInput:        nameInput,
		typeInput:        itempicker.New(accountTypes),
		bankNumbersInput: bankNumbersInput,
		notesInput:       notesInput,
		activeInput:      ACCOUNTSNAMEINPUT,

		modelId: modelId,

		colours: meta.ACCOUNTSCOLOURS,
	}
}

func (uv *accountsUpdateView) Init() tea.Cmd {
	return database.MakeLoadAccountsDetailCmd(uv.modelId)
}

func (uv *accountsUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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
			previousInput(&uv.activeInput, 4)

		case meta.NEXT:
			nextInput(&uv.activeInput, 4)
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

func (uv *accountsUpdateView) View() string {
	return genericMutateViewView(uv)
}

func (uv *accountsUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (uv *accountsUpdateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (uv *accountsUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *accountsUpdateView) Reload() View {
	return NewAccountsUpdateView(uv.modelId)
}

func (uv *accountsUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

func (cv *accountsUpdateView) title() string {
	return "Creating new account"
}

func (cv *accountsUpdateView) inputNames() []string {
	return []string{"Name", "Type", "Bank numbers", "Notes"}
}

func (cv *accountsUpdateView) inputs() []viewable {
	return []viewable{cv.nameInput, cv.typeInput, cv.bankNumbersInput, cv.notesInput}
}

func (cv *accountsUpdateView) getActiveInput() *int {
	return (*int)(&cv.activeInput)
}

func (cv *accountsUpdateView) getColours() meta.AppColours {
	return cv.colours
}

type accountsDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Account

	colours meta.AppColours
}

func NewAccountsDeleteView(modelId int) *accountsDeleteView {
	return &accountsDeleteView{
		modelId: modelId,

		colours: meta.ACCOUNTSCOLOURS,
	}
}

func (dv *accountsDeleteView) Init() tea.Cmd {
	return database.MakeLoadAccountsDetailCmd(dv.modelId)
}

func (dv *accountsDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

func (dv *accountsDeleteView) View() string {
	return genericDeleteViewView(dv)
}

func (dv *accountsDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (dv *accountsDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *accountsDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *accountsDeleteView) Reload() View {
	return NewAccountsDeleteView(dv.modelId)
}

func (dv *accountsDeleteView) title() string {
	return fmt.Sprintf("Delete account: %s", dv.model.String())
}

func (dv *accountsDeleteView) inputValues() []string {
	return []string{dv.model.Name, dv.model.Type.String(), dv.model.BankNumbers.Collapse(), dv.model.Notes.Collapse()}
}

func (dv *accountsDeleteView) inputNames() []string {
	return []string{"Name", "Type", "Bank numbers", "Notes"}
}

func (dv *accountsDeleteView) getColours() meta.AppColours {
	return dv.colours
}

func (dv *accountsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
