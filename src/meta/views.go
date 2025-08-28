package meta

import (
	"fmt"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type View interface {
	tea.Model

	MotionSet() *MotionSet
	CommandSet() *CommandSet
}

type ListView struct {
	ListModel list.Model

	app App
}

func NewListView(app App) *ListView {
	viewStyles := styles.NewListViewStyles(app.Colours().Accent, app.Colours().Foreground)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = app.Name()
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return &ListView{
		ListModel: model,

		app: app,
	}
}

func (lv *ListView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, MessageCmd(UpdateViewMotionSetMsg(lv.app.CurrentMotionSet())))
	cmds = append(cmds, MessageCmd(UpdateViewCommandSetMsg(lv.app.CurrentCommandSet())))

	cmds = append(cmds, lv.app.MakeLoadListCmd())

	return tea.Batch(cmds...)
}

func (lv *ListView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		lv.ListModel.SetItems(message.Data.([]list.Item))

		return lv, nil

	case NavigateMsg:
		keyMsg := NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		lv.ListModel, cmd = lv.ListModel.Update(keyMsg)

		return lv, cmd

	// Returning to prevent panic
	// Required because other views do accept these messages
	case tea.WindowSizeMsg:
		// TODO Maybe rescale the rendering of the inputs by the window size or something
		return lv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (lv *ListView) View() string {
	return lv.ListModel.View()
}

func (lv *ListView) Type() ViewType {
	return LISTVIEWTYPE
}

func (lv *ListView) MotionSet() *MotionSet {
	var normalMotions Trie[tea.Msg]

	normalMotions.Insert(Motion{"g", "d"}, SwitchViewMsg{
		ViewType: DETAILVIEWTYPE,
	}) // [g]oto [d]etails

	return &MotionSet{Normal: normalMotions}
}

func (lv *ListView) CommandSet() *CommandSet {
	return &CommandSet{}
}

// A generic, placeholder view that just renders all entries on a ledger/journal/account in a list.
type DetailView struct {
	listModel list.Model

	app App

	ModelId int
}

func NewDetailView(app App, itemId int) *DetailView {
	viewStyles := styles.NewDetailViewStyles(app.Colours().Foreground)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	// TODO: Change placeholder to, ykno, the name of the item being detail-view'd
	model.Title = fmt.Sprintf("%s: %s", app.Name(), "PLACEHOLDER")
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return &DetailView{
		listModel: model,

		app: app,

		ModelId: itemId,
	}
}

func (dv *DetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, MessageCmd(UpdateViewMotionSetMsg(dv.app.CurrentMotionSet())))
	cmds = append(cmds, MessageCmd(UpdateViewCommandSetMsg(dv.app.CurrentCommandSet())))

	// TODO: Also show the model metadata and not just the rows?
	cmds = append(cmds, dv.app.MakeLoadRowsCmd())

	return tea.Batch(cmds...)
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		if message.Model != ENTRYROW {
			panic(fmt.Sprintf("Expected an EntryRow, but got %v", message.Model))
		}

		dv.listModel.SetItems(message.Data.([]list.Item))

		return dv, nil

	case NavigateMsg:
		keyMsg := NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.listModel, cmd = dv.listModel.Update(keyMsg)

		return dv, cmd

	case tea.WindowSizeMsg:
		// TODO maybe?
		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *DetailView) View() string {
	return dv.listModel.View()
}

func (dv *DetailView) Type() ViewType {
	return DETAILVIEWTYPE
}

func (dv *DetailView) MotionSet() *MotionSet {
	var normalMotions Trie[tea.Msg]

	normalMotions.Insert(Motion{"g", "l"}, SwitchViewMsg{ViewType: LISTVIEWTYPE})
	normalMotions.Insert(Motion{"g", "x"}, SwitchViewMsg{ViewType: DELETEVIEWTYPE, Data: dv.ModelId})

	normalMotions.Insert(Motion{"g", "e"}, SwitchViewMsg{ViewType: UPDATEVIEWTYPE, Data: dv.ModelId})

	return &MotionSet{
		Normal: normalMotions,
	}
}

func (dv *DetailView) CommandSet() *CommandSet {
	return &CommandSet{}
}
