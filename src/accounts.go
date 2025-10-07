package main

import (
	"fmt"
	"log/slog"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type AccountsApp struct {
	viewWidth, viewHeight int

	currentView meta.View
}

func NewAccountsApp() meta.App {
	model := &AccountsApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *AccountsApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *AccountsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

	case meta.SetupSchemaMsg:
		changed, err := database.SetupSchemaAccounts()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `accounts` TABLE: %v", err)
			return app, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changed {
			slog.Info("Set up `Accounts` schema")
			return app, nil
		}

		return app, nil
	}

	return app, nil
}

func (app *AccountsApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render("TODO accounts")
}

func (app *AccountsApp) Name() string {
	return "Accounts"
}

func (app *AccountsApp) Colours() meta.AppColours {
	return meta.AppColours{
		Foreground: "#7BD4EA",
		Accent:     "#7BD4EA",
		Background: "#7BD4EA",
	}
}

func (app *AccountsApp) CurrentMotionSet() *meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *AccountsApp) CurrentCommandSet() *meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *AccountsApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ACCOUNT:  {},
		meta.ENTRYROW: {},
	}
}

func (app *AccountsApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectAccounts()
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ACCOUNTS,
			Model:     meta.ACCOUNT,
			Data:      items,
		}
	}
}

func (app *AccountsApp) MakeLoadRowsCmd(modelId int) tea.Cmd {
	// Aren't closures just great (still)
	return func() tea.Msg {
		rows, err := database.SelectRowsByAccount(modelId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ACCOUNTS,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
