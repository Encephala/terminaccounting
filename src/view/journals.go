package view

import (
	"errors"
	"fmt"
	"local/bubbles/itempicker"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type JournalsDetailsView struct {
	listModel list.Model

	app meta.App

	modelId int

	journal database.Journal
}

func NewJournalsDetailsView(journal database.Journal, app meta.App) *JournalsDetailsView {
	viewStyles := meta.NewListViewStyles(app.Colours().Accent, app.Colours().Foreground)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = fmt.Sprintf("Journals detail view: %q", journal.Name)
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return &JournalsDetailsView{
		listModel: model,

		app: app,

		modelId: journal.Id,

		journal: journal,
	}
}

func (dv *JournalsDetailsView) Init() tea.Cmd {
	return database.MakeSelectEntriesByJournalCmd(dv.journal.Id)
}

func (dv *JournalsDetailsView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

func (dv *JournalsDetailsView) View() string {
	return dv.listModel.View()
}

func (dv *JournalsDetailsView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYMODEL: {},
	}
}

func (dv *JournalsDetailsView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"/"}, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}) // true -> yes search mode

	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())
	normalMotions.Insert(meta.Motion{"g", "c"}, meta.SwitchViewMsg{ViewType: meta.CREATEVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})
	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *JournalsDetailsView) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (dv *JournalsDetailsView) Reload() View {
	return NewJournalsDetailsView(dv.journal, dv.app)
}

// Contrary to the generic list view, going to detail view here jumps to an entries detail view
func (dv *JournalsDetailsView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		item := dv.listModel.SelectedItem()

		if item == nil {
			return errors.New("no item to goto detail view of")
		}

		entriesAppType := meta.ENTRIESAPP
		return meta.SwitchViewMsg{App: &entriesAppType, ViewType: meta.DETAILVIEWTYPE, Data: item}
	}
}

type JournalsCreateView struct {
	nameInput  textinput.Model
	typeInput  itempicker.Model
	notesInput textarea.Model
	activeInput

	colours meta.AppColours
}

func NewJournalsCreateView(colours meta.AppColours) *JournalsCreateView {
	journalTypes := []itempicker.Item{
		database.INCOMEJOURNAL,
		database.EXPENSEJOURNAL,
		database.CASHFLOWJOURNAL,
		database.GENERALJOURNAL,
	}

	nameInput := textinput.New()
	nameInput.Focus()
	nameInput.Cursor.SetMode(cursor.CursorStatic)
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	return &JournalsCreateView{
		nameInput:   nameInput,
		typeInput:   itempicker.New(journalTypes),
		notesInput:  noteInput,
		activeInput: NAMEINPUT,

		colours: colours,
	}
}

func (cv *JournalsCreateView) Init() tea.Cmd {
	return nil
}

