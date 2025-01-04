package ledgers

import (
	"fmt"
	"strings"
	"terminaccounting/meta"
	"terminaccounting/styles"

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

type activeInput int

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

type CreateView struct {
	nameInput   textinput.Model
	typeInput   itempicker.Model
	noteInput   textarea.Model
	activeInput activeInput

	colours styles.AppColours
}

func NewCreateView(colours styles.AppColours) *CreateView {
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
		activeInput: NAMEINPUT,

		colours: colours,
	}

	return result
}

func (cv *CreateView) Init() tea.Cmd {
	return nil
}

func (cv *CreateView) title() string {
	return "Create new Ledger"
}
func (cv *CreateView) getNameInput() *textinput.Model {
	return &cv.nameInput
}
func (cv *CreateView) getTypeInput() *itempicker.Model {
	return &cv.typeInput
}
func (cv *CreateView) getNoteInput() *textarea.Model {
	return &cv.noteInput
}
func (cv *CreateView) getActiveInput() *activeInput {
	return &cv.activeInput
}
func (cv *CreateView) getColours() styles.AppColours {
	return cv.colours
}
func (cv *CreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return createUpdateViewUpdate(cv, message)
}

func (cv *CreateView) View() string {
	return createUpdateViewView(cv)
}

func (cv *CreateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"ctrl+o"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return &meta.MotionSet{Normal: normalMotions}
}

func (cv *CreateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitCreateMsg{})

	return &meta.CommandSet{Commands: commands}
}

type UpdateView struct {
	nameInput   textinput.Model
	typeInput   itempicker.Model
	noteInput   textarea.Model
	activeInput activeInput

	modelId       int
	startingValue Ledger

	colours styles.AppColours
}

func NewUpdateView(modelId int, colours styles.AppColours) *UpdateView {
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

	return &UpdateView{
		nameInput:   nameInput,
		typeInput:   typeInput,
		noteInput:   noteInput,
		activeInput: NAMEINPUT,

		modelId: modelId,

		colours: colours,
	}
}

func (uv *UpdateView) Init() tea.Cmd {
	return nil
}

func (uv *UpdateView) title() string {
	return fmt.Sprintf("Update Ledger: %s", uv.nameInput.Value())
}
func (uv *UpdateView) getNameInput() *textinput.Model {
	return &uv.nameInput
}
func (uv *UpdateView) getTypeInput() *itempicker.Model {
	return &uv.typeInput
}
func (uv *UpdateView) getNoteInput() *textarea.Model {
	return &uv.noteInput
}
func (uv *UpdateView) getActiveInput() *activeInput {
	return &uv.activeInput
}
func (uv *UpdateView) getColours() styles.AppColours {
	return uv.colours
}

func (uv *UpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		ledger := message.Data.(Ledger)

		uv.startingValue = ledger

		uv.nameInput.SetValue(ledger.Name)
		uv.typeInput.SetValue(ledger.Type)
		uv.noteInput.SetValue(strings.Join(ledger.Notes, "\n"))

		return uv, nil

	case meta.ResetInputFieldMsg:
		switch uv.activeInput {
		case NAMEINPUT:
			uv.nameInput.SetValue(uv.startingValue.Name)
		case TYPEINPUT:
			uv.typeInput.SetValue(uv.startingValue.Type)
		case NOTEINPUT:
			uv.noteInput.SetValue(strings.Join(uv.startingValue.Notes, "\n"))
		}

		return uv, nil
	}

	return createUpdateViewUpdate(uv, message)
}

func (uv *UpdateView) View() string {
	return createUpdateViewView(uv)
}

func (uv *UpdateView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"ctrl+o"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	return &meta.MotionSet{Normal: normalMotions}
}

func (uv *UpdateView) CommandSet() *meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command{"w"}, meta.CommitUpdateMsg{})

	return &meta.CommandSet{Commands: commands}
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

type DeleteView struct {
	model Ledger

	colours styles.AppColours
}

func NewDeleteView(colours styles.AppColours) *DeleteView {
	return &DeleteView{
		colours: colours,
	}
}

func (dv *DeleteView) Init() tea.Cmd {
	return nil
}

func (dv *DeleteView) title() string {
	return fmt.Sprintf("Delete Ledger: %s", dv.model.Name)
}

func (dv *DeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(Ledger)

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *DeleteView) View() string {
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
		style.Render(strings.Join(dv.model.Notes, "\n")),
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

func (dv *DeleteView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"ctrl+o"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *DeleteView) CommandSet() *meta.CommandSet {
	return &meta.CommandSet{}
}
