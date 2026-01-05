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

type journalsApp struct {
	DB *sqlx.DB

	viewWidth, viewHeight int

	currentView view.View
}

func NewJournalsApp(DB *sqlx.DB) meta.App {
	model := &journalsApp{DB: DB}

	model.currentView = view.NewListView(model)

	return model
}

func (app *journalsApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *journalsApp) Update(message tea.Msg) (meta.App, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.SwitchAppViewMsg:
		if message.App != nil && *message.App != meta.JOURNALSAPP {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			journal := message.Data.(database.Journal)

			app.currentView = view.NewJournalsDetailsView(app.DB, journal, app)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewJournalsCreateView(app.DB)

		case meta.UPDATEVIEWTYPE:
			journalId := message.Data.(int)

			app.currentView = view.NewJournalsUpdateView(app.DB, journalId)

		case meta.DELETEVIEWTYPE:
			journalId := message.Data.(int)

			app.currentView = view.NewJournalsDeleteView(app.DB, journalId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()
	}

	var cmd tea.Cmd
	app.currentView, cmd = app.currentView.Update(message)

	return app, cmd
}

func (app *journalsApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *journalsApp) Name() string {
	return "Journals"
}

func (app *journalsApp) Type() meta.AppType {
	return meta.JOURNALSAPP
}

func (app *journalsApp) CurrentViewType() meta.ViewType {
	return app.currentView.Type()
}

func (app *journalsApp) Colour() lipgloss.Color {
	return meta.JOURNALSCOLOUR
}

func (app *journalsApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *journalsApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *journalsApp) CurrentViewAllowsInsertMode() bool {
	return app.currentView.AllowsInsertMode()
}

func (app *journalsApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
}

func (app *journalsApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectJournals(app.DB)
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD JOURNALS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.JOURNALSAPP,
			Model:     meta.JOURNALMODEL,
			Data:      items,
		}
	}
}

func (app *journalsApp) ReloadView() tea.Cmd {
	app.currentView = app.currentView.Reload()

	return app.currentView.Init()
}
