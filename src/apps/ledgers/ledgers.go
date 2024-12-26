package ledgers

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/apps/entries"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	view meta.View
}

func New(db *sqlx.DB) meta.App {
	return &model{
		db: db,
	}
}

func (m *model) Init() tea.Cmd {
	return m.showListView()
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.SetupSchemaMsg:
		changed, err := setupSchema(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `ledgers` TABLE: %v", err)
			return m, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changed {
			slog.Info("Set up `ledgers` schema")
			return m, nil
		}

		return m, nil

	case meta.DataLoadedMsg:
		message.ActualApp = m.Name()

		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.SwitchViewMsg:
		var cmds []tea.Cmd

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			cmds = append(cmds, m.showListView())

		case meta.DETAILVIEWTYPE:
			cmds = append(cmds, m.showDetailView())

		case meta.CREATEVIEWTYPE:
			cmds = append(cmds, m.showCreateView())
		}

		cmds = append(
			cmds,
			meta.MessageCmd(meta.UpdateViewMotionSetMsg(m.CurrentMotionSet())),
			meta.MessageCmd(meta.UpdateViewCommandSetMsg(m.CurrentCommandSet())),
		)

		return m, tea.Batch(cmds...)

	case meta.SwitchFocusMsg:
		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.CompletedCommandMsg:
		return m.handleCommandMessage(message)
	}

	newView, cmd := m.view.Update(message)
	m.view = newView.(meta.View)

	return m, cmd
}

func (m *model) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.view.View())
}

func (m *model) Name() string {
	return "Ledgers"
}

func (m *model) Colours() styles.AppColours {
	return styles.AppColours{
		Foreground: "#A1EEBDD0",
		Background: "#A1EEBD60",
		Accent:     "#A1EEBDFF",
	}
}

func (m *model) CurrentMotionSet() *meta.MotionSet {
	return m.view.MotionSet()
}

func (m *model) CurrentCommandSet() *meta.CommandSet {
	return m.view.CommandSet()
}

func (m *model) handleCommandMessage(message meta.CompletedCommandMsg) (*model, tea.Cmd) {
	switch message.Type {
	case meta.WRITE:
		// TODO
		// Uhh is the view even able to save the model? Should it even?
		// Like I guess the view is the only one that's able to access the input fields' values,
		// although that is fixed by a little type assertion.
		// Let's start writing and if it feels wrong, move the logic to the Ledgers model itself
		//
		// Wait I'm yapping dumb shit, this is the Ledgers model itself

		createView := m.view.(*CreateView)

		ledgerName := createView.nameInput.Value()
		ledgerType := createView.typeInput.Value().(LedgerType)
		ledgerNotes := createView.noteInput.Value()

		newLedger := Ledger{
			Name:       ledgerName,
			LedgerType: ledgerType,
			Notes:      strings.Split(ledgerNotes, "\n"),
		}

		err := newLedger.Insert(m.db)

		if err != nil {
			return m, meta.MessageCmd(err)
		}

		return m, nil

	default:
		panic(fmt.Sprintf("unexpected meta.completedCommandType: %#v", message.Type))
	}
}

func (m *model) makeLoadLedgersCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectLedgers(m.db)
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: m.Name(),
			Model:     "Ledger",
			Items:     items,
		}
	}
}

func (m *model) showListView() tea.Cmd {
	m.view = meta.NewListView(m)
	return m.makeLoadLedgersCmd()
}

func (m *model) makeLoadLedgerRowsCmd(selectedLedger Ledger) tea.Cmd {
	return func() tea.Msg {
		rows, err := entries.SelectRowsByLedger(m.db, selectedLedger.Id)
		if err != nil {
			meta.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: m.Name(),
			Model:     "EntryRow",
			Items:     items,
		}
	}
}

func (m *model) showDetailView() tea.Cmd {
	selectedLedger := m.view.(*meta.ListView).Model.SelectedItem().(Ledger)
	m.view = meta.NewDetailView(m, selectedLedger.Name)
	return m.makeLoadLedgerRowsCmd(selectedLedger)
}

func (m *model) showCreateView() tea.Cmd {
	m.view = NewCreateView(m, m.Colours(), m.viewWidth, m.viewHeight)
	return nil
}
