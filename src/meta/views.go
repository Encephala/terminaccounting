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

	motionSet  MotionSet
	commandSet CommandSet
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

	var normalMotions Trie[tea.Msg]
	normalMotions.Insert(Motion{"g", "d"}, SwitchViewMsg{ViewType: DETAILVIEWTYPE}) // [g]oto [d]etails

	MotionSet := MotionSet{Normal: normalMotions}

	return &ListView{
		ListModel: model,

		motionSet:  MotionSet,
		commandSet: CommandSet{},
	}
}

func (lv *ListView) Init() tea.Cmd {
	return nil
}

func (lv *ListView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		if message.ActualApp != message.TargetApp {
			panic(fmt.Sprintf("App %s received %#v for %s", message.ActualApp, message, message.TargetApp))
		}
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
	return &lv.motionSet
}

func (lv *ListView) CommandSet() *CommandSet {
	return &lv.commandSet
}

// A generic, placeholder view that just renders all entries on a ledger/journal/account in a lis.([]list.Item)t
type DetailView struct {
	listModel list.Model

	ModelId int

	motionSet  MotionSet
	commandSet CommandSet
}

func NewDetailView(app App, itemId int, itemName string) *DetailView {
	viewStyles := styles.NewDetailViewStyles(app.Colours().Foreground)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = fmt.Sprintf("%s: %s", app.Name(), itemName)
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	var normalMotions Trie[tea.Msg]
	normalMotions.Insert(Motion{"ctrl+o"}, SwitchViewMsg{ViewType: LISTVIEWTYPE})

	var commands Trie[tea.Msg]
	commands.Insert(Motion{"e"}, SwitchViewMsg{ViewType: UPDATEVIEWTYPE})

	return &DetailView{
		listModel: model,

		ModelId: itemId,

		motionSet:  MotionSet{Normal: normalMotions},
		commandSet: CommandSet{Commands: commands},
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return nil
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		if message.ActualApp != message.TargetApp {
			panic(fmt.Sprintf("App %s received %#v for %s", message.ActualApp, message, message.TargetApp))
		}

		if message.Model != "EntryRow" {
			panic(fmt.Sprintf("Setting detail view items, but got %q rather than EntryRow", message.Model))
		}
		dv.listModel.SetItems(message.Data.([]list.Item))

		return dv, nil

	case NavigateMsg:
		keyMsg := NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.listModel, cmd = dv.listModel.Update(keyMsg)

		return dv, cmd

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
	return &dv.motionSet
}

func (dv *DetailView) CommandSet() *CommandSet {
	return &dv.commandSet
}
