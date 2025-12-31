package view

import (
	"errors"
	"fmt"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ListView struct {
	listModel list.Model

	app meta.App
}

func NewListView(app meta.App) *ListView {
	viewStyles := meta.NewListViewStyles(app.Colour())

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

	// List dimensions will be updated according to tea.WindowSizeMsg
	model := list.New([]list.Item{}, delegate, 80, 16)
	model.Title = app.Name()
	model.Styles.Title = viewStyles.Title
	model.SetShowHelp(false)

	return &ListView{
		listModel: model,

		app: app,
	}
}

func (lv *ListView) Init() tea.Cmd {
	return lv.app.MakeLoadListCmd()
}

func (lv *ListView) Update(message tea.Msg) (View, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		lv.listModel.SetItems(message.Data.([]list.Item))

		return lv, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		lv.listModel, cmd = lv.listModel.Update(keyMsg)

		return lv, cmd

	// Returning to prevent panic
	// Required because other views do accept these messages
	case tea.WindowSizeMsg:
		// -2 because of horizontal padding
		lv.listModel.SetWidth(message.Width - 2)

		// -1 to leave some bottom padding
		lv.listModel.SetHeight(message.Height - 1)

		return lv, nil

	case meta.UpdateSearchMsg:
		if message.Query == "" {
			lv.listModel.ResetFilter()
		} else {
			lv.listModel.SetFilterText(message.Query)
		}

		return lv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (lv *ListView) View() string {
	return lv.listModel.View()
}

func (lv *ListView) AllowsInsertMode() bool {
	return false
}

func (lv *ListView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNTMODEL: {},
		meta.LEDGERMODEL:  {},
		meta.ENTRYMODEL:   {},
		meta.JOURNALMODEL: {},
	}
}

func (lv *ListView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"/"}, meta.SwitchModeMsg{InputMode: meta.COMMANDMODE, Data: true}) // true -> yes search mode

	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	normalMotions.Insert(meta.Motion{"g", "d"}, lv.makeGoToDetailViewCmd()) // [g]oto [d]etails
	normalMotions.Insert(meta.Motion{"g", "c"}, meta.SwitchViewMsg{
		ViewType: meta.CREATEVIEWTYPE,
	}) // [g]oto [c]reate view

	return meta.MotionSet{Normal: normalMotions}
}

func (lv *ListView) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (lv *ListView) Reload() View {
	return NewListView(lv.app)
}

func (lv *ListView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		item := lv.listModel.SelectedItem()

		if item == nil {
			return errors.New("no item to goto detail view of")
		}

		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: item}
	}
}
