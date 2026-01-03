package main

import (
	"fmt"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type ledgersApp struct {
	DB *sqlx.DB

	viewWidth, viewHeight int

	currentView view.View
}

func NewLedgersApp(DB *sqlx.DB) meta.App {
	model := &ledgersApp{DB: DB}

	model.currentView = view.NewListView(model)

	return model
}

func (app *ledgersApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *ledgersApp) Update(message tea.Msg) (meta.App, tea.Cmd) {
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
			ledger := message.Data.(database.Ledger)

			app.currentView = view.NewLedgersDetailView(app.DB, ledger.Id)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewLedgersCreateView(app.DB)

		case meta.UPDATEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersUpdateView(app.DB, ledgerId)

		case meta.DELETEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersDeleteView(app.DB, ledgerId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()
	}

	var cmd tea.Cmd
	app.currentView, cmd = app.currentView.Update(message)

	return app, cmd
}

func (app *ledgersApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *ledgersApp) Name() string {
	return "Ledgers"
}

func (app *ledgersApp) Type() meta.AppType {
	return meta.LEDGERSAPP
}

func (app *ledgersApp) CurrentViewType() meta.ViewType {
	return app.currentView.Type()
}

func (app *ledgersApp) Colour() lipgloss.Color {
	return meta.LEDGERSCOLOUR
}

func (app *ledgersApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *ledgersApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *ledgersApp) CurrentViewAllowsInsertMode() bool {
	return app.currentView.AllowsInsertMode()
}

func (app *ledgersApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
}

func (app *ledgersApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectLedgers(app.DB)
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.LEDGERSAPP,
			Model:     meta.LEDGERMODEL,
			Data:      items,
		}
	}
}

func (app *ledgersApp) ReloadView() tea.Cmd {
	app.currentView = app.currentView.Reload()

	return app.currentView.Init()
}
