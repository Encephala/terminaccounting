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

type JournalsApp struct {
	viewWidth, viewHeight int

	currentView meta.View
}

func NewJournalsApp() meta.App {
	model := &JournalsApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *JournalsApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *JournalsApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.SetupSchemaMsg:
		changed, err := database.SetupSchemaJournals()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `journals` TABLE: %v", err)
			return app, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changed {
			slog.Info("Set up `Journals` schema")
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
			journal := message.Data.(database.Journal)

			app.currentView = view.NewDetailView(app, journal.Id, journal.Name)

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewJournalsCreateView(app.Colours())

		case meta.UPDATEVIEWTYPE:
			// journalId := message.Data.(int)

			// app.currentView = view.NewJournalsUpdateView(journalId, app.Colours())

		case meta.DELETEVIEWTYPE:
			// journalId := message.Data.(int)

			// app.currentView = view.NewJournalsDeleteView(journalId, app.Colours())

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

func (app *JournalsApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *JournalsApp) Name() string {
	return "Journals"
}

func (app *JournalsApp) Colours() meta.AppColours {
	return meta.JOURNALSCOLOURS
}

func (app *JournalsApp) CurrentMotionSet() *meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *JournalsApp) CurrentCommandSet() *meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *JournalsApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.JOURNAL:  {},
		meta.ENTRYROW: {},
	}
}

func (app *JournalsApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectJournals()
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.JOURNALS,
			Model:     meta.JOURNAL,
			Data:      items,
		}
	}
}

func (app *JournalsApp) MakeLoadRowsCmd(journalId int) tea.Cmd {
	// Aren't closures just great
	return func() tea.Msg {
		rows, err := database.SelectRowsByJournal(journalId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.JOURNALS,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
