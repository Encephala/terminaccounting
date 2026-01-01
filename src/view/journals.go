package view

import (
	"errors"
	"fmt"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type journalsDetailsView struct {
	listModel list.Model

	app meta.App

	modelId int

	journal database.Journal
}

func NewJournalsDetailsView(journal database.Journal, app meta.App) *journalsDetailsView {
	viewStyles := meta.NewListViewStyles(app.Colour())

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	// List dimensions will be updated according to tea.WindowSizeMsg
	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = fmt.Sprintf("Journals detail view: %q", journal.Name)
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return &journalsDetailsView{
		listModel: model,

		app: app,

		modelId: journal.Id,

		journal: journal,
	}
}

func (dv *journalsDetailsView) Init() tea.Cmd {
	return database.MakeSelectEntriesByJournalCmd(dv.journal.Id)
}

func (dv *journalsDetailsView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		entries := message.Data.([]database.Entry)

		asItems := make([]list.Item, len(entries))
		for i, row := range entries {
			asItems[i] = row
		}

		dv.listModel.SetItems(asItems)

		return dv, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.listModel, cmd = dv.listModel.Update(keyMsg)

		return dv, cmd

	// Returning to prevent panic
	// Required because other views do accept these messages
	case tea.WindowSizeMsg:
		// TODO Maybe rescale the rendering of the inputs by the window size or something
		return dv, nil

	case meta.UpdateSearchMsg:
		if message.Query == "" {
			dv.listModel.ResetFilter()
		} else {
			dv.listModel.SetFilterText(message.Query)
		}

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *journalsDetailsView) View() string {
	return dv.listModel.View()
}

func (dv *journalsDetailsView) AllowsInsertMode() bool {
	return false
}

func (dv *journalsDetailsView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL: {},
	}
}

func (dv *journalsDetailsView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"/"}, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}) // true -> yes search mode

	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())
	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchAppViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *journalsDetailsView) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (dv *journalsDetailsView) Reload() View {
	return NewJournalsDetailsView(dv.journal, dv.app)
}

// Contrary to the generic list view, going to detail view here jumps to an entries detail view
func (dv *journalsDetailsView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		item := dv.listModel.SelectedItem()

		if item == nil {
			return errors.New("no item to goto detail view of")
		}

		entriesAppType := meta.ENTRIESAPP
		return meta.SwitchAppViewMsg{App: &entriesAppType, ViewType: meta.DETAILVIEWTYPE, Data: item}
	}
}

type journalsCreateView struct {
	inputManager *inputManager

	colour lipgloss.Color
}

func NewJournalsCreateView() *journalsCreateView {
	journalTypes := []itempicker.Item{
		database.INCOMEJOURNAL,
		database.EXPENSEJOURNAL,
		database.CASHFLOWJOURNAL,
		database.GENERALJOURNAL,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	// -2 because of the prompt, -1 because of the cursor
	nameInput.Cursor.SetMode(cursor.CursorStatic)

	typeInput := itempicker.New(journalTypes)

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)

	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.JOURNALSCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	inputs := []any{nameInput, typeInput, notesInput}
	names := []string{"Name", "Type", "Notes"}

	return &journalsCreateView{
		inputManager: newInputManager(inputs, names),

		colour: meta.JOURNALSCOLOUR,
	}
}

func (cv *journalsCreateView) Init() tea.Cmd {
	return nil
}

