package main

import (
	"fmt"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func (app *accountsApp) Update(message tea.Msg) (meta.App, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.SwitchAppViewMsg:
		if message.App != nil && *message.App != meta.LEDGERSAPP {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			account := message.Data.(database.Account)

			app.currentView = view.NewAccountsDetailView(account.Id)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewAccountsCreateView()

		case meta.UPDATEVIEWTYPE:
			accountId := message.Data.(int)

			app.currentView = view.NewAccountsUpdateView(accountId)

		case meta.DELETEVIEWTYPE:
			accountId := message.Data.(int)

			app.currentView = view.NewAccountsDeleteView(accountId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()
	}

	var cmd tea.Cmd
	app.currentView, cmd = app.currentView.Update(message)

	return app, cmd
}

func (app *accountsApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *accountsApp) Name() string {
	return "Accounts"
}

func (app *accountsApp) Type() meta.AppType {
	return meta.ACCOUNTSAPP
}

func (app *accountsApp) Colour() lipgloss.Color {
	return meta.ACCOUNTSCOLOUR
}

func (app *accountsApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *accountsApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *accountsApp) CurrentViewAllowsInsertMode() bool {
	return app.currentView.AllowsInsertMode()
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

func (app *accountsApp) ReloadView() tea.Cmd {
	app.currentView = app.currentView.Reload()

	return app.currentView.Init()
}
