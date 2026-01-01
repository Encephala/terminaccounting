package view

import (
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"terminaccounting/bubbles/booleaninput"
	"terminaccounting/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ledgersDetailView struct {
	// The ledger whose rows are being shown
	modelId int
	model   database.Ledger

	canReconcile bool

	viewer *entryRowViewer
}

func NewLedgersDetailView(modelId int) *ledgersDetailView {
	return &ledgersDetailView{
		modelId: modelId,

		viewer: newEntryRowViewer(meta.LEDGERSCOLOUR),
	}
}

func (dv *ledgersDetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, database.MakeLoadLedgersDetailCmd(dv.modelId))
	cmds = append(cmds, database.MakeLoadLedgersRowsCmd(dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *ledgersDetailView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.LEDGERMODEL:
			dv.model = message.Data.(database.Ledger)

			switch dv.model.Type {
			case database.INCOMELEDGER, database.EXPENSELEDGER:
				dv.canReconcile = false

			case database.ASSETLEDGER, database.EQUITYLEDGER, database.LIABILITYLEDGER:
				dv.canReconcile = true

			default:
				panic(fmt.Sprintf("unexpected database.LedgerType: %#v", dv.model.Type))
			}

			return dv, nil

		case meta.ENTRYROWMODEL:
			return genericDetailViewUpdate(dv, message)

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}
	}

	return genericDetailViewUpdate(dv, message)
}

func (dv *ledgersDetailView) View() string {
	return genericDetailViewView(dv)
}

func (dv *ledgersDetailView) title() string {
	return fmt.Sprintf("Ledger %s details", dv.model.Name)
}

func (dv *ledgersDetailView) getCanReconcile() bool {
	return dv.canReconcile
}

func (dv *ledgersDetailView) AllowsInsertMode() bool {
	return false
}

func (dv *ledgersDetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL:   {},
		meta.ENTRYROWMODEL: {},
	}
}

func (dv *ledgersDetailView) MotionSet() meta.MotionSet {
	result := genericDetailViewMotionSet()

	result.Normal.Insert(meta.Motion{"g", "x"}, meta.SwitchAppViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	result.Normal.Insert(meta.Motion{"g", "e"}, meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	result.Normal.Insert(meta.Motion{"g", "d"}, makeGoToEntryDetailViewCmd(dv.viewer.getActiveRow()))

	return result
}

func (dv *ledgersDetailView) CommandSet() meta.CommandSet {
	return genericDetailViewCommandSet()
}

func (dv *ledgersDetailView) Reload() View {
	return NewLedgersDetailView(dv.modelId)
}

func (dv *ledgersDetailView) getViewer() *entryRowViewer {
	return dv.viewer
}

func (dv *ledgersDetailView) getColour() lipgloss.Color {
	return meta.LEDGERSCOLOUR
}

type ledgersCreateView struct {
	inputManager *inputManager

	colour lipgloss.Color
}

func NewLedgersCreateView() *ledgersCreateView {
	ledgerTypes := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	typeInput := itempicker.New(ledgerTypes)

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.LEDGERSCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	isAccountsInput := booleaninput.New()

	inputs := []any{nameInput, typeInput, notesInput, isAccountsInput}
	names := []string{"Name", "Type", "Notes", "Is accounts ledger?"}

	return &ledgersCreateView{
		inputManager: newInputManager(inputs, names),

		colour: meta.LEDGERSCOLOUR,
	}
}

func (cv *ledgersCreateView) Init() tea.Cmd {
	return nil
}

func (cv *ledgersCreateView) title() string {
	return "Create new Ledger"
}

func (cv *ledgersCreateView) getColour() lipgloss.Color {
	return cv.colour
}

func (cv *ledgersCreateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		name := cv.inputManager.inputs[0].value().(string)
		ledgerType := cv.inputManager.inputs[1].value().(database.LedgerType)
		notes := meta.CompileNotes(cv.inputManager.inputs[2].value().(string))
		isAccounts := cv.inputManager.inputs[3].value().(bool)

		currentAccountsLedger := database.GetAccountsLedger()

		if isAccounts && currentAccountsLedger != nil {
			return cv, meta.MessageCmd(fmt.Errorf("ledger %q already is accounts ledger, can't have multiple", currentAccountsLedger))
		}

		newLedger := database.Ledger{
			Name:       name,
			Type:       ledgerType,
			Notes:      notes,
			IsAccounts: isAccounts,
		}

		id, err := newLedger.Insert()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Account %q", name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)
	}

	return genericMutateViewUpdate(cv, message)
}

func (cv *ledgersCreateView) View() string {
	return genericMutateViewView(cv)
}

func (cv *ledgersCreateView) AllowsInsertMode() bool {
	return true
}

func (cv *ledgersCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *ledgersCreateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func (cv *ledgersCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *ledgersCreateView) Reload() View {
	return NewLedgersCreateView()
}

func (cv *ledgersCreateView) getInputManager() *inputManager {
	return cv.inputManager
}

type ledgersUpdateView struct {
	inputManager *inputManager

	modelId       int
	startingValue database.Ledger

	colour lipgloss.Color
}

func NewLedgersUpdateView(modelId int) *ledgersUpdateView {
	types := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	typeInput := itempicker.New(types)

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.LEDGERSCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	isAccountsInput := booleaninput.New()

	inputs := []any{nameInput, typeInput, notesInput, isAccountsInput}
	names := []string{"Name", "Type", "Notes", "Is accounts ledger?"}

	return &ledgersUpdateView{
		inputManager: newInputManager(inputs, names),

		modelId: modelId,

		colour: meta.LEDGERSCOLOUR,
	}
}

func (uv *ledgersUpdateView) Init() tea.Cmd {
	return database.MakeLoadLedgersDetailCmd(uv.modelId)
}

func (uv *ledgersUpdateView) title() string {
	return fmt.Sprintf("Update Ledger: %s", uv.inputManager.inputs[0].value().(string))
}
func (uv *ledgersUpdateView) getColour() lipgloss.Color {
	return uv.colour
}

func (uv *ledgersUpdateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		ledger := message.Data.(database.Ledger)

		uv.startingValue = ledger

		uv.inputManager.inputs[0].setValue(ledger.Name)
		err := uv.inputManager.inputs[1].setValue(ledger.Type)
		uv.inputManager.inputs[2].setValue(ledger.Notes.Collapse())
		uv.inputManager.inputs[3].setValue(ledger.IsAccounts)

		return uv, meta.MessageCmd(err)

	case meta.ResetInputFieldMsg:
		var startingValue any
		switch uv.inputManager.activeInput {
		case 0:
			startingValue = uv.startingValue.Name
		case 1:
			startingValue = uv.startingValue.Type
		case 2:
			startingValue = uv.startingValue.Notes.Collapse()
		case 3:
			startingValue = uv.startingValue.IsAccounts
		default:
			panic(fmt.Sprintf("unexpected activeInput: %d", uv.inputManager.activeInput))
		}

		err := uv.inputManager.inputs[uv.inputManager.activeInput].setValue(startingValue)

		return uv, meta.MessageCmd(err)

	case meta.CommitMsg:
		name := uv.inputManager.inputs[0].value().(string)
		ledgerType := uv.inputManager.inputs[1].value().(database.LedgerType)
		notes := meta.CompileNotes(uv.inputManager.inputs[2].value().(string))
		isAccounts := uv.inputManager.inputs[3].value().(bool)

		currentAccountsLedger := database.GetAccountsLedger()

		if isAccounts && currentAccountsLedger != nil {
			return uv, meta.MessageCmd(fmt.Errorf("ledger %q already is accounts ledger, can't have multiple", currentAccountsLedger))
		}

		ledger := database.Ledger{
			Id:         uv.modelId,
			Name:       name,
			Type:       ledgerType,
			Notes:      notes,
			IsAccounts: isAccounts,
		}

		err := ledger.Update()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Ledger %q", name,
		)})
	}

	return genericMutateViewUpdate(uv, message)
}

