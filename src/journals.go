package main

// import (
// 	"fmt"
// 	"log/slog"
// 	"terminaccounting/meta"
// 	"terminaccounting/styles"
// 	"terminaccounting/utils"

// 	tea "github.com/charmbracelet/bubbletea"
// )

// type JournalsApp struct {
// 	viewWidth, viewHeight int
// }

// func NewJournalsApp() meta.App {
// 	return &model{}
// }

// func (m *JournalsApp) Init() tea.Cmd {
// 	return nil
// }

// func (m *JournalsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
// 	switch message := message.(type) {
// 	case tea.WindowSizeMsg:
// 		m.viewWidth = message.Width
// 		m.viewHeight = message.Height

// 	case meta.SetupSchemaMsg:
// 		changed, err := setupSchema()
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

// func (m *JournalsApp) View() string {
// 	style := styles.Body(m.viewWidth, m.viewHeight, m.Styles().Accent)
// 	return style.Render("TODO journals")
// }

// func (m *JournalsApp) Name() string {
// 	return "Journals"
// }

// func (m *JournalsApp) Styles() styles.AppColours {
// 	return styles.AppColours{
// 		Foreground: "#F6D6D6D0",
// 		Accent:     "#F6D6D680",
// 		Background: "#F6D6D6FF",
// 	}
// }
