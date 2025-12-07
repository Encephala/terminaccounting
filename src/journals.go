package main

import (
	"fmt"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type journalsApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewJournalsApp() meta.App {
	model := &journalsApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *journalsApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *journalsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(view.View)

		return app, cmd

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
			journal := message.Data.(database.Journal)

			app.currentView = view.NewJournalsDetailsView(journal, app)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewJournalsCreateView(app.Colours())

		case meta.UPDATEVIEWTYPE:
			journalId := message.Data.(int)

			app.currentView = view.NewJournalsUpdateView(journalId, app.Colours())

		case meta.DELETEVIEWTYPE:
			journalId := message.Data.(int)

			app.currentView = view.NewJournalsDeleteView(journalId, app.Colours())

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

func (app *journalsApp) Colours() meta.AppColours {
	return meta.JOURNALSCOLOURS
}

func (app *journalsApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *journalsApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *journalsApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
}

func (app *journalsApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectJournals()
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

func (app *journalsApp) MakeLoadRowsCmd(journalId int) tea.Cmd {
	panic("Journals has its own detailsview, this shouldn't get called")
}
