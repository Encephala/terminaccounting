package entries

import (
	"fmt"
	"log/slog"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/utils"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	viewWidth, viewHeight int
}

func New() meta.App {
	return &model{}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

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

	return m, nil
}

func (m *model) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight, m.Styles().Accent)
	return style.Render("TODO entries")
}

func (m *model) Name() string {
	return "Entries"
}

func (m *model) Styles() styles.AppStyles {
	return styles.AppStyles{
		Foreground: "#F0F1B2D0",
		Accent:     "#F0F1B280",
		Background: "#EBECABFF",
	}
}
