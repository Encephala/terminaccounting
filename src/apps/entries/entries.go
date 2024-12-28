package entries

import (
	"fmt"
	"log/slog"
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

		return m, nil

	case meta.SetupSchemaMsg:
		changedEntries, err := setupSchemaEntries(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entries` TABLE: %v", err)
			return m, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		changedEntryRows, err := setupSchemaEntryRows(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entryrows` TABLE: %v", err)
			return m, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
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

	case meta.NavigateMsg:
		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd
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

func (m *model) CurrentMotionSet() *meta.MotionSet {
	return m.view.MotionSet()
}

func (m *model) CurrentCommandSet() *meta.CommandSet {
	return m.view.CommandSet()
}

func (m *model) makeLoadEntriesCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectEntries(m.db)
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD ENTRIES: %v", err))
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
