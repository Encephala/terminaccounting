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

type ledgersApp struct {
	viewWidth, viewHeight int

	currentView meta.View
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

		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.SetupSchemaMsg:
		changed, err := database.SetupSchemaLedgers()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `ledgers` TABLE: %v", err)
			return app, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changed {
			slog.Info("Set up `ledgers` schema")
			return app, nil
		}

		return app, nil

	case meta.DataLoadedMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.SwitchViewMsg:
		if message.App != nil && *message.App != meta.LEDGERS {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			ledger := message.Data.(database.Ledger)

			app.currentView = view.NewDetailView(app, ledger.Id, ledger.Name)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewLedgersCreateView(app.Colours())

		case meta.UPDATEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersUpdateView(ledgerId, app.Colours())

		case meta.DELETEVIEWTYPE:
			ledgerId := message.Data.(int)

			app.currentView = view.NewLedgersDeleteView(ledgerId, app.Colours())

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()

	case meta.SwitchFocusMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.NavigateMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd
	}

	newView, cmd := app.currentView.Update(message)
	app.currentView = newView.(meta.View)

	return app, cmd
}

func (app *ledgersApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *ledgersApp) Name() string {
	return "Ledgers"
}

func (app *ledgersApp) Colours() meta.AppColours {
	return meta.LEDGERSCOLOURS
}

func (app *ledgersApp) CurrentMotionSet() *meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *ledgersApp) CurrentCommandSet() *meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *ledgersApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGER:   {},
		meta.ENTRYROW: {},
	}
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
			TargetApp: meta.LEDGERS,
			Model:     meta.LEDGER,
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
			TargetApp: meta.LEDGERS,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
