package meta

import (
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ListView struct {
	Model  list.Model
	Styles styles.ListViewStyles
	Title  string
}

func NewListView(title string, styles styles.ListViewStyles) ListView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = styles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = styles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = title
	model.Styles.Title = styles.Title
	model.SetShowHelp(false)

	return ListView{
		Model:  model,
		Styles: styles,
		Title:  title,
	}
}

func (lv *ListView) Init() tea.Cmd {
	return nil
}

func (lv *ListView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case DataLoadedMsg:
		lv.Model.SetItems(message.Items)
	}

	var cmd tea.Cmd
	lv.Model, cmd = lv.Model.Update(message)

	return lv, cmd
}

func (lv *ListView) View() string {
	return lv.Model.View()
}

type DetailView struct {
	Entries list.Model

	Styles styles.DetailViewStyles
	Title  string
}

func (dv *DetailView) Init() tea.Cmd {
	return nil
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return dv, nil
}

func (dv *DetailView) View() string {
	return ""
}
