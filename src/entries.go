package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type EntriesApp struct {
	viewWidth, viewHeight int

	currentView meta.View
}

func NewEntriesApp() meta.App {
	model := &EntriesApp{}

	model.currentView = meta.NewListView(model)

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
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.CommitCreateMsg:
		createView := m.currentView.(*view.EntryCreateView)

		entryJournal := createView.JournalInput.Value().(database.Journal)
		entryNotes := createView.NotesInput.Value()

		entryRows, err := createView.EntryRowsManager.CompileRows()
		if err != nil {
			return m, meta.MessageCmd(err)
		}

		// TODO: Decide on this implementation vs the one in CreateView.Update
		// note from later me: wish I knew what this meant
		newEntry := database.Entry{
			Journal: entryJournal.Id,
			Notes:   strings.Split(entryNotes, "\n"),
		}

		_, err = newEntry.Insert(entryRows)
		if err != nil {
			return m, meta.MessageCmd(err)
		}

		// TODO: make this actually show update view instead of listview (should it?)
		// m.currentView = NewEntryUpdateView(id)
		m.currentView = meta.NewListView(m)

		return m, m.currentView.Init()

	case meta.SwitchViewMsg:
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.currentView = meta.NewListView(m)

		case meta.DETAILVIEWTYPE:
			selectedEntry := m.currentView.(*meta.ListView).ListModel.SelectedItem().(database.Entry)

			m.currentView = meta.NewDetailView(m, selectedEntry.Id, strconv.Itoa(selectedEntry.Id))

		case meta.CREATEVIEWTYPE:
			m.currentView = view.NewEntryCreateView(m.Colours())

		case meta.UPDATEVIEWTYPE:
			// TODO

		case meta.DELETEVIEWTYPE:
			// TODO

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return m, m.currentView.Init()
	}

	newView, cmd := m.currentView.Update(message)
	m.currentView = newView.(meta.View)

	return m, cmd
}

func (m *EntriesApp) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.currentView.View())
}

func (m *EntriesApp) Name() string {
	return "Entries"
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

func (m *EntriesApp) MakeLoadRowsCmd() tea.Cmd {
	// Aren't closures just great
	entryId := m.currentView.(*meta.DetailView).ModelId

	return func() tea.Msg {
		rows, err := database.SelectRowsByEntry(entryId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ENTRY ROWS: %v", err)
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRYROW,
			Data:      items,
		}
	}
}

func (m *EntriesApp) MakeLoadDetailCmd() tea.Cmd {
	panic("TODO")
}
