package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type EntriesApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewEntriesApp() meta.App {
	model := &EntriesApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (m *EntriesApp) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m *EntriesApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

		return m, nil

	case meta.SetupSchemaMsg:
		changedEntries, err := database.SetupSchemaEntries()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entries` TABLE: %v", err)
			return m, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		changedEntryRows, err := database.SetupSchemaEntryRows()
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
		newView, cmd := m.currentView.Update(message)
		// TODO: This is crashing for some reason after comitting an entry create, idk
		m.currentView = newView.(view.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(view.View)

		return m, cmd

	case meta.SwitchViewMsg:
		if message.App != nil && *message.App != m.Type() {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.currentView = view.NewListView(m)

		case meta.DETAILVIEWTYPE:
			entry := m.currentView.(*view.ListView).ListModel.SelectedItem().(database.Entry)

			// No better model name to be had than the entry Id
			m.currentView = view.NewDetailView(m, entry.Id, strconv.Itoa(entry.Id))

		case meta.CREATEVIEWTYPE:
			m.currentView = view.NewEntryCreateView(m.Colours())

		case meta.UPDATEVIEWTYPE:
			entryId := message.Data.(int)

			m.currentView = view.NewEntryUpdateView(entryId, m.Colours())

		case meta.DELETEVIEWTYPE:
			// TODO

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return m, m.currentView.Init()
	}

	newView, cmd := m.currentView.Update(message)
	m.currentView = newView.(view.View)

	return m, cmd
}

func (m *EntriesApp) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.currentView.View())
}

func (m *EntriesApp) Name() string {
	return "Entries"
}

func (m *EntriesApp) Type() meta.AppType {
	return meta.ENTRIES
}

func (m *EntriesApp) Colours() styles.AppColours {
	return styles.ENTRIESCOLOURS
}

func (m *EntriesApp) CurrentMotionSet() *meta.MotionSet {
	return m.currentView.MotionSet()
}

func (m *EntriesApp) CurrentCommandSet() *meta.CommandSet {
	return m.currentView.CommandSet()
}

func (m *EntriesApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRY:    {},
		meta.ENTRYROW: {},
		meta.JOURNAL:  {},
		meta.LEDGER:   {},
		meta.ACCOUNT:  {},
	}
}

func (m *EntriesApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectEntries()
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD ENTRIES: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRY,
			Data:      items,
		}
	}
}

func (m *EntriesApp) MakeLoadRowsCmd(entryId int) tea.Cmd {
	// Aren't closures just great
	return func() tea.Msg {
		rows, err := database.SelectRowsByEntry(entryId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ENTRY ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
