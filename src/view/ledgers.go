package view

import (
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"local/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ledgerCreateOrUpdateView interface {
	meta.View

	title() string

	getNameInput() *textinput.Model
	getTypeInput() *itempicker.Model
	getNotesInput() *textarea.Model

	getActiveInput() *activeInput

	getColours() meta.AppColours
}

type LedgersCreateView struct {
	nameInput  textinput.Model
	typeInput  itempicker.Model
	notesInput textarea.Model
	activeInput

	colours meta.AppColours
}

func NewLedgersCreateView(colours meta.AppColours) *LedgersCreateView {
	ledgerTypes := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	result := &LedgersCreateView{
		nameInput:   nameInput,
		typeInput:   itempicker.New(ledgerTypes),
		notesInput:  noteInput,
		activeInput: NAMEINPUT,

		colours: colours,
	}

	return result
}

func (cv *LedgersCreateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(cv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(cv.CommandSet())))

	return tea.Batch(cmds...)
}

func (cv *LedgersCreateView) title() string {
	return "Create new Ledger"
}

// NOTE from future me: I'm not sure why these exist?
// I think just to take a reference without having to write & everywhere?
func (cv *LedgersCreateView) getNameInput() *textinput.Model {
	return &cv.nameInput
}
func (cv *LedgersCreateView) getTypeInput() *itempicker.Model {
	return &cv.typeInput
}
func (cv *LedgersCreateView) getNotesInput() *textarea.Model {
	return &cv.notesInput
}
func (cv *LedgersCreateView) getActiveInput() *activeInput {
	return &cv.activeInput
}
func (cv *LedgersCreateView) getColours() meta.AppColours {
	return cv.colours
}

func (cv *LedgersCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

		// TODO: Add a vimesque message to inform the user of successful creation (when vimesque messages are implemented)
		// Or maybe this should just switch to the list view or the detail view? Idk

		return cv, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		})

	default:
		return ledgersCreateUpdateViewUpdate(cv, message)
	}
}

func (cv *LedgersCreateView) View() string {
	return ledgersCreateUpdateViewView(cv)
}

func (cv *LedgersCreateView) MotionSet() *meta.MotionSet {
	return ledgersCreateUpdateViewMotionSet()
}

func (cv *LedgersCreateView) CommandSet() *meta.CommandSet {
	return ledgersCreateUpdateViewCommandSet()
}

type LedgersUpdateView struct {
	nameInput   textinput.Model
	typeInput   itempicker.Model
	notesInput  textarea.Model
	activeInput activeInput

	modelId       int
	startingValue database.Ledger

	colours meta.AppColours
}

func NewLedgersUpdateView(modelId int, colours meta.AppColours) *LedgersUpdateView {
	types := []itempicker.Item{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	typeInput := itempicker.New(types)
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	return &LedgersUpdateView{
		nameInput:   nameInput,
		typeInput:   typeInput,
		notesInput:  noteInput,
		activeInput: NAMEINPUT,

		modelId: modelId,

		colours: colours,
	}
}

func (uv *LedgersUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(uv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(uv.CommandSet())))

	cmds = append(cmds, database.MakeLoadLedgersDetailCmd(uv.modelId))

	return tea.Batch(cmds...)
}

func (uv *LedgersUpdateView) title() string {
	return fmt.Sprintf("Update Ledger: %s", uv.nameInput.Value())
}
func (uv *LedgersUpdateView) getNameInput() *textinput.Model {
	return &uv.nameInput
}
func (uv *LedgersUpdateView) getTypeInput() *itempicker.Model {
	return &uv.typeInput
}
func (uv *LedgersUpdateView) getNotesInput() *textarea.Model {
	return &uv.notesInput
}
func (uv *LedgersUpdateView) getActiveInput() *activeInput {
	return &uv.activeInput
}
func (uv *LedgersUpdateView) getColours() meta.AppColours {
	return uv.colours
}

