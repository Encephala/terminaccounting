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

	currentView meta.View
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

		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

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

		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

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

		case meta.UPDATEVIEWTYPE:
			cmds = append(cmds, m.showUpdateView())

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		cmds = append(
			cmds,
			meta.MessageCmd(meta.UpdateViewMotionSetMsg(m.CurrentMotionSet())),
			meta.MessageCmd(meta.UpdateViewCommandSetMsg(m.CurrentCommandSet())),
		)

		return m, tea.Batch(cmds...)

	case meta.SwitchFocusMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.SaveMsg:
		createView := m.currentView.(*CreateView)

		ledgerName := createView.nameInput.Value()
		ledgerType := createView.typeInput.Value().(LedgerType)
		ledgerNotes := createView.noteInput.Value()

		newLedger := Ledger{
			Name:       ledgerName,
			LedgerType: ledgerType,
			Notes:      strings.Split(ledgerNotes, "\n"),
		}

		_, err := newLedger.Insert(m.db)

		if err != nil {
			return m, meta.MessageCmd(err)
		}

		// TODO:
		// - Move to the update view after writing, using the ID returned from `newLedger.Insert` above
		// - Send a vimesque message to inform the user of successful creation (when vimesque messages are implemented)

		return m, nil
	}

	newView, cmd := m.currentView.Update(message)
	m.currentView = newView.(meta.View)

	return m, cmd
}

func (m *model) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.currentView.View())
}

func (m *model) Name() string {
	return "Ledgers"
}

func (m *model) Colours() styles.AppColours {
	return styles.LEDGERSSTYLES
}

func (m *model) CurrentMotionSet() *meta.MotionSet {
	return m.currentView.MotionSet()
}

func (m *model) CurrentCommandSet() *meta.CommandSet {
	return m.currentView.CommandSet()
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
			Data:      items,
		}
	}
}

func (m *model) showListView() tea.Cmd {
	m.currentView = meta.NewListView(m)
	return m.makeLoadLedgersCmd()
}

func (m *model) makeLoadLedgerRowsCmd(ledgerId int) tea.Cmd {
	return func() tea.Msg {
		rows, err := entries.SelectRowsByLedger(m.db, ledgerId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err)
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: m.Name(),
			Model:     "EntryRow",
			Data:      items,
		}
	}
}

func (m *model) showDetailView() tea.Cmd {
	selectedLedger := m.currentView.(*meta.ListView).ListModel.SelectedItem().(Ledger)
	m.currentView = meta.NewDetailView(m, selectedLedger.Id, selectedLedger.Name)
	return m.makeLoadLedgerRowsCmd(selectedLedger.Id)
}

func (m *model) showCreateView() tea.Cmd {
	m.currentView = NewCreateView(m.Colours())
	return nil
}

func (m *model) makeLoadLedgerCmd(ledgerId int) tea.Cmd {
	return func() tea.Msg {
		ledger, err := SelectLedger(m.db, ledgerId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER WITH ID %d: %#v", ledgerId, err)
		}

		return meta.DataLoadedMsg{
			TargetApp: m.Name(),
			Model:     "Ledger",
			Data:      ledger,
		}
	}
}

func (m *model) showUpdateView() tea.Cmd {
	ledgerId := m.currentView.(*meta.DetailView).ModelId
	m.currentView = NewUpdateView(ledgerId, m.Colours())
	return m.makeLoadLedgerCmd(ledgerId)
}
