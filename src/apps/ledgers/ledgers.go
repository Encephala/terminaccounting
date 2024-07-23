package ledgers

import (
	"fmt"
	"log/slog"
	"terminaccounting/apps/entries"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/utils"
	"terminaccounting/vim"

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
	m.view = meta.NewListView(m)

	return func() tea.Msg {
		rows, err := SelectLedgers(m.db)
		if err != nil {
			return utils.MessageCommand(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			Model: "Ledgers",
			Items: items,
		}
	}
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

		return m, nil

	case meta.SetupSchemaMsg:
		changed, err := setupSchema(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `ledgers` TABLE: %v", err)
			return m, utils.MessageCommand(meta.FatalErrorMsg{Error: message})
		}

		if changed != 0 {
			return m, func() tea.Msg {
				slog.Info("Set up `ledgers` schema")
				return nil
			}
		}

		return m, nil

	case meta.CompletedMotionMsg:
		return m.handleMotionMessage(message)

	case tea.KeyMsg:
		panic(fmt.Sprintf("App received %#v, this should be a meta.KeyModeMsg", message))
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

func (m *model) handleMotionMessage(message meta.CompletedMotionMsg) (meta.App, tea.Cmd) {
	msg := vim.Stroke(message)

	switch {
	case msg.Equals(vim.Stroke{"enter"}):
		return m.showDetailView()

	case msg.Equals(vim.Stroke{"ctrl+o"}):
		return m.showListView()

	case msg.Equals(vim.Stroke{"ctrl+n"}):
		return m.showCreateView()

	case len(msg) == 1 && vim.MotionKeys.Contains(msg[0]):
		keyMsg := tea.KeyMsg{
			Type:  -1,
			Runes: []rune(msg[0]),
			Alt:   false,
			Paste: false,
		}

		newView, cmd := m.view.Update(keyMsg)
		m.view = newView.(meta.View)
		return m, cmd
	}

	return m, nil
}

func (m *model) loadLedgersCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectLedgers(m.db)
		if err != nil {
			return utils.MessageCommand(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			Model: "Ledger",
			Items: items,
		}
	}
}

func (m *model) showListView() (meta.App, tea.Cmd) {
	switch m.view.Type() {
	case meta.DetailViewType:
		m.view = meta.NewListView(m)
		return m, m.loadLedgersCmd()
	}

	return m, nil
}

func (m *model) loadLedgerRowsCmd(selectedLedger Ledger) tea.Cmd {
	return func() tea.Msg {
		rows, err := entries.SelectRowsByLedger(m.db, selectedLedger.Id)
		if err != nil {
			utils.MessageCommand(fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			Model: "EntryRow",
			Items: items,
		}
	}
}

func (m *model) showDetailView() (meta.App, tea.Cmd) {
	switch m.view.Type() {
	case meta.ListViewType:
		selectedLedger := m.view.(*meta.ListView).Model.SelectedItem().(Ledger)

		m.view = meta.NewDetailView(m, selectedLedger.Name)
		return m, m.loadLedgerRowsCmd(selectedLedger)
	}

	return m, nil
}

func (m *model) showCreateView() (meta.App, tea.Cmd) {
	m.view = NewCreateView(m, m.Colours())
	return m, nil
}
