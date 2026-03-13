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
	"github.com/jmoiron/sqlx"
)

type accountsDetailView struct {
	DB *sqlx.DB

	// The account whose rows are being shown
	modelId int
	model   database.Account

	viewer *entryRowViewer
}

func NewAccountsDetailView(DB *sqlx.DB, modelId int) *accountsDetailView {
	return &accountsDetailView{
		DB: DB,

		modelId: modelId,

		viewer: newEntryRowViewer(meta.ACCOUNTSCOLOUR),
	}
}

func (dv *accountsDetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeLoadAccountsDetailCmd(dv.DB, dv.modelId))
	cmds = append(cmds, database.MakeLoadAccountsRowsCmd(dv.DB, dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *accountsDetailView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ACCOUNTMODEL:
			dv.model = message.Data.(database.Account)

			return dv, nil

		case meta.ENTRYROWMODEL:
			return genericDetailViewUpdate(dv, message)

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}
	}

	return genericDetailViewUpdate(dv, message)
}

func (dv *accountsDetailView) View() string {
	return genericDetailViewView(dv)
}

func (dv *accountsDetailView) Title() string {
	style := lipgloss.NewStyle().Background(meta.ACCOUNTSCOLOUR).Padding(0, 1)
	return style.Render(fmt.Sprintf("Account %s details", dv.model.Name))
}

func (dv *accountsDetailView) Type() meta.ViewType {
	return meta.DETAILVIEWTYPE
}

func (dv *accountsDetailView) getDB() *sqlx.DB {
	return dv.DB
}

func (dv *accountsDetailView) getCanReconcile() bool {
	return true
}

func (dv *accountsDetailView) AllowsInsertMode() bool {
	return false
}

func (dv *accountsDetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL:  {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *accountsDetailView) MotionSet() meta.MotionSet {
	result := genericDetailViewMotionSet()

	result.Insert(meta.Motion{"g", "x"}, meta.SwitchAppViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	result.Insert(meta.Motion{"g", "e"}, meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	result.Insert(meta.Motion{"g", "d"}, makeGoToEntryDetailViewCmd(dv.DB, dv.viewer.getActiveRow()))

	return result
}

func (dv *accountsDetailView) CommandSet() meta.CommandSet {
	return genericDetailViewCommandSet()
}

func (dv *accountsDetailView) Reload() View {
	return NewAccountsDetailView(dv.DB, dv.modelId)
}

func (dv *accountsDetailView) getViewer() *entryRowViewer {
	return dv.viewer
}

const (
	ACCOUNTSNAMEINPUT int = iota
	ACCOUNTSTYPEINPUT
	ACCOUNTSBANKNUMBERSINPUT
	ACCOUNTSNOTESINPUT
)

const NUMACCOUNTSINPUTS int = 4

type accountsCreateView struct {
	DB *sqlx.DB

	inputManager *inputManager

	colour lipgloss.Color
}

func NewAccountsCreateView(DB *sqlx.DB) *accountsCreateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	typeInput := itempicker.New(accountTypes)

	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.ACCOUNTSCOLOUR)
	bankNumbersInput.FocusedStyle.Prompt = notesFocusStyle
	bankNumbersInput.FocusedStyle.Text = notesFocusStyle
	bankNumbersInput.FocusedStyle.CursorLine = notesFocusStyle
	bankNumbersInput.FocusedStyle.LineNumber = notesFocusStyle
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	inputs := []any{nameInput, typeInput, bankNumbersInput, notesInput}
	names := []string{"Name", "Type", "Bank numbers", "Notes"}

	return &accountsCreateView{
		DB: DB,

		inputManager: newInputManager(inputs, names),

		colour: meta.ACCOUNTSCOLOUR,
	}
}

func (cv *accountsCreateView) Init() tea.Cmd {
	return nil
}

func (cv *accountsCreateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		name := cv.inputManager.inputs[0].value().(string)
		accountType := cv.inputManager.inputs[1].value().(database.AccountType)
		bankNumbers := meta.CompileNotes(cv.inputManager.inputs[2].value().(string))
		notes := meta.CompileNotes(cv.inputManager.inputs[3].value().(string))

		newAccount := database.Account{
			Name:        name,
			Type:        accountType,
			BankNumbers: bankNumbers,
			Notes:       notes,
		}

		id, err := newAccount.Insert(cv.DB)
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Account %q", name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)

	case meta.NavigateMsg:
		return cv, nil
	}

	return genericMutateViewUpdate(cv, message)
}

