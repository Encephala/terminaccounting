package journals

import "github.com/jmoiron/sqlx"

// import (
// 	"fmt"
// 	"log/slog"
// 	"terminaccounting/meta"
// 	"terminaccounting/styles"
// 	"terminaccounting/utils"

// 	tea "github.com/charmbracelet/bubbletea"
// )

var DB *sqlx.DB

// type model struct {
// 	viewWidth, viewHeight int
// }

// func New() meta.App {
// 	return &model{}
// }

// func (m *model) Init() tea.Cmd {
// 	return nil
// }

// func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
// 	switch message := message.(type) {
// 	case tea.WindowSizeMsg:
// 		m.viewWidth = message.Width
// 		m.viewHeight = message.Height

// 	case meta.SetupSchemaMsg:
// 		changed, err := setupSchema(message.Db)
// 		if err != nil {
// 			message := fmt.Errorf("COULD NOT CREATE `journals` TABLE: %v", err)
// 			return m, utils.MessageCommand(meta.FatalErrorMsg{Error: message})
// 		}

// 		if changed != 0 {
//			slog.Info("Set up `Journals` schema")
// 			return m, nil
// 		}

// 		return m, nil
// 	}

// 	return m, nil
// }

// func (m *model) View() string {
// 	style := styles.Body(m.viewWidth, m.viewHeight, m.Styles().Accent)
// 	return style.Render("TODO journals")
// }

// func (m *model) Name() string {
// 	return "Journals"
// }

// func (m *model) Styles() styles.AppColours {
// 	return styles.AppColours{
// 		Foreground: "#F6D6D6D0",
// 		Accent:     "#F6D6D680",
// 		Background: "#F6D6D6FF",
// 	}
// }
