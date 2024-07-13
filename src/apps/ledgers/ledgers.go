package ledgers

import (
	"fmt"
	"log/slog"
	"terminaccounting/apps/entries"
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

	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyEnter:
			return m.handleEnterMsg()

		default:
			var cmd tea.Cmd
			m.model, cmd = m.model.Update(message)
			return m, cmd
		}

	case meta.SetActiveViewMsg:
		m.model = message.View
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
	return "Ledgers"
}

func (m *model) Colours() styles.AppColours {
	return styles.AppColours{
		Foreground: "#A1EEBDD0",
		Background: "#A1EEBD60",
		Accent:     "#A1EEBDFF",
	}
}

func (m *model) ActiveView() meta.ViewType {
	return m.activeView
}

func (m *model) handleEnterMsg() (meta.App, tea.Cmd) {
	switch m.activeView {
	case meta.ListViewType:
		selectedLedgerId := m.model.(*meta.ListView).Model.SelectedItem().(Ledger).Id
		newViewCmd := func() tea.Msg {
			rows, err := entries.SelectRowsByLedger(m.db, selectedLedgerId)
			if err != nil {
				utils.MessageCommand(fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err))
			}

			rowsAsItems := make([]list.Item, len(rows))
			for i, row := range rows {
				rowsAsItems[i] = row
			}

			return meta.SetActiveViewMsg{View: meta.NewDetailView(m, selectedLedgerId, rowsAsItems)}
		}

		return m, newViewCmd
	}

	return m, nil
}
