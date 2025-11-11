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

type accountsApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewAccountsApp() meta.App {
	model := &accountsApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *accountsApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *accountsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(view.View)

		return app, cmd

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

	case meta.DataLoadedMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(view.View)

		return app, cmd

	case meta.SwitchViewMsg:
		if message.App != nil && *message.App != meta.LEDGERSAPP {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			account := message.Data.(database.Account)

			app.currentView = view.NewDetailView(app, account.Id, account.Name)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewAccountsCreateView(app.Colours())

		case meta.UPDATEVIEWTYPE:
			accountId := message.Data.(int)

			app.currentView = view.NewAccountsUpdateView(accountId, app.Colours())

		case meta.DELETEVIEWTYPE:
			accountId := message.Data.(int)

			app.currentView = view.NewAccountsDeleteView(accountId, app.Colours())

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()

	case meta.SwitchFocusMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(view.View)

		return app, cmd

	case meta.NavigateMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(view.View)

		return app, cmd
	}

	newView, cmd := app.currentView.Update(message)
	app.currentView = newView.(view.View)

	return app, cmd
}

func (app *accountsApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *accountsApp) Name() string {
	return "Accounts"
}

func (app *accountsApp) Colours() meta.AppColours {
	return meta.ACCOUNTSCOLOURS
}

func (app *accountsApp) CurrentMotionSet() *meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *accountsApp) CurrentCommandSet() *meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *accountsApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
}

func (app *accountsApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectAccounts()
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD ACCOUNTS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ACCOUNTSAPP,
			Model:     meta.ACCOUNTMODEL,
			Data:      items,
		}
	}
}

func (app *accountsApp) MakeLoadRowsCmd(modelId int) tea.Cmd {
	// Aren't closures just great (still)
	return func() tea.Msg {
		rows, err := database.SelectRowsByAccount(modelId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ACCOUNT ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ACCOUNTSAPP,
			Model:     meta.ENTRYROWMODEL,
			Data:      rows,
		}
	}
}