func (cv *JournalsCreateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.CommitMsg:
		name := cv.nameInput.Value()
		journalType := cv.typeInput.Value().(database.JournalType)
		notes := cv.notesInput.Value()

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
			"Successfully created Journal %q", cv.nameInput.Value(),
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
			cv.activeInput.previous(3)

		case meta.NEXT:
			cv.activeInput.next(3)
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

func (cv *JournalsCreateView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(cv.colours.Background).Padding(0, 1).MarginLeft(2)

	result.WriteString(titleStyle.Render("Creating new Journal"))
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

func (cv *JournalsCreateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{}
}

func (cv *JournalsCreateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	return meta.MotionSet{Normal: normalMotions}
}

func (cv *JournalsCreateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (cv *JournalsCreateView) Reload() View {
	return NewJournalsCreateView(cv.colours)
}

type JournalsUpdateView struct {
	nameInput  textinput.Model
	typeInput  itempicker.Model
	notesInput textarea.Model
	activeInput

	modelId       int
	startingValue database.Journal

	colours meta.AppColours
}

func NewJournalsUpdateView(modelId int, colours meta.AppColours) *JournalsUpdateView {
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
	noteInput := textarea.New()
	noteInput.Cursor.SetMode(cursor.CursorStatic)

	return &JournalsUpdateView{
		nameInput:   nameInput,
		typeInput:   typeInput,
		notesInput:  noteInput,
		activeInput: NAMEINPUT,

		modelId: modelId,

		colours: colours,
	}
}

func (uv *JournalsUpdateView) Init() tea.Cmd {
	return database.MakeLoadJournalsDetailCmd(uv.modelId)
}

func (uv *JournalsUpdateView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		// Loaded the current(/"starting") properties of the ledger being edited
		journal := message.Data.(database.Journal)

		uv.startingValue = journal

		uv.nameInput.SetValue(journal.Name)
		err := uv.typeInput.SetValue(journal.Type)
		uv.notesInput.SetValue(journal.Notes.Collapse())

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
		journal := database.Journal{
			Id:    uv.modelId,
			Name:  uv.nameInput.Value(),
			Type:  uv.typeInput.Value().(database.JournalType),
			Notes: meta.CompileNotes(uv.notesInput.Value()),
		}

		err := journal.Update()
		if err != nil {
			return uv, meta.MessageCmd(err)
		}

		return uv, meta.MessageCmd(meta.NotificationMessageMsg{Message: fmt.Sprintf(
			"Successfully updated Journal %q", uv.nameInput.Value(),
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
			uv.activeInput.previous(3)

		case meta.NEXT:
			uv.activeInput.next(3)
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

func (uv *JournalsUpdateView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(uv.colours.Background).Padding(0, 1).Margin(0, 0, 0, 2)

	result.WriteString(titleStyle.Render(fmt.Sprintf("Update Journal: %s", uv.startingValue.Name)))
	result.WriteString("\n\n")

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Center)
	rightStyle := style.Margin(0, 0, 0, 1)

	const inputWidth = 26
	uv.nameInput.Width = inputWidth - 2 // -2 because of the prompt
	uv.notesInput.SetWidth(inputWidth)

	// TODO: Render active input with a different colour
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

func (uv *JournalsUpdateView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.JOURNALMODEL: {},
	}
}

func (uv *JournalsUpdateView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"tab"}, meta.SwitchFocusMsg{Direction: meta.NEXT})
	normalMotions.Insert(meta.Motion{"shift+tab"}, meta.SwitchFocusMsg{Direction: meta.PREVIOUS})

	normalMotions.Insert(meta.Motion{"u"}, meta.ResetInputFieldMsg{})

	normalMotions.Insert(meta.Motion{"g", "d"}, uv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (uv *JournalsUpdateView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (uv *JournalsUpdateView) Reload() View {
	return NewJournalsUpdateView(uv.modelId, uv.colours)
}

func (uv *JournalsUpdateView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: uv.startingValue}
	}
}

type JournalsDeleteView struct {
	modelId int
	model   database.Journal

	colours meta.AppColours
}

func NewJournalsDeleteView(modelId int, colours meta.AppColours) *JournalsDeleteView {
	return &JournalsDeleteView{
		modelId: modelId,

		colours: colours,
	}
}

func (dv *JournalsDeleteView) Init() tea.Cmd {
	return database.MakeLoadJournalsDetailCmd(dv.modelId)
}

func (dv *JournalsDeleteView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

		cmds = append(cmds, meta.MessageCmd(meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE}))

		return dv, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		// TODO

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *JournalsDeleteView) View() string {
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

func (dv *JournalsDeleteView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.JOURNALMODEL: {},
	}
}

func (dv *JournalsDeleteView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *JournalsDeleteView) CommandSet() meta.CommandSet {
	var commands meta.Trie[tea.Msg]

	commands.Insert(meta.Command(strings.Split("write", "")), meta.CommitMsg{})

	return meta.CommandSet(commands)
}

func (dv *JournalsDeleteView) Reload() View {
	return NewJournalsDeleteView(dv.modelId, dv.colours)
}

func (dv *JournalsDeleteView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: dv.model}
	}
}
