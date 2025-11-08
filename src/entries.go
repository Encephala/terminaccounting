package main

import (
	"fmt"
	"log/slog"
	"strconv"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type entriesApp struct {
	viewWidth, viewHeight int

	currentView meta.View
}

func NewEntriesApp() meta.App {
	model := &entriesApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *entriesApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *entriesApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.SetupSchemaMsg:
		changedEntries, err := database.SetupSchemaEntries()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entries` TABLE: %v", err)
			return app, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		changedEntryRows, err := database.SetupSchemaEntryRows()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `entryrows` TABLE: %v", err)
			return app, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changedEntries || changedEntryRows {
			slog.Info("Set up `Entries` schema")
			return app, nil
		}

		return app, nil

	case meta.DataLoadedMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.NavigateMsg:
		newView, cmd := app.currentView.Update(message)
		app.currentView = newView.(meta.View)

		return app, cmd

	case meta.SwitchViewMsg:
		if message.App != nil && *message.App != meta.ENTRIES {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			entry := message.Data.(database.Entry)

			// No better model name to be had than the entry Id
			app.currentView = view.NewDetailView(app, entry.Id, strconv.Itoa(entry.Id))

		case meta.CREATEVIEWTYPE:
			app.currentView = view.NewEntryCreateView(app.Colours())

		case meta.UPDATEVIEWTYPE:
			entryId := message.Data.(int)

			app.currentView = view.NewEntryUpdateView(entryId, app.Colours())

		case meta.DELETEVIEWTYPE:
			entryId := message.Data.(int)

			app.currentView = view.NewEntryDeleteView(entryId, app.Colours())

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()
	}

	newView, cmd := app.currentView.Update(message)
	app.currentView = newView.(meta.View)

	return app, cmd
}

func (app *entriesApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *entriesApp) Name() string {
	return "Entries"
}

func (app *entriesApp) Colours() meta.AppColours {
	return meta.ENTRIESCOLOURS
}

func (app *entriesApp) CurrentMotionSet() *meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *entriesApp) CurrentCommandSet() *meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *entriesApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.ENTRY:    {},
		meta.ENTRYROW: {},
		meta.JOURNAL:  {},
		meta.LEDGER:   {},
		meta.ACCOUNT:  {},
	}
}

func (app *entriesApp) MakeLoadListCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := database.SelectEntries()
		if err != nil {
			return meta.MessageCmd(fmt.Errorf("FAILED TO LOAD ENTRIES: %v", err))
		}

		items := make([]list.Item, len(rows))
		for i, row := range rows {
			items[i] = row
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRY,
			Data:      items,
		}
	}
}

func (app *entriesApp) MakeLoadRowsCmd(entryId int) tea.Cmd {
	// Aren't closures just great
	return func() tea.Msg {
		rows, err := database.SelectRowsByEntry(entryId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ENTRY ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ENTRIES,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
