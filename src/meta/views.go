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
	Model list.Model

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
		Model: model,

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
			panic(fmt.Sprintf("App %s received %T for %s", message.ActualApp, message, message.TargetApp))
		}
		lv.Model.SetItems(message.Items)

	case NavigateMsg:
		keyMsg := NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		lv.Model, cmd = lv.Model.Update(keyMsg)

		return lv, cmd

	case tea.WindowSizeMsg:
		// Explicitly break to avoid the panic
		break

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}

	var cmd tea.Cmd
	lv.Model, cmd = lv.Model.Update(message)

	return lv, cmd
}

func (lv *ListView) View() string {
	return lv.Model.View()
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

type DetailView struct {
	Model list.Model

	motionSet  MotionSet
	commandSet CommandSet
}

func NewDetailView(app App, itemName string) *DetailView {
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

	return &DetailView{
		Model: model,

		motionSet:  MotionSet{Normal: normalMotions},
		commandSet: CommandSet{},
	}
}

func (dv *DetailView) Init() tea.Cmd {
	return nil
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		if message.Model != "EntryRow" {
			panic(fmt.Sprintf("Setting detail view items, but got %q rather than EntryRow", message.Model))
		}
		dv.Model.SetItems(message.Items)

	case NavigateMsg:
		keyMsg := NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.Model, cmd = dv.Model.Update(keyMsg)

		return dv, cmd

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}

	var cmd tea.Cmd
	dv.Model, cmd = dv.Model.Update(message)

	return dv, cmd
}

func (dv *DetailView) View() string {
	return dv.Model.View()
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
