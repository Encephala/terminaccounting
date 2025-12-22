package view

import (
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"terminaccounting/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ledgerMutateView interface {
	View

	title() string

	getNameInput() *textinput.Model
	getTypeInput() *itempicker.Model
	getNotesInput() *textarea.Model

	getActiveInput() *int

	getColours() meta.AppColours
}

type ledgersCreateView struct {
	nameInput   textinput.Model
	typeInput   itempicker.Model
	notesInput  textarea.Model
	activeInput int

	colours meta.AppColours
}

func NewLedgersCreateView() *ledgersCreateView {
	colours := meta.LEDGERSCOLOURS

	ledgerTypes := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	const baseInputWidth = 26
	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Width = baseInputWidth - 2 - 1
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput.SetWidth(baseInputWidth)

	notesFocusStyle := lipgloss.NewStyle().Foreground(colours.Foreground)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	return &ledgersCreateView{
		nameInput:   nameInput,
		typeInput:   itempicker.New(ledgerTypes),
		notesInput:  notesInput,
		activeInput: NAMEINPUT,

		colours: colours,
	}
}

func (cv *ledgersCreateView) Init() tea.Cmd {
	return nil
}

func (cv *ledgersCreateView) title() string {
	return "Create new Ledger"
}

func (cv *ledgersCreateView) getNameInput() *textinput.Model {
	return &cv.nameInput
}
func (cv *ledgersCreateView) getTypeInput() *itempicker.Model {
	return &cv.typeInput
}
func (cv *ledgersCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}
func (cv *ledgersCreateView) getActiveInput() *int {
	return &cv.activeInput
}
func (cv *ledgersCreateView) getColours() meta.AppColours {
	return cv.colours
}

func (cv *ledgersCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		ledgerName := cv.nameInput.Value()
		ledgerType := cv.typeInput.Value().(database.LedgerType)
		ledgerNotes := cv.notesInput.Value()

		newLedger := database.Ledger{
			Name:  ledgerName,
			Type:  ledgerType,
			Notes: meta.CompileNotes(ledgerNotes),
		}

		id, err := newLedger.Insert()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Account %q", cv.nameInput.Value(),
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		}))

		return cv, tea.Batch(cmds...)

	default:
		return ledgersMutateViewUpdate(cv, message)
	}
}

func (cv *ledgersCreateView) View() string {
	panic("never call this ye?")
}

func (cv *ledgersCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *ledgersCreateView) MotionSet() meta.MotionSet {
	return ledgersMutateViewMotionSet()
}

func (cv *ledgersCreateView) CommandSet() meta.CommandSet {
	return ledgersMutateViewCommandSet()
}

func (cv *ledgersCreateView) Reload() View {
	return NewLedgersCreateView()
}

func (cv *ledgersCreateView) inputNames() []string {
	return []string{"Name", "Type", "Notes"}
}

func (cv *ledgersCreateView) inputs() []viewable {
	return []viewable{cv.nameInput, cv.typeInput, cv.notesInput}
}

type ledgersUpdateView struct {
	nameInput   textinput.Model
	typeInput   itempicker.Model
	notesInput  textarea.Model
	activeInput int

	modelId       int
	startingValue database.Ledger

	colours meta.AppColours
}

func NewLedgersUpdateView(modelId int) *ledgersUpdateView {
	types := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	const baseInputWidth = 26
	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Width = baseInputWidth - 2 - 1
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	typeInput := itempicker.New(types)
	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)
	notesInput.SetWidth(baseInputWidth)

	return &ledgersUpdateView{
		nameInput:   nameInput,
		typeInput:   typeInput,
		notesInput:  notesInput,
		activeInput: NAMEINPUT,

		modelId: modelId,

		colours: meta.LEDGERSCOLOURS,
	}
}

func (uv *ledgersUpdateView) Init() tea.Cmd {
	return database.MakeLoadLedgersDetailCmd(uv.modelId)
}

func (uv *ledgersUpdateView) title() string {
	return fmt.Sprintf("Update Ledger: %s", uv.nameInput.Value())
}
func (uv *ledgersUpdateView) getNameInput() *textinput.Model {
	return &uv.nameInput
}
func (uv *ledgersUpdateView) getTypeInput() *itempicker.Model {
	return &uv.typeInput
}
func (uv *ledgersUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}
func (uv *ledgersUpdateView) getActiveInput() *int {
	return &uv.activeInput
}
func (uv *ledgersUpdateView) getColours() meta.AppColours {
	return uv.colours
}

