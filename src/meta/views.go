package meta

import (
	"fmt"
	"terminaccounting/styles"
	"terminaccounting/vim"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewType int

const (
	ListViewType ViewType = iota
	DetailViewType
	CreateViewType
)

type View interface {
	tea.Model

	Type() ViewType

	MotionSet() *vim.MotionSet
}

type ListView struct {
	Model list.Model

	motionSet vim.MotionSet
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

	var normalMotions vim.Trie
	normalMotions.Insert(vim.Motion{"g", "d"}, vim.CompletedMotionMsg{Type: vim.SWITCHVIEW, Data: vim.DETAILVIEW}) // [g]oto [d]etails

	MotionSet := vim.MotionSet{Normal: normalMotions}

	return &ListView{
		Model: model,

		motionSet: MotionSet,
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
	}

	var cmd tea.Cmd
	lv.Model, cmd = lv.Model.Update(message)

	return lv, cmd
}

func (lv *ListView) View() string {
	return lv.Model.View()
}

func (lv *ListView) Type() ViewType {
	return ListViewType
}

func (lv *ListView) MotionSet() *vim.MotionSet {
	return &lv.motionSet
}

type DetailView struct {
	Model list.Model

	motionSet vim.MotionSet
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

	var normalMotions vim.Trie
	normalMotions.Insert(vim.Motion{"ctrl+o"}, vim.CompletedMotionMsg{Type: vim.SWITCHVIEW, Data: vim.LISTVIEW})

	return &DetailView{
		Model: model,

		motionSet: vim.MotionSet{Normal: normalMotions},
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
	}

	var cmd tea.Cmd
	dv.Model, cmd = dv.Model.Update(message)

	return dv, cmd
}

func (dv *DetailView) View() string {
	return dv.Model.View()
}

func (dv *DetailView) Type() ViewType {
	return DetailViewType
}

func (dv *DetailView) MotionSet() *vim.MotionSet {
	return &dv.motionSet
}
