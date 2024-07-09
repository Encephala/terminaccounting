package ledgers

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

	activeView int
	models     []tea.Model

	ledgers []Ledger
}

func New(db *sqlx.DB) meta.App {
	result := &model{
		db: db,

		activeView: 0,
		models:     []tea.Model{},

		ledgers: []Ledger{},
	}

	listView := meta.NewListView(
		"Ledgers",
		styles.NewListViewStyles(result.Styles().Accent, result.Styles().Foreground),
	)
	result.models = append(result.models, &listView)

	return result
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd

	for _, model := range m.models {
		cmds = append(cmds, model.Init())
	}

	loadDataCmd := func() tea.Msg {
		ledgers, err := SelectAll(m.db)

		if err != nil {
			errorMessage := fmt.Errorf("FAILED TO LOAD `ledgers` TABLE: %v", err)
			return meta.FatalErrorMsg{Error: errorMessage}
		}

		items := []list.Item{}
		for _, ledger := range ledgers {
			items = append(items, ledger)
		}

		return meta.DataLoadedMsg{
			Model: "Ledger",
			Items: items,
		}
	}
	cmds = append(cmds, loadDataCmd)

	return tea.Batch(cmds...)
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

	case meta.DataLoadedMsg:
		if message.Model != "Ledger" {
			return m, nil
		}

		ledgers := []Ledger{}
		for _, item := range message.Items {
			ledgers = append(ledgers, item.(Ledger))
		}
		m.ledgers = ledgers

		// Update list view
		var cmd tea.Cmd
		m.models[0], cmd = m.models[0].Update(message)

		return m, cmd

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
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd
	for i, model := range m.models {
		m.models[i], cmd = model.Update(message)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if m.activeView < 0 || m.activeView >= len(m.models) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.activeView))
	}

	style := styles.Body(m.viewWidth, m.viewHeight, m.Styles().Accent)

	return style.Render(m.models[m.activeView].View())
}

func (m *model) Name() string {
	return "Ledgers"
}

func (m *model) Styles() styles.AppStyles {
	return styles.AppStyles{
		Foreground: "#A1EEBDD0",
		Background: "#A1EEBD60",
		Accent:     "#A1EEBDFF",
	}
}
