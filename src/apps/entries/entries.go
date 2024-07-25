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
	m.model = meta.NewListView(m)

	return func() tea.Msg {
		rows, err := SelectEntries(m.db)
		if err != nil {
			return utils.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			Model: "Entries",
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
	style := styles.Body(m.viewWidth, m.viewHeight)

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
