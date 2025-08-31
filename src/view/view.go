package view

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type activeInput int

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
	normalMotions.Insert(meta.Motion{"g", "c"}, meta.SwitchViewMsg{
		ViewType: meta.CREATEVIEWTYPE,
	}) // [g]oto [c]reate view

	return &meta.MotionSet{Normal: normalMotions}
}

func (lv *ListView) CommandSet() *meta.CommandSet {
	return &meta.CommandSet{}
}

// A generic, placeholder(?) view that just renders all entries on a ledger/journal/account in a list.
type DetailView struct {
	table table.Model

	app meta.App

	// The ledger/account etc. whose rows are being shown
	modelId   int
	modelName string

	rows []database.EntryRow
}

func (dv *DetailView) ModelId() int {
	return dv.modelId
}

func NewDetailView(app meta.App, itemId int, itemName string) *DetailView {
	tableModel := table.New(
		table.WithColumns([]table.Column{
			{
				Title: "Date",
				Width: 10,
			},
			{
				Title: "Ledger",
				Width: 20,
			},
			{
				Title: "Account",
				Width: 20,
			},
			{
				Title: "Debit",
				Width: 20,
			},
			{
				Title: "Credit",
				Width: 20,
			},
		}),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = lipgloss.NewStyle().Foreground(app.Colours().Foreground)
	tableModel.SetStyles(tableStyle)

	// I don't think we ever have to blur the table
	tableModel.Focus()

	return &DetailView{
		table: tableModel,

		app: app,

		modelId:   itemId,
		modelName: itemName,
	}
}

func (dv *DetailView) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(dv.app.CurrentMotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(dv.app.CurrentCommandSet())))

	// TODO: Also show the model metadata and not just the rows?
	cmds = append(cmds, dv.app.MakeLoadRowsCmd(dv.modelId))

	return tea.Batch(cmds...)
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		if message.Model != meta.ENTRYROW {
			panic(fmt.Sprintf("Expected an EntryRow, but got %v", message.Model))
		}

		dv.rows = message.Data.([]database.EntryRow)

		var tableRows []table.Row
		for _, row := range dv.rows {
			newTableRow := table.Row{}

			newTableRow = append(newTableRow, row.Date.String())
			// newTableRow = append(newTableRow, row.Ledger.String())
			newTableRow = append(newTableRow, "LedgerName")
			newTableRow = append(newTableRow, "AccountName")
			if row.Value > 0 {
				newTableRow = append(newTableRow, row.Value.String())
				newTableRow = append(newTableRow, "")
			} else {
				newTableRow = append(newTableRow, "")
				newTableRow = append(newTableRow, row.Value.String())
			}

			tableRows = append(tableRows, newTableRow)
		}

		dv.table.SetRows(tableRows)

		return dv, nil

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.table, cmd = dv.table.Update(keyMsg)

		slog.Debug(fmt.Sprintf("updated table: %#v", keyMsg))

		return dv, cmd

	case tea.WindowSizeMsg:
		// TODO maybe?
		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *DetailView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.app.Colours().Background).MarginLeft(2)
	result.WriteString(titleStyle.Render(fmt.Sprintf("%s detail view: %s", dv.app.Name(), dv.modelName)))
	result.WriteString("\n\n")

	result.WriteString(lipgloss.JoinVertical(
		lipgloss.Right,
		dv.table.View(),
		fmt.Sprintf("Total: %s", database.CalculateTotal(dv.rows)),
	))

	return result.String()
}

func (dv *DetailView) Type() meta.ViewType {
	return meta.DETAILVIEWTYPE
}

func (dv *DetailView) MotionSet() *meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.ModelId})

	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.ModelId})

	normalMotions.Insert(meta.Motion{"g", "d"}, meta.CommandMsg{
		Command: func(entries meta.App) tea.Msg {
			// I don't love the type assertion necessary here, but I don't hate it
			// This is a motion on DetailView anyway, how could it ever be a different view?
			// Well technically if the user is fast enough to insta switch to update view or smth,
			// and that MessageCmd happens to get processed faster
			row := entries.GetView().(*DetailView).table.Cursor()

			entryId := dv.rows[row].Entry

			// Stupid go not allowing to reference a const
			targetApp := meta.ENTRIES

			return meta.SwitchViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entryId}
		},
	})

	return &meta.MotionSet{
		Normal: normalMotions,
	}
}

func (dv *DetailView) CommandSet() *meta.CommandSet {
	return &meta.CommandSet{}
}
