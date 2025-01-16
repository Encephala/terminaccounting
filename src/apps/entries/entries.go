package entries

import (
	"fmt"
	"log/slog"
	"strconv"
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
	model := &model{
		db: db,
	}

	model.view = meta.NewListView(model)

	return model
}

func (m *model) Init() tea.Cmd {
	return m.view.Init()
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
		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.view.Update(message)
		m.view = newView.(meta.View)

		return m, cmd

	case meta.SwitchViewMsg:
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.view = meta.NewListView(m)

		case meta.DETAILVIEWTYPE:
			selectedEntry := m.view.(*meta.ListView).ListModel.SelectedItem().(Entry)

			m.view = meta.NewDetailView(m, selectedEntry.Id, strconv.Itoa(selectedEntry.Id))

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return m, m.view.Init()
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

func (m *model) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRY:    {},
		meta.ENTRYROW: {},
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
	entryId := m.view.(*meta.DetailView).ModelId

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
