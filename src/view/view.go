package view

import (
	"fmt"
	"io"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type activeInput int

type View interface {
	tea.Model

	MotionSet() *meta.MotionSet
	CommandSet() *meta.CommandSet
}

type ListView struct {
	ListModel list.Model

	app meta.App
}

func NewListView(app meta.App) *ListView {
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

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(lv.app.CurrentMotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(lv.app.CurrentCommandSet())))

	cmds = append(cmds, lv.app.MakeLoadListCmd())

	return tea.Batch(cmds...)
}

func (lv *ListView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		lv.ListModel.SetItems(message.Data.([]list.Item))

		return lv, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

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

func (lv *ListView) Type() meta.ViewType {
	return meta.LISTVIEWTYPE
}

func (lv *ListView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "d"}, meta.SwitchViewMsg{
		ViewType: meta.DETAILVIEWTYPE,
	}) // [g]oto [d]etails

	return &meta.MotionSet{Normal: normalMotions}
}

func (lv *ListView) CommandSet() *meta.CommandSet {
	return &meta.CommandSet{}
}

// A generic, placeholder view that just renders all entries on a ledger/journal/account in a list.
type DetailView struct {
	listModel list.Model

	app meta.App

	ModelId int
}

type entryRowDelegate struct {
	style styles.DetailViewStyles
}

func (erd entryRowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	er := item.(database.EntryRow)

	var debit, credit database.CurrencyValue
	if er.Value > 0 {
		debit = er.Value
	} else {
		credit = er.Value
	}

	line := fmt.Sprintf("%s | %-20s | %-20s | %s | %s", er.Date, "LedgerName", "AccountName", debit, credit)
	if index == m.Index() {
		fmt.Fprint(w, erd.style.ItemSelected.Render(line))
	} else {
		fmt.Fprint(w, erd.style.Item.Render(line))
	}
}

func (erd entryRowDelegate) Height() int { return 1 }

func (erd entryRowDelegate) Spacing() int { return 0 }

func (erd entryRowDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func NewDetailView(app meta.App, itemId int) *DetailView {
	viewStyles := styles.NewDetailViewStyles(app.Colours())

	delegate := entryRowDelegate{
		style: viewStyles,
	}

	model := list.New([]list.Item{}, delegate, 20, 16)
	// TODO: Change PLACEHOLDER to, ykno, the name of the item being detail-view'd
	model.Title = fmt.Sprintf("%s: %s", app.Name(), "PLACEHOLDER")
	model.Styles.Title = viewStyles.Title
	// TODO: Make this scale when outer model scales and stuff
	// Think that's the WindowResizeMsg or smth, that should be handled
	model.SetWidth(100)
	model.SetShowHelp(false)

	return &DetailView{
		listModel: model,

		app: app,

		ModelId: itemId,
	}
}

func (dv *DetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(dv.app.CurrentMotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(dv.app.CurrentCommandSet())))

	// TODO: Also show the model metadata and not just the rows?
	cmds = append(cmds, dv.app.MakeLoadRowsCmd())

	return tea.Batch(cmds...)
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		if message.Model != meta.ENTRYROW {
			panic(fmt.Sprintf("Expected an EntryRow, but got %v", message.Model))
		}

		dv.listModel.SetItems(message.Data.([]list.Item))

		return dv, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

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

func (dv *DetailView) Type() meta.ViewType {
	return meta.DETAILVIEWTYPE
}

func (dv *DetailView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.ModelId})

	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.ModelId})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *DetailView) CommandSet() *meta.CommandSet {
	return &meta.CommandSet{}
}