func (cv *accountsCreateView) View() string {
	return genericMutateViewView(cv, meta.ACCOUNTSCOLOUR)
}

func (cv *accountsCreateView) Title() string {
	style := lipgloss.NewStyle().Background(meta.ACCOUNTSCOLOUR).Padding(0, 1)
	return style.Render("Creating new account")
}

func (cv *accountsCreateView) Type() meta.ViewType {
	return meta.CREATEVIEWTYPE
}

func (cv *accountsCreateView) AllowsInsertMode() bool {
	return true
}

func (cv *accountsCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *accountsCreateView) MotionSet() meta.MotionSet {
	var motions meta.MotionSet

	motions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	motions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	motions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return motions
}

func (cv *accountsCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *accountsCreateView) Reload() View {
	return NewAccountsCreateView(cv.DB)
}

func (cv *accountsCreateView) getInputManager() *inputManager {
	return cv.inputManager
}

type accountsUpdateView struct {
	DB *sqlx.DB

	inputManager *inputManager

	modelId       int
	startingValue database.Account

	colour lipgloss.Color
}

func NewAccountsUpdateView(DB *sqlx.DB, modelId int) *accountsUpdateView {
	accountTypes := []itempicker.Item{
		database.DEBTOR,
		database.CREDITOR,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	typeInput := itempicker.New(accountTypes)

	bankNumbersInput := textarea.New()
	bankNumbersInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.ACCOUNTSCOLOUR)
	bankNumbersInput.FocusedStyle.Prompt = notesFocusStyle
	bankNumbersInput.FocusedStyle.Text = notesFocusStyle
	bankNumbersInput.FocusedStyle.CursorLine = notesFocusStyle
	bankNumbersInput.FocusedStyle.LineNumber = notesFocusStyle
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	inputs := []any{nameInput, typeInput, bankNumbersInput, notesInput}
	names := []string{"Name", "Type", "Bank numbers", "Notes"}

	return &accountsUpdateView{
		DB: DB,

		inputManager: newInputManager(inputs, names),

		modelId: modelId,

		colour: meta.ACCOUNTSCOLOUR,
	}
}

func (uv *accountsUpdateView) Init() tea.Cmd {
	return database.MakeLoadAccountsDetailCmd(uv.DB, uv.modelId)
}

func (uv *accountsUpdateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the account being edited
		account := message.Data.(database.Account)

		uv.startingValue = account

		uv.inputManager.inputs[0].setValue(account.Name)
		err := uv.inputManager.inputs[1].setValue(account.Type)
		uv.inputManager.inputs[2].setValue(account.BankNumbers.Collapse())
		uv.inputManager.inputs[3].setValue(account.Notes.Collapse())

		return uv, meta.MessageCmd(err)

	case meta.ResetInputFieldMsg:
		var startingValue any
		switch uv.inputManager.activeInput {
		case 0:
			startingValue = uv.startingValue.Name
		case 1:
			startingValue = uv.startingValue.Type
		case 2:
			startingValue = uv.startingValue.BankNumbers
		case 3:
			startingValue = uv.startingValue.Notes.Collapse()
		default:
			panic(fmt.Sprintf("unexpected activeInput: %d", uv.inputManager.activeInput))
		}

		err := uv.inputManager.inputs[uv.inputManager.activeInput].setValue(startingValue)

		return uv, meta.MessageCmd(err)

	case meta.CommitMsg:
		name := uv.inputManager.inputs[0].value().(string)
		accountType := uv.inputManager.inputs[1].value().(database.AccountType)
		bankNumbers := meta.CompileNotes(uv.inputManager.inputs[2].value().(string))
		notes := meta.CompileNotes(uv.inputManager.inputs[3].value().(string))

		account := database.Account{
			Id:          uv.modelId,
			Name:        name,
			Type:        accountType,
			BankNumbers: bankNumbers,
			Notes:       notes,
		}

		err := account.Update(uv.DB)
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Account %q", name,
		)})

	case meta.NavigateMsg:
		return uv, nil
	}

	return genericMutateViewUpdate(uv, message)
}