func (uv *ledgersUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		ledger := message.Data.(database.Ledger)

		uv.startingValue = ledger

		uv.nameInput.SetValue(ledger.Name)
		err := uv.typeInput.SetValue(ledger.Type)
		uv.notesInput.SetValue(ledger.Notes.Collapse())

		return uv, meta.MessageCmd(err)

	case meta.ResetInputFieldMsg:
		var err error

		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput.SetValue(uv.startingValue.Name)
		case TYPEINPUT:
			err = uv.typeInput.SetValue(uv.startingValue.Type)
		case NOTEINPUT:
			uv.notesInput.SetValue(uv.startingValue.Notes.Collapse())
		}

		return uv, meta.MessageCmd(err)

	case meta.CommitMsg:
		ledger := database.Ledger{
			Id:    uv.modelId,
			Name:  uv.nameInput.Value(),
			Type:  uv.typeInput.Value().(database.LedgerType),
			Notes: meta.CompileNotes(uv.notesInput.Value()),
		}

		err := ledger.Update()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Ledger %q", uv.nameInput.Value(),
		)})

	default:
		return ledgersMutateViewUpdate(uv, message)
	}
}

func (uv *ledgersUpdateView) View() string {
	panic("never call this ye?")
}

func (uv *ledgersUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL: {},
	}
}

func (uv *ledgersUpdateView) MotionSet() meta.MotionSet {
	result := ledgersMutateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *ledgersUpdateView) CommandSet() meta.CommandSet {
	return ledgersMutateViewCommandSet()
}

func (uv *ledgersUpdateView) Reload() View {
	return NewLedgersUpdateView(uv.modelId)
}

func (cv *ledgersUpdateView) inputNames() []string {
	return []string{"Name", "Type", "Notes"}
}

func (cv *ledgersUpdateView) inputs() []viewable {
	return []viewable{cv.nameInput, cv.typeInput, cv.notesInput}
}

func (uv *ledgersUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

// The common parts of the Update function for a create- and update view
func ledgersMutateViewUpdate(view ledgerMutateView, message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		// If currently on a textinput, blur it
		// Shouldn't matter too much because we only send the update to the right input, but FWIW
		// Note from later me: might actually delete this as an implicit check that only the right input
		// gets the update message.
		switch *view.getActiveInput() {
		case NAMEINPUT:
			view.getNameInput().Blur()
		case NOTEINPUT:
			view.getNotesInput().Blur()
		}

		switch message.Direction {
		case meta.PREVIOUS:
			previousInput(view.getActiveInput(), 3)

		case meta.NEXT:
			nextInput(view.getActiveInput(), 3)
		}

		// If now on a textinput, focus it
		switch *view.getActiveInput() {
		case NAMEINPUT:
			view.getNameInput().Focus()
		case NOTEINPUT:
			view.getNotesInput().Focus()
		}

		return view, nil

	case meta.NavigateMsg:
		return view, nil

	case tea.WindowSizeMsg:
		// -18 covers padding on both sides, name column and borders
		inputWidth := message.Width - 18
		// -2 for title, -6 for the name/type input, -2 for its borders and -1 for padding at bottom
		notesHeight := message.Height - 2 - 6 - 2 - 1

		// -2 because of the prompt, -1 because of the cursor
		view.getNameInput().Width = inputWidth - 2 - 1
		view.getNotesInput().SetWidth(inputWidth)
		view.getNotesInput().SetHeight(notesHeight)

		return view, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		switch *view.getActiveInput() {
		case NAMEINPUT:
			*view.getNameInput(), cmd = view.getNameInput().Update(message)
		case TYPEINPUT:
			*view.getTypeInput(), cmd = view.getTypeInput().Update(message)
		case NOTEINPUT:
			*view.getNotesInput(), cmd = view.getNotesInput().Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *view.getActiveInput()))
		}

		return view, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func ledgersMutateViewMotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func ledgersMutateViewCommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

type ledgersDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Ledger

	colours meta.AppColours
}

func NewLedgersDeleteView(modelId int) *ledgersDeleteView {
	return &ledgersDeleteView{
		modelId: modelId,

		colours: meta.LEDGERSCOLOURS,
	}
}

func (dv *ledgersDeleteView) Init() tea.Cmd {
	return database.MakeLoadLedgersDetailCmd(dv.modelId)
}

func (dv *ledgersDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

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
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Delete Ledger: %s", dv.model.Name)))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	rightStyle := style.Margin(0, 0, 0, 1)

	// TODO: Render active input with a different colour
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

func (dv *ledgersDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGERMODEL: {},
	}
}

func (dv *ledgersDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

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

func (dv *ledgersDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
