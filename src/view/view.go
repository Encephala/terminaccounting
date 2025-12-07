package view

import (
	"errors"
	"fmt"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View interface {
	tea.Model

	AcceptedModels() map[meta.ModelType]struct{}

	MotionSet() meta.MotionSet
	CommandSet() meta.CommandSet
}

type activeInput int

func (input *activeInput) previous(numInputs int) {
	*input--

	if *input < 0 {
		*input += activeInput(numInputs)
	}
}

func (input *activeInput) next(numInputs int) {
	*input++

	*input %= activeInput(numInputs)
}

const (
	NAMEINPUT activeInput = iota
	TYPEINPUT
	NOTEINPUT
)

type ListView struct {
	listModel list.Model

	app meta.App
}

func NewListView(app meta.App) *ListView {
	viewStyles := meta.NewListViewStyles(app.Colours().Accent, app.Colours().Foreground)

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = viewStyles.ListDelegateSelectedTitle
	delegate.Styles.SelectedDesc = viewStyles.ListDelegateSelectedDesc

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

func (lv *ListView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
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

	case meta.ResetSearchMsg:
		lv.listModel.ResetFilter()

		return lv, nil

	case meta.UpdateSearchMsg:
		lv.listModel.SetFilterText(message.Query)

		return lv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (lv *ListView) View() string {
	return lv.listModel.View()
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
	normalMotions.Insert(meta.Motion{"ctrl+l"}, meta.ResetSearchMsg{})

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

func (lv *ListView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		item := lv.listModel.SelectedItem()

		if item == nil {
			return errors.New("no item to goto detail view of")
		}

		return meta.SwitchViewMsg{ViewType: meta.DETAILVIEWTYPE, Data: item}
	}
}

// A generic, placeholder(?) view that just renders all entries on a ledger/journal/account in a list.
type DetailView struct {
	table table.Model

	app meta.App

	// The ledger/account etc. whose rows are being shown
	modelId   int
	modelName string

	rows []database.EntryRow

	availableLedgers  []database.Ledger
	availableAccounts []database.Account
}

func NewDetailView(app meta.App, itemId int, itemName string) *DetailView {
	tableModel := table.New()
	// I don't think we ever have to blur the table
	tableModel.Focus()

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = lipgloss.NewStyle().Foreground(app.Colours().Foreground)
	tableModel.SetStyles(tableStyle)

	view := &DetailView{
		table: tableModel,

		app: app,

		modelId:   itemId,
		modelName: itemName,
	}

	view.updateTableWidth(90)

	return view
}

func (dv *DetailView) Init() tea.Cmd {
	// TODO: Also show the model metadata and not just the rows?
	var cmds []tea.Cmd
	cmds = append(cmds, dv.app.MakeLoadRowsCmd(dv.modelId))

	cmds = append(cmds, database.MakeSelectLedgersCmd(dv.app.Type()))
	cmds = append(cmds, database.MakeSelectAccountsCmd(dv.app.Type()))

	return tea.Batch(cmds...)
}

func (dv *DetailView) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.DataLoadedMsg:
		switch message.Model {
		case meta.ENTRYROWMODEL:
			dv.rows = message.Data.([]database.EntryRow)

		case meta.ACCOUNTMODEL:
			dv.availableAccounts = message.Data.([]database.Account)

		case meta.LEDGERMODEL:
			dv.availableLedgers = message.Data.([]database.Ledger)

		default:
			panic(fmt.Sprintf("unexpected meta.ModelType: %#v", message.Model))
		}

		cmd := dv.updateTableRows()

		return dv, cmd

	case meta.NavigateMsg:
		keyMsg := meta.NavigateMessageToKeyMsg(message)

		var cmd tea.Cmd
		dv.table, cmd = dv.table.Update(keyMsg)

		return dv, cmd

	case tea.WindowSizeMsg:
		dv.updateTableWidth(message.Width)

		// -3 for the title and table header (header is not considered for table width)
		// -3 to for the total row
		// -1 for padding at the bottom
		dv.table.SetHeight(message.Height - 3 - 1 - 1)

		return dv, nil

	default:
		panic(fmt.Sprintf("unexpected tea.Msg: %#v", message))
	}
}

func (dv *DetailView) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(dv.app.Colours().Background).MarginLeft(2)
	result.WriteString(titleStyle.Render(fmt.Sprintf("%s Details: %s", dv.app.Name(), dv.modelName)))
	result.WriteString("\n\n")

	result.WriteString(lipgloss.JoinVertical(
		lipgloss.Right,
		dv.table.View(),
		fmt.Sprintf("Total: %s", database.CalculateTotal(dv.rows)),
	))

	return result.String()
}