func (uv *accountsUpdateView) View() string {
	return genericMutateViewView(uv, meta.ACCOUNTSCOLOUR)
}

func (uv *accountsUpdateView) Title() string {
	style := lipgloss.NewStyle().Background(meta.ACCOUNTSCOLOUR).Padding(0, 1)
	return style.Render(fmt.Sprintf("Updating account: %s", uv.inputManager.inputs[0].value()))
}

func (uv *accountsUpdateView) Type() meta.ViewType {
	return meta.UPDATEVIEWTYPE
}

func (uv *accountsUpdateView) AllowsInsertMode() bool {
	return true
}

func (uv *accountsUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (uv *accountsUpdateView) MotionSet() meta.MotionSet {
	var motions meta.MotionSet

	motions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	motions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	motions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	motions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	motions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return motions
}

func (uv *accountsUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *accountsUpdateView) Reload() View {
	return NewAccountsUpdateView(uv.DB, uv.modelId)
}

func (uv *accountsUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

func (uv *accountsUpdateView) getInputManager() *inputManager {
	return uv.inputManager
}

type accountsDeleteView struct {
	DB *sqlx.DB

	width, height int

	modelId int // only for retrieving the model itself initially
	model   database.Account

	colour lipgloss.Color
}

func NewAccountsDeleteView(DB *sqlx.DB, modelId int) *accountsDeleteView {
	return &accountsDeleteView{
		DB: DB,

		modelId: modelId,

		colour: meta.ACCOUNTSCOLOUR,
	}
}

func (dv *accountsDeleteView) Init() tea.Cmd {
	return database.MakeLoadAccountsDetailCmd(dv.DB, dv.modelId)
}

func (dv *accountsDeleteView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(database.Account)

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteAccount(dv.DB, dv.modelId)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Account %q", dv.model.Name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		dv.width = message.Width
		dv.height = message.Height

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *accountsDeleteView) View() string {
	return genericDeleteViewView(dv, dv.width, dv.height)
}

func (dv *accountsDeleteView) Title() string {
	style := lipgloss.NewStyle().Background(meta.ACCOUNTSCOLOUR).Padding(0, 1)
	return style.Render(fmt.Sprintf("Delete account: %s", dv.model.String()))
}

func (dv *accountsDeleteView) Type() meta.ViewType {
	return meta.DELETEVIEWTYPE
}

func (dv *accountsDeleteView) AllowsInsertMode() bool {
	return true
}

func (dv *accountsDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
	}
}

func (dv *accountsDeleteView) MotionSet() meta.MotionSet {
	var motions meta.MotionSet

	motions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	motions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return motions
}

func (dv *accountsDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *accountsDeleteView) Reload() View {
	return NewAccountsDeleteView(dv.DB, dv.modelId)
}

func (dv *accountsDeleteView) inputValues() []string {
	return []string{dv.model.Name, dv.model.Type.String(), dv.model.BankNumbers.Collapse(), dv.model.Notes.Collapse()}
}

func (dv *accountsDeleteView) inputNames() []string {
	return []string{"Name", "Type", "Bank numbers", "Notes"}
}

func (dv *accountsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
