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
	model := &model{
		db: db,
	}

	model.currentView = meta.NewListView(model)

	return model
}

func (m *model) Init() tea.Cmd {
	return m.currentView.Init()
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
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.SwitchViewMsg:
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.currentView = meta.NewListView(m)

		case meta.DETAILVIEWTYPE:
			selectedLedger := m.currentView.(*meta.ListView).ListModel.SelectedItem().(Ledger)

			m.currentView = meta.NewDetailView(m, selectedLedger.Id, selectedLedger.Name)

		case meta.CREATEVIEWTYPE:
			m.currentView = NewCreateView(m.Colours())

		case meta.UPDATEVIEWTYPE:
			ledgerId := m.currentView.(*meta.DetailView).ModelId

			m.currentView = NewUpdateView(m, ledgerId)

		case meta.DELETEVIEWTYPE:
			ledgerId := m.currentView.(*meta.DetailView).ModelId

			m.currentView = NewDeleteView(m, ledgerId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return m, m.currentView.Init()

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

		m.currentView = NewUpdateView(m, id)
		// TODO: Add a vimesque message to inform the user of successful creation (when vimesque messages are implemented)
		// Or maybe this should just switch to the list view or the detail view? Idk

		return m, m.currentView.Init()

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

	case meta.CommitDeleteMsg:
		view := m.currentView.(*DeleteView)

		err := DeleteLedger(m.db, view.model.Id)

		m.currentView = meta.NewListView(m)
		// TODO: Add a vimesque message to inform user of successful deletion

		return m, tea.Batch(meta.MessageCmd(err), m.currentView.Init())
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

func (m *model) MakeLoadListCmd() tea.Cmd {
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

func (m *model) MakeLoadRowsCmd() tea.Cmd {
	// Aren't closures just great
	ledgerId := m.currentView.(*meta.DetailView).ModelId

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

func (m *model) MakeLoadDetailCmd() tea.Cmd {
	var ledgerId int
	switch view := m.currentView.(type) {
	case *UpdateView:
		ledgerId = view.modelId
	case *DeleteView:
		ledgerId = view.modelId

	default:
		panic(fmt.Sprintf("unexpected view: %#v", view))
	}

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
