package main

// import (
// 	"fmt"
// 	"log/slog"
// 	"terminaccounting/meta"
// 	"terminaccounting/styles"
// 	"terminaccounting/utils"

// 	tea "github.com/charmbracelet/bubbletea"
// )

// type AccountsApp struct {
// 	viewWidth, viewHeight int
// }

// func NewAccountsApp() meta.App {
// 	return &model{}
// }

// func (m *AccountsApp) Init() tea.Cmd {
// 	return nil
// }

// func (m *AccountsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
// 	switch message := message.(type) {
// 	case tea.WindowSizeMsg:
// 		m.viewWidth = message.Width
// 		m.viewHeight = message.Height

// 	case meta.SetupSchemaMsg:
// 		changed, err := setupSchema()
// 		if err != nil {
// 			message := fmt.Errorf("COULD NOT CREATE `accounts` TABLE: %v", err)
// 			return m, utils.MessageCommand(meta.FatalErrorMsg{Error: message})
// 		}

// 		if changed != 0 {
//			slog.Info("Set up `Accounts` schema")
// 			return m, nil
// 		}

// 		return m, nil
// 	}

// 	return m, nil
// }

// func (m *AccountsApp) View() string {
// 	style := styles.Body(m.viewWidth, m.viewHeight, m.Styles().Accent)
// 	return style.Render("TODO accounts")
// }

// func (m *AccountsApp) Name() string {
// 	return "Accounts"
// }

// func (m *AccountsApp) Styles() styles.AppStyles {
// 	return styles.AppStyles{
// 		Foreground: "#7BD4EAD0",
// 		Accent:     "#7BD4EA50",
// 		Background: "#7BD4EAFF",
// 	}
// }
