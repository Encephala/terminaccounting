package entries

import (
	"fmt"
	"log/slog"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/utils"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeView meta.ViewType
	model      tea.Model
}

func New(db *sqlx.DB) meta.App {
	return &model{
		db: db,
	}
}

func (m *model) Init() tea.Cmd {
	return nil
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
			return m, utils.MessageCommand(meta.FatalErrorMsg{Error: message})
		}

		changedEntryRows, err := setupSchemaEntryRows(message.Db)
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entryrows` TABLE: %v", err)
			return m, utils.MessageCommand(meta.FatalErrorMsg{Error: message})
		}

		if changedEntries+changedEntryRows != 0 {
			return m, func() tea.Msg {
				slog.Info("Set up `Entries` schema")
				return nil
			}
		}

		return m, nil
	}

	var cmd tea.Cmd
	m.model, cmd = m.model.Update(message)

	return m, cmd
}

func (m *model) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight, m.Colours().Accent)

	return style.Render(m.model.View())
}

func (m *model) Name() string {
	return "Entries"
}

func (m *model) Colours() styles.AppColours {
	return styles.ENTRIESCOLOURS
}

func (m *model) ActiveView() meta.ViewType {
	return m.activeView
}

func (m *model) SetActiveView(view meta.ViewType) (meta.App, tea.Cmd) {
	var cmd tea.Cmd

	viewInt := int(view)
	numberOfRegisteredViews := 2
	if view < 0 {
		viewInt += numberOfRegisteredViews
	} else if view >= meta.ViewType(numberOfRegisteredViews) {
		viewInt -= numberOfRegisteredViews
	}
	view = meta.ViewType(viewInt)

	switch view {
	case meta.ListViewType:
		listView := meta.NewListView(
			"Entries",
			styles.NewListViewStyles(m.Colours().Accent, m.Colours().Foreground),
		)
		m.model = &listView

		cmd = func() tea.Msg {
			rows, err := SelectEntries(m.db)

			if err != nil {
				errorMessage := fmt.Errorf("FAILED TO LOAD `entries` TABLE: %v", err)
				return meta.FatalErrorMsg{Error: errorMessage}
			}

			items := []list.Item{}
			for _, row := range rows {
				items = append(items, row)
			}

			return meta.DataLoadedMsg{
				Model: "EntryRow",
				Items: items,
			}
		}

	case meta.DetailViewType:
		detailView := meta.NewDetailView(
			"Entries",
			styles.NewDetailViewStyles(m.Colours().Foreground),
		)
		m.model = &detailView

		cmd = func() tea.Msg {
			// TODO: This shouldn't be hardcoded 1. Rework this function to be able to take (a) paremeter(s)
			entryrows, err := SelectRowsByEntry(m.db, 1)
			if err != nil {
				errorMessage := fmt.Errorf("FAILED TO LOAD `entryrows`: %v", err)
				return meta.FatalErrorMsg{Error: errorMessage}
			}

			items := []list.Item{}
			for _, entryrow := range entryrows {
				items = append(items, entryrow)
			}

			return meta.DataLoadedMsg{
				Model: "EntryRow",
				Items: items,
			}
		}

	default:
		panic(fmt.Sprintf("Unimplemented entries view %v", meta.ListViewType))
	}

	m.activeView = view
	return m, cmd
}
