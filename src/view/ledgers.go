package view

import (
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"local/bubbles/itempicker"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	NAMEINPUT activeInput = iota
	TYPEINPUT
	NOTEINPUT
)

type createOrUpdateView interface {
	meta.View

	title() string

	getNameInput() *textinput.Model
	getTypeInput() *itempicker.Model
	getNoteInput() *textarea.Model

	getActiveInput() *activeInput

	getColours() styles.AppColours
}

type LedgersCreateView struct {
	NameInput   textinput.Model
	TypeInput   itempicker.Model
	NoteInput   textarea.Model
	activeInput activeInput

	colours styles.AppColours
}

func NewLedgersCreateView(colours styles.AppColours) *LedgersCreateView {
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

	result := &LedgersCreateView{
		NameInput:   nameInput,
		TypeInput:   typeInput,
		NoteInput:   noteInput,
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
	return &cv.NameInput
}
func (cv *LedgersCreateView) getTypeInput() *itempicker.Model {
	return &cv.TypeInput
}
func (cv *LedgersCreateView) getNoteInput() *textarea.Model {
	return &cv.NoteInput
}
func (cv *LedgersCreateView) getActiveInput() *activeInput {
	return &cv.activeInput
}
func (cv *LedgersCreateView) getColours() styles.AppColours {
	return cv.colours
}

func (cv *LedgersCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message.(type) {
	case meta.CommitCreateMsg:
		ledgerName := cv.NameInput.Value()
		ledgerType := cv.TypeInput.Value().(database.LedgerType)
		ledgerNotes := cv.NoteInput.Value()

		newLedger := database.Ledger{
			Name:  ledgerName,
			Type:  ledgerType,
			Notes: strings.Split(ledgerNotes, "\n"),
		}

		id, err := newLedger.Insert()

		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		// m.currentView = view.NewLedgersUpdateView(id, m.Colours())
		// TODO: Add a vimesque message to inform the user of successful creation (when vimesque messages are implemented)
		// Or maybe this should just switch to the list view or the detail view? Idk

		return cv, meta.MessageCmd(meta.SwitchViewMsg{
			ViewType: meta.UPDATEVIEWTYPE,
			Data:     id,
		})

	default:
		return createUpdateViewUpdate(cv, message)
	}
}

func (cv *LedgersCreateView) View() string {
	return createUpdateViewView(cv)
}

func (cv *LedgersCreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{Normal: normalMotions}
}

func (cv *LedgersCreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitCreateMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

type LedgersUpdateView struct {
	NameInput   textinput.Model
	TypeInput   itempicker.Model
	NoteInput   textarea.Model
	activeInput activeInput

	ModelId       int
	startingValue database.Ledger

	colours styles.AppColours
}

func NewLedgersUpdateView(modelId int, colours styles.AppColours) *LedgersUpdateView {
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
		NameInput:   nameInput,
		TypeInput:   typeInput,
		NoteInput:   noteInput,
		activeInput: NAMEINPUT,

		ModelId: modelId,

		colours: colours,
	}
}

func (uv *LedgersUpdateView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(uv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(uv.CommandSet())))

	cmds = append(cmds, database.MakeLoadLedgersDetailCmd(uv.ModelId))

	return tea.Batch(cmds...)
}

func (uv *LedgersUpdateView) title() string {
	return fmt.Sprintf("Update Ledger: %s", uv.NameInput.Value())
}
func (uv *LedgersUpdateView) getNameInput() *textinput.Model {
	return &uv.NameInput
}
func (uv *LedgersUpdateView) getTypeInput() *itempicker.Model {
	return &uv.TypeInput
}
func (uv *LedgersUpdateView) getNoteInput() *textarea.Model {
	return &uv.NoteInput
}
func (uv *LedgersUpdateView) getActiveInput() *activeInput {
	return &uv.activeInput
}
func (uv *LedgersUpdateView) getColours() styles.AppColours {
	return uv.colours
}

func (uv *LedgersUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		ledger := message.Data.(database.Ledger)

		uv.startingValue = ledger

		uv.NameInput.SetValue(ledger.Name)
		uv.TypeInput.SetValue(ledger.Type)
		uv.NoteInput.SetValue(strings.Join(ledger.Notes, "\n"))

		return uv, nil

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case NAMEINPUT:
			uv.NameInput.SetValue(uv.startingValue.Name)
		case TYPEINPUT:
			uv.TypeInput.SetValue(uv.startingValue.Type)
		case NOTEINPUT:
			uv.NoteInput.SetValue(strings.Join(uv.startingValue.Notes, "\n"))
		}

		return uv, nil

	case meta.CommitUpdateMsg:
		currentValues := database.Ledger{
			Id:    uv.ModelId,
			Name:  uv.NameInput.Value(),
			Type:  uv.TypeInput.Value().(database.LedgerType),
			Notes: strings.Split(uv.NoteInput.Value(), "\n"),
		}

		currentValues.Update()

		return uv, nil

	default:
		return createUpdateViewUpdate(uv, message)
	}

}

func (uv *LedgersUpdateView) View() string {
	return createUpdateViewView(uv)
}

func (uv *LedgersUpdateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	return &meta.MotionSet{Normal: normalMotions}
}

func (uv *LedgersUpdateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitUpdateMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}

// The common parts of the Update function for a create- and update view
func createUpdateViewUpdate(view createOrUpdateView, message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.SwitchFocusMsg:
		// If currently on a textinput, blur it
		// Shouldn't matter too much because we only send the update to the right input, but FWIW
		// Note from later me: might actually delete this as an implicit assertion that only the right input
		// gets the update message.
		switch *view.getActiveInput() {
		case NAMEINPUT:
			view.getNameInput().Blur()
		case NOTEINPUT:
			view.getNoteInput().Blur()
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
			view.getNoteInput().Focus()
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
			*view.getNoteInput(), cmd = view.getNoteInput().Update(message)

		default:
			panic(fmt.Sprintf("Updating create view but active input was %d", *view.getActiveInput()))
		}

		return view, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

// The common parts of the View function for a create- and update view
func createUpdateViewView(view createOrUpdateView) string {
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
	view.getNoteInput().SetWidth(inputWidth)

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
		style.Render(view.getNoteInput().View()),
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

type LedgersDeleteView struct {
	ModelId int
	model   database.Ledger

	colours styles.AppColours
}

func NewLedgersDeleteView(modelId int, colours styles.AppColours) *LedgersDeleteView {
	return &LedgersDeleteView{
		ModelId: modelId,

		colours: colours,
	}
}

func (dv *LedgersDeleteView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(dv.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(dv.CommandSet())))

	cmds = append(cmds, database.MakeLoadLedgersDetailCmd(dv.ModelId))

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

	case meta.CommitDeleteMsg:
		err := database.DeleteLedger(dv.ModelId)

		// TODO: Add a vimesque message to inform user of successful deletion
		var cmds []tea.Cmd
		if err != nil {
			cmds = append(cmds, tea.Batch(meta.MessageCmd(err), meta.MessageCmd(meta.SwitchViewMsg{
				ViewType: meta.LISTVIEWTYPE,
			})))
		}

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *LedgersDeleteView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.colours.Background).Padding(0, 1)

	result.WriteString(fmt.Sprintf("  %s", titleStyle.Render(dv.title())))
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
		style.AlignHorizontal(lipgloss.Left).Render(strings.Join(dv.model.Notes, "\n")),
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

	normalMotions.Insert(meta.Motion{"g", "d"}, meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.ModelId})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *LedgersDeleteView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitDeleteMsg{})

	asCommandSet := meta.CommandSet(commands)
	return &asCommandSet
}