func (dv *DetailView) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRYROWMODEL: {},
		meta.ACCOUNTMODEL:  {},
		meta.LEDGERMODEL:   {},
	}
}

func (dv *DetailView) MotionSet() meta.MotionSet {
	var normalMotions meta.Trie[tea.Msg]

	normalMotions.Insert(meta.Motion{"h"}, meta.NavigateMsg{Direction: meta.LEFT})
	normalMotions.Insert(meta.Motion{"j"}, meta.NavigateMsg{Direction: meta.DOWN})
	normalMotions.Insert(meta.Motion{"k"}, meta.NavigateMsg{Direction: meta.UP})
	normalMotions.Insert(meta.Motion{"l"}, meta.NavigateMsg{Direction: meta.RIGHT})

	normalMotions.Insert(meta.Motion{"g", "l"}, meta.SwitchViewMsg{ViewType: meta.LISTVIEWTYPE})
	normalMotions.Insert(meta.Motion{"g", "x"}, meta.SwitchViewMsg{ViewType: meta.DELETEVIEWTYPE, Data: dv.modelId})

	normalMotions.Insert(meta.Motion{"g", "e"}, meta.SwitchViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: dv.modelId})

	normalMotions.Insert(meta.Motion{"g", "d"}, dv.makeGoToDetailViewCmd())

	return meta.MotionSet{Normal: normalMotions}
}

func (dv *DetailView) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}

func (dv *DetailView) updateTableRows() tea.Cmd {
	var tableRows []table.Row
	for _, row := range dv.rows {
		newTableRow := table.Row{}

		newTableRow = append(newTableRow, row.Date.String())

		italicStyle := lipgloss.NewStyle().Italic(true)
		var ledger, account string

		if dv.availableLedgers != nil {
			found := false
			for _, l := range dv.availableLedgers {
				if l.Id == row.Ledger {
					ledger = l.Name
					found = true
					break
				}
			}

			if !found {
				return meta.MessageCmd(meta.FatalErrorMsg{Error: fmt.Errorf("couldn't find ledger %d", row.Ledger)})
			}
		} else {
			ledger = italicStyle.Render("Ledger")
		}

		if row.Account == nil {
			account = lipgloss.NewStyle().Italic(true).Render("None")
		} else if dv.availableAccounts != nil {
			found := false
			for _, a := range dv.availableAccounts {
				if a.Id == *row.Account {
					account = a.Name
					found = true
					break
				}
			}

			if !found {
				return meta.MessageCmd(meta.FatalErrorMsg{Error: fmt.Errorf("couldn't find account %d", row.Account)})
			}
		} else {
			account = italicStyle.Render("Account")
		}

		newTableRow = append(newTableRow, ledger)
		newTableRow = append(newTableRow, account)
		if row.Value > 0 {
			newTableRow = append(newTableRow, row.Value.String())
			newTableRow = append(newTableRow, "")
		} else {
			newTableRow = append(newTableRow, "")
			newTableRow = append(newTableRow, (-row.Value).String())
		}

		tableRows = append(tableRows, newTableRow)
	}

	dv.table.SetRows(tableRows)

	return nil
}

func (dv *DetailView) makeGoToDetailViewCmd() tea.Cmd {
	return func() tea.Msg {
		entryId := dv.rows[dv.table.Cursor()].Entry

		// Do the database query for the entry here, because it is a command and thus asynchronous
		entry, err := database.SelectEntry(entryId)

		if err != nil {
			return meta.MessageCmd(err)
		}

		// Stupid go not allowing to reference a const
		targetApp := meta.ENTRIESAPP
		return meta.SwitchViewMsg{App: &targetApp, ViewType: meta.DETAILVIEWTYPE, Data: entry}
	}
}

func (dv *DetailView) updateTableWidth(totalWidth int) {
	// This is simply the width of a date field
	dateWidth := 10

	// -2 because of left/right padding
	remainingWidth := totalWidth - dateWidth - 2
	// -8 because of the 2-wide gap between columns
	othersWidth := (remainingWidth - 8) / 4

	dv.table.SetColumns([]table.Column{
		{
			Title: "Date",
			Width: dateWidth,
		},
		{
			Title: "Ledger",
			Width: othersWidth,
		},
		{
			Title: "Account",
			Width: othersWidth,
		},
		{
			Title: "Debit",
			Width: othersWidth,
		},
		{
			Title: "Credit",
			Width: othersWidth,
		},
	})
}
