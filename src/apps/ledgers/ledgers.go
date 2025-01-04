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
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			cmd := m.showListView()
			return m, cmd

		case meta.DETAILVIEWTYPE:
			selectedLedger := m.currentView.(*meta.ListView).ListModel.SelectedItem().(Ledger)

			cmd := m.showDetailView(selectedLedger)
			return m, cmd

		case meta.CREATEVIEWTYPE:
			cmd := m.showCreateView()
			return m, cmd

		case meta.UPDATEVIEWTYPE:
			ledgerId := m.currentView.(*meta.DetailView).ModelId

			cmd := m.showUpdateView(ledgerId)
			return m, cmd

		case meta.DELETEVIEWTYPE:
			ledgerId := m.currentView.(*meta.DetailView).ModelId

			cmd := m.showDeleteView(ledgerId)
			return m, cmd

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

	case meta.SwitchFocusMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.CommitCreateMsg:
		createView := m.currentView.(*CreateView)

		ledgerName := createView.nameInput.Value()
		ledgerType := createView.typeInput.Value().(LedgerType)
		ledgerNotes := createView.noteInput.Value()

		newLedger := Ledger{
			Name:  ledgerName,
			Type:  ledgerType,
			Notes: strings.Split(ledgerNotes, "\n"),
		}

		id, err := newLedger.Insert(m.db)

		if err != nil {
			return m, meta.MessageCmd(err)
		}

		switchViewCmd := m.showUpdateView(id)
		// TODO: Add a vimesque message to inform the user of successful creation (when vimesque messages are implemented)

		return m, switchViewCmd

	case meta.CommitUpdateMsg:
		view := m.currentView.(*UpdateView)

		currentValues := Ledger{
			Id:    view.modelId,
			Name:  view.nameInput.Value(),
			Type:  view.typeInput.Value().(LedgerType),
			Notes: strings.Split(view.noteInput.Value(), "\n"),
		}

		currentValues.Update(m.db)

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
	var cmds []tea.Cmd

	view := meta.NewListView(m)

	m.currentView = view
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(view.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(view.CommandSet())))

	cmds = append(cmds, m.makeLoadLedgersCmd())

	return tea.Batch(cmds...)
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

func (m *model) showDetailView(ledger Ledger) tea.Cmd {
	var cmds []tea.Cmd

	view := meta.NewDetailView(m, ledger.Id, ledger.Name)

	m.currentView = view
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(view.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(view.CommandSet())))

	cmds = append(cmds, m.makeLoadLedgerRowsCmd(ledger.Id))

	return tea.Batch(cmds...)
}

func (m *model) showCreateView() tea.Cmd {
	var cmds []tea.Cmd

	view := NewCreateView(m.Colours())

	m.currentView = view
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(view.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(view.CommandSet())))

	return tea.Batch(cmds...)
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

func (m *model) showUpdateView(ledgerId int) tea.Cmd {
	var cmds []tea.Cmd

	view := NewUpdateView(ledgerId, m.Colours())

	m.currentView = view
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(view.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(view.CommandSet())))

	cmds = append(cmds, m.makeLoadLedgerCmd(ledgerId))

	return tea.Batch(cmds...)
}

func (m *model) showDeleteView(ledgerId int) tea.Cmd {
	var cmds []tea.Cmd

	view := NewDeleteView(m.Colours())

	m.currentView = view
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(view.MotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(view.CommandSet())))

	cmds = append(cmds, m.makeLoadLedgerCmd(ledgerId))

	return tea.Batch(cmds...)
}