func (cv *journalsCreateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message.(type) {
	case meta.CommitMsg:
		name := cv.inputManager.inputs[0].value().(string)
		journalType := cv.inputManager.inputs[1].value().(database.JournalType)
		notes := cv.inputManager.inputs[2].value().(string)

		newJournal := database.Journal{
			Name:  name,
			Type:  journalType,
			Notes: meta.CompileNotes(notes),
		}

		id, err := newJournal.Insert()
		if err != nil {
			return cv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully created Journal %q", name,
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

func (cv *journalsCreateView) View() string {
	return genericMutateViewView(cv)
}

func (cv *journalsCreateView) AllowsInsertMode() bool {
	return true
}

func (cv *journalsCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *journalsCreateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func (cv *journalsCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *journalsCreateView) Reload() View {
	return NewJournalsCreateView()
}

func (cv *journalsCreateView) getInputManager() *inputManager {
	return cv.inputManager
}

func (cv *journalsCreateView) title() string {
	return "Creating new journal"
}

func (cv *journalsCreateView) getColour() lipgloss.Color {
	return cv.colour
}

type journalsUpdateView struct {
	inputManager *inputManager

	modelId       int
	startingValue database.Journal

	colour lipgloss.Color
}

func NewJournalsUpdateView(modelId int) *journalsUpdateView {
	types := []itempicker.Item{
		database.INCOMEJOURNAL,
		database.EXPENSEJOURNAL,
		database.CASHFLOWJOURNAL,
		database.GENERALJOURNAL,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	typeInput := itempicker.New(types)

	notesInput := textarea.New()
	notesInput.Cursor.SetMode(cursor.CursorStatic)
	notesFocusStyle := lipgloss.NewStyle().Foreground(meta.JOURNALSCOLOUR)
	notesInput.FocusedStyle.Prompt = notesFocusStyle
	notesInput.FocusedStyle.Text = notesFocusStyle
	notesInput.FocusedStyle.CursorLine = notesFocusStyle
	notesInput.FocusedStyle.LineNumber = notesFocusStyle

	inputs := []any{nameInput, typeInput, notesInput}
	names := []string{"Name", "Type", "Notes"}

	return &journalsUpdateView{
		inputManager: newInputManager(inputs, names),

		modelId: modelId,

		colour: meta.JOURNALSCOLOUR,
	}
}

func (uv *journalsUpdateView) Init() tea.Cmd {
	return database.MakeLoadJournalsDetailCmd(uv.modelId)
}

func (uv *journalsUpdateView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		journal := message.Data.(database.Journal)

		uv.startingValue = journal

		uv.inputManager.inputs[0].setValue(journal.Name)
		err := uv.inputManager.inputs[1].setValue(journal.Type)
		uv.inputManager.inputs[2].setValue(journal.Notes.Collapse())

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
		default:
			panic(fmt.Sprintf("unexpected activeInput: %d", uv.inputManager.activeInput))
		}

		err := uv.inputManager.inputs[uv.inputManager.activeInput].setValue(startingValue)

		return uv, meta.MessageCmd(err)

	case meta.CommitMsg:
		inputs := uv.inputManager.inputs
		name := inputs[0].value().(string)
		journalType := inputs[1].value().(database.JournalType)
		notes := meta.CompileNotes(inputs[2].value().(string))

		journal := database.Journal{
			Id:    uv.modelId,
			Name:  name,
			Type:  journalType,
			Notes: notes,
		}

		err := journal.Update()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Journal %q", name,
		)})

	case meta.NavigateMsg:
		return uv, nil
	}

	return genericMutateViewUpdate(uv, message)
}

func (uv *journalsUpdateView) View() string {
	return genericMutateViewView(uv)
}

func (uv *journalsUpdateView) AllowsInsertMode() bool {
	return true
}

func (uv *journalsUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.JOURNALMODEL: {},
	}
}

func (uv *journalsUpdateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (uv *journalsUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *journalsUpdateView) Reload() View {
	return NewJournalsUpdateView(uv.modelId)
}

func (uv *journalsUpdateView) getInputManager() *inputManager {
	return uv.inputManager
}

func (cv *journalsUpdateView) title() string {
	return "Creating new journal"
}

func (cv *journalsUpdateView) getColour() lipgloss.Color {
	return cv.colour
}

func (uv *journalsUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

type journalsDeleteView struct {
	modelId int
	model   database.Journal

	colour lipgloss.Color
}

func NewJournalsDeleteView(modelId int) *journalsDeleteView {
	return &journalsDeleteView{
		modelId: modelId,

		colour: meta.JOURNALSCOLOUR,
	}
}

func (dv *journalsDeleteView) Init() tea.Cmd {
	return database.MakeLoadJournalsDetailCmd(dv.modelId)
}

func (dv *journalsDeleteView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		dv.model = message.Data.(database.Journal)

		return dv, nil

	case meta.CommitMsg:
		err := database.DeleteJournal(dv.modelId)
		if err != nil {
			return dv, meta.MessageCmd(err)
		}

		var cmds []tea.Cmd

		cmds = append(cmds, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully deleted Journal %q", dv.model.Name,
		)}))

		cmds = append(cmds, meta.MessageCmd(meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *journalsDeleteView) View() string {
	return genericDeleteViewView(dv)
}

func (dv *journalsDeleteView) AllowsInsertMode() bool {
	return false
}

func (dv *journalsDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.JOURNALMODEL: {},
	}
}

func (dv *journalsDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *journalsDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *journalsDeleteView) Reload() View {
	return NewJournalsDeleteView(dv.modelId)
}

func (dv *journalsDeleteView) title() string {
	return fmt.Sprintf("Delete journal: %s", dv.model.String())
}

func (dv *journalsDeleteView) inputValues() []string {
	return []string{dv.model.Name, dv.model.Type.String(), dv.model.Notes.Collapse()}
}

func (dv *journalsDeleteView) inputNames() []string {
	return []string{"Name", "Type", "Notes"}
}

func (dv *journalsDeleteView) getColour() lipgloss.Color {
	return dv.colour
}

func (dv *journalsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchAppViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
