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
	return m.showListView()
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
			return m, utils.MessageCmd(meta.FatalErrorMsg{Error: message})
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

	case vim.CompletedMotionMsg:
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

func (m *model) CurrentMotionSet() *vim.MotionSet {
	return m.view.MotionSet()
}

func (m *model) handleMotionMessage(message vim.CompletedMotionMsg) (*model, tea.Cmd) {
	switch message.Type {
	case vim.NAVIGATE:
		keyMsg := tea.KeyMsg{
			Type:  tea.KeyRunes,
			Alt:   false,
			Paste: false,
		}

		switch message.Data.(vim.Direction) {
		case vim.DOWN:
			keyMsg.Runes = []rune{'j'}

		case vim.UP:
			keyMsg.Runes = []rune{'k'}

		case vim.LEFT:
			keyMsg.Runes = []rune{'h'}

		case vim.RIGHT:
			keyMsg.Runes = []rune{'l'}
		}

		newView, cmd := m.view.Update(keyMsg)
		m.view = newView.(meta.View)
		return m, cmd

	case vim.SWITCHVIEW:
		var cmds []tea.Cmd
		switch message.Data.(vim.View) {
		case vim.LISTVIEW:
			cmds = append(cmds, m.showListView())

		case vim.DETAILVIEW:
			cmds = append(cmds, m.showDetailView())

		case vim.CREATEVIEW:
			cmds = append(cmds, m.showCreateView())
		}

		cmds = append(cmds, utils.MessageCmd(meta.UpdateViewMotionSetMsg(
			m.CurrentMotionSet(),
		)))

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m *model) makeLoadLedgersCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectLedgers(m.db)
		if err != nil {
			return utils.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
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
			utils.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err))
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
	m.view = NewCreateView(m, m.Colours())
	return nil
}
