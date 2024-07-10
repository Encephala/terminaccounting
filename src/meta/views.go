package meta

import (
	"fmt"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewType int

const (
	ListViewType ViewType = iota
	DetailViewType
)

type ListView struct {
	Model  list.Model
	Styles styles.ListViewStyles
	Title  string
}

func NewListView(title string, viewStyles styles.ListViewStyles) ListView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = title
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return ListView{
		Model:  model,
		Styles: viewStyles,
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
	Model list.Model

	Styles styles.DetailViewStyles
	Title  string
}

// You like that code reuse? Very DRY
func NewDetailView(title string, viewStyles styles.DetailViewStyles) DetailView {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	model := list.New([]list.Item{}, delegate, 20, 16)
	model.Title = title
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return DetailView{
		Model:  model,
		Styles: viewStyles,
		Title:  title,
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