func (uv *ledgersUpdateView) View() string {
	return genericMutateViewView(uv)
}

func (uv *ledgersUpdateView) AllowsInsertMode() bool {
	return true
}

func (uv *ledgersUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL: {},
	}
}

func (uv *ledgersUpdateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (uv *ledgersUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *ledgersUpdateView) Reload() View {
	return NewLedgersUpdateView(uv.modelId)
}

func (uv *ledgersUpdateView) getInputManager() *inputManager {
	return uv.inputManager
}

func (uv *ledgersUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

type ledgersDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Ledger

	colour lipgloss.Color
}

func NewLedgersDeleteView(modelId int) *ledgersDeleteView {
	return &ledgersDeleteView{
		modelId: modelId,

		colour: meta.LEDGERSCOLOUR,
	}
}

func (dv *ledgersDeleteView) Init() tea.Cmd {
	return database.MakeLoadLedgersDetailCmd(dv.modelId)
}

func (dv *ledgersDeleteView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(database.Ledger)

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteLedger(dv.modelId)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Ledger %q", dv.model.Name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// Not much to do, view automatically updates with size of name/notes etc.
		// TODO: when View() is updated to draw columns, do some stuff here to make columns max out at width of view
		// with the truncate package

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *ledgersDeleteView) View() string {
	return genericDeleteViewView(dv)
}

func (dv *ledgersDeleteView) AllowsInsertMode() bool {
	return false
}

func (dv *ledgersDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL: {},
	}
}

func (dv *ledgersDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *ledgersDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *ledgersDeleteView) Reload() View {
	return NewLedgersDeleteView(dv.modelId)
}

func (dv *ledgersDeleteView) title() string {
	return fmt.Sprintf("Delete ledger %s", dv.model.String())
}

func (dv *ledgersDeleteView) inputValues() []string {
	return []string{dv.model.Name, dv.model.Type.String(), dv.model.Notes.Collapse(), renderBoolean(dv.model.IsAccounts)}
}

func (dv *ledgersDeleteView) inputNames() []string {
	return []string{"Name", "Type", "Notes", "Is accounts ledger"}
}

func (dv *ledgersDeleteView) getColour() lipgloss.Color {
	return dv.colour
}

func (dv *ledgersDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
