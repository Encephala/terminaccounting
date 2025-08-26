package entries

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"terminaccounting/apps/journals"
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
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(meta.View)

		return m, cmd

	case meta.CommitCreateMsg:
		createView := m.currentView.(*CreateView)

		entryJournal := createView.journalInput.Value().(journals.Journal)
		entryNotes := createView.notesInput.Value()

		// TODO: Actually create the entry in db and stuff.
		// TODO: Decide on this implementation vs the one in CreateView.Update
		_ = Entry{
			Journal: entryJournal.Id,
			Notes:   strings.Split(entryNotes, "\n"),
		}

		// id, err := newEntry.Insert(m.db)

		// if err != nil {
		// 	return m, meta.MessageCmd(err)
		// }

		// m.currentView = NewUpdateView(m, id)

		return m, m.currentView.Init()

	case meta.SwitchViewMsg:
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.currentView = meta.NewListView(m)

		case meta.DETAILVIEWTYPE:
			selectedEntry := m.currentView.(*meta.ListView).ListModel.SelectedItem().(Entry)

			m.currentView = meta.NewDetailView(m, selectedEntry.Id, strconv.Itoa(selectedEntry.Id))

		case meta.CREATEVIEWTYPE:
			m.currentView = NewCreateView(m.db, m.Colours())

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

func (m *model) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.currentView.View())
}

func (m *model) Name() string {
	return "Entries"
}

func (m *model) Colours() styles.AppColours {
	return styles.ENTRIESCOLOURS
}

func (m *model) CurrentMotionSet() *meta.MotionSet {
	return m.currentView.MotionSet()
}

func (m *model) CurrentCommandSet() *meta.CommandSet {
	return m.currentView.CommandSet()
}

func (m *model) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRY:    {},
		meta.ENTRYROW: {},
		meta.JOURNAL:  {},
	}
}

func (m *model) MakeLoadListCmd() tea.Cmd {
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
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRY,
			Data:      items,
		}
	}
}

func (m *model) MakeLoadRowsCmd() tea.Cmd {
	// Aren't closures just great
	entryId := m.currentView.(*meta.DetailView).ModelId

	return func() tea.Msg {
		rows, err := SelectRowsByEntry(m.db, entryId)
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

func (m *model) MakeLoadDetailCmd() tea.Cmd {
	panic("TODO")
}

func makeSelectJournalsCmd(db *sqlx.DB) tea.Cmd {
	return func() tea.Msg {
		rows, err := journals.SelectJournals(db)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD JOURNALS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.JOURNAL,
			Data:      rows,
		}
	}
}
