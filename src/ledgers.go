package main

import (
	"fmt"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type ledgersApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewLedgersApp() meta.App {
	model := &ledgersApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *ledgersApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *ledgersApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.DataLoadedMsg:
		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.SwitchViewMsg:
		if message.App != nil && *message.App != meta.LEDGERSAPP {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			ledger := message.Data.(database.Ledger)

			app.currentView = view.NewDetailView(app, ledger.Id, ledger.Name)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewLedgersCreateView()

		case meta.UPDATEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersUpdateView(ledgerId)

		case meta.DELETEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersDeleteView(ledgerId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()

	case meta.SwitchFocusMsg:
		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.NavigateMsg:
		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd
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

func (app *ledgersApp) Colours() meta.AppColours {
	return meta.LEDGERSCOLOURS
}

func (app *ledgersApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *ledgersApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *ledgersApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
}

func (app *ledgersApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectLedgers()
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

func (app *ledgersApp) MakeLoadRowsCmd(ledgerId int) tea.Cmd {
	// Aren't closures just great
	return func() tea.Msg {
		rows, err := database.SelectRowsByLedger(ledgerId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.LEDGERSAPP,
			Model:     meta.ENTRYROWMODEL,
			Data:      rows,
		}
	}
}

func (app *ledgersApp) ReloadView() tea.Cmd {
	app.currentView = app.currentView.Reload()

	return app.currentView.Init()
}
