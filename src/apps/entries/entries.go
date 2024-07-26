package entries

import (
	"fmt"
	"log/slog"
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
		changedEntries, err := setupSchemaEntries(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entries` TABLE: %v", err)
			return m, utils.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		changedEntryRows, err := setupSchemaEntryRows(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entryrows` TABLE: %v", err)
			return m, utils.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changedEntries || changedEntryRows {
			slog.Info("Set up `Entries` schema")
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
	return "Entries"
}

func (m *model) Colours() styles.AppColours {
	return styles.ENTRIESCOLOURS
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

		default:
			panic(fmt.Sprintf("unexpected vim.Direction %#v", message.Data.(vim.Direction)))
		}

		newView, cmd := m.view.Update(keyMsg)
		m.view = newView.(meta.View)
		return m, cmd
	}

	return m, nil
}

func (m *model) makeLoadEntriesCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectEntries(m.db)
		if err != nil {
			return utils.MessageCmd(fmt.Errorf("FAILED TO LOAD ENTRIES: %v", err))
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
	return m.makeLoadEntriesCmd()
}