func (uv *LedgersUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		ledger := message.Data.(database.Ledger)

		uv.startingValue = ledger

		uv.nameInput.SetValue(ledger.Name)
		uv.typeInput.SetValue(ledger.Type)
		uv.notesInput.SetValue(ledger.Notes.Collapse())

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
		ledger := database.Ledger{
			Id:    uv.modelId,
			Name:  uv.nameInput.Value(),
			Type:  uv.typeInput.Value().(database.LedgerType),
			Notes: meta.CompileNotes(uv.notesInput.Value()),
		}

		ledger.Update()

		return uv, nil

	default:
		return ledgersCreateUpdateViewUpdate(uv, message)
	}
}

func (uv *LedgersUpdateView) View() string {
	return ledgersCreateUpdateViewView(uv)
}

func (uv *LedgersUpdateView) MotionSet() *meta.MotionSet {
	result := ledgersCreateUpdateViewMotionSet()

	result.Normal.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	result.Normal.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return result
}

func (uv *LedgersUpdateView) CommandSet() *meta.CommandSet {
	return ledgersCreateUpdateViewCommandSet()
}

func (uv *LedgersUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

// The common parts of the Update function for a create- and update view
func ledgersCreateUpdateViewUpdate(view ledgerCreateOrUpdateView, message tea.Msg) (tea.Model, tea.Cmd) {
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
			*view.getActiveInput()--
			if *view.getActiveInput() < 0 {
				*view.getActiveInput() += 3
			}

		case meta.NEXT:
			*view.getActiveInput()++
			*view.getActiveInput() %= 3
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
		// TODO

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

// The common parts of the View function for a create- and update view
func ledgersCreateUpdateViewView(view ledgerCreateOrUpdateView) string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(view.getColours().Background).Padding(0, 1)

	result.WriteString(fmt.Sprintf("  %s", titleStyle.Render(view.title())))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	const inputWidth = 26
	view.getNameInput().Width = inputWidth - 2 // -2 because of the prompt
	view.getNotesInput().SetWidth(inputWidth)

	// TODO: Render active input with a different colour
	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Name"),
		" ",
		style.Render(view.getNameInput().View()),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Type"),
		" ",
		style.Width(view.getTypeInput().MaxViewLength()+2).AlignHorizontal(lipgloss.Left).Render(view.getTypeInput().View()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.Render(view.getNotesInput().View()),
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

func ledgersCreateUpdateViewMotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{Normal: normalMotions}
}

func ledgersCreateUpdateViewCommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

type LedgersDeleteView struct {
	modelId int // only for retrieving the model itself initially
	model   database.Ledger

	colours meta.AppColours
}

func NewLedgersDeleteView(modelId int, colours meta.AppColours) *LedgersDeleteView {
	return &LedgersDeleteView{
		modelId: modelId,

		colours: colours,
	}
}

func (dv *LedgersDeleteView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(dv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(dv.CommandSet())))

	cmds = append(cmds, database.MakeLoadLedgersDetailCmd(dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *LedgersDeleteView) title() string {
	return fmt.Sprintf("Delete Ledger: %s", dv.model.Name)
}

func (dv *LedgersDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(database.Ledger)

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteLedger(dv.modelId)

		// TODO: Add a vimesque message to inform user of successful deletion
		var cmds []tea.Cmd

		if err != nil {
			cmds = append(cmds, meta.MessageCmd(err))
		}

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *LedgersDeleteView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render(dv.title()))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)

	// TODO: Render active input with a different colour
	var nameRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Name"),
		" ",
		style.Render(dv.model.Name),
	)

	var typeRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Type"),
		" ",
		style.Render(dv.model.Type.String()),
	)

	var notesRow = lipgloss.JoinHorizontal(
		lipgloss.Top,
		"  ",
		style.Render("Note"),
		" ",
		style.AlignHorizontal(lipgloss.Left).Render(dv.model.Notes.Collapse()),
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

func (dv *LedgersDeleteView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *LedgersDeleteView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

func (dv *LedgersDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		ledgerId := dv.model.Id

		ledger, err := database.SelectLedger(ledgerId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: ledger}
	}
}
