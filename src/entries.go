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

type entriesApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewEntriesApp() meta.App {
	model := &entriesApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (app *entriesApp) Init() tea.Cmd {
	return app.currentView.Init()
}

func (app *entriesApp) Update(message tea.Msg) (meta.App, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		app.viewWidth = message.Width
		app.viewHeight = message.Height

		var cmd tea.Cmd
		app.currentView, cmd = app.currentView.Update(message)

		return app, cmd

	case meta.SwitchAppViewMsg:
		if message.App != nil && *message.App != meta.ENTRIESAPP {
			panic("wrong app type, something went wrong")
		}

		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			app.currentView = view.NewListView(app)

		case meta.DETAILVIEWTYPE:
			entry := message.Data.(database.Entry)

			// No better model name to be had than the entry Id
			app.currentView = view.NewEntriesDetailView(entry.Id)

		case meta.CREATEVIEWTYPE:
			if message.Data != nil {
				newView, err := view.NewEntryCreateViewPrefilled(message.Data.(view.EntryPrefillData))
				if err != nil {
					return app, meta.MessageCmd(err)
				}
				app.currentView = newView
			} else {
				app.currentView = view.NewEntryCreateView()
			}

		case meta.UPDATEVIEWTYPE:
			entryId := message.Data.(int)

			app.currentView = view.NewEntryUpdateView(entryId)

		case meta.DELETEVIEWTYPE:
			entryId := message.Data.(int)

			app.currentView = view.NewEntryDeleteView(entryId)

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return app, app.currentView.Init()
	}

	var cmd tea.Cmd
	app.currentView, cmd = app.currentView.Update(message)

	return app, cmd
}

func (app *entriesApp) View() string {
	style := meta.BodyStyle(app.viewWidth, app.viewHeight)

	return style.Render(app.currentView.View())
}

func (app *entriesApp) Name() string {
	return "Entries"
}

func (app *entriesApp) Type() meta.AppType {
	return meta.ENTRIESAPP
}

func (app *entriesApp) Colour() lipgloss.Color {
	return meta.ENTRIESCOLOUR
}

func (app *entriesApp) CurrentMotionSet() meta.MotionSet {
	return app.currentView.MotionSet()
}

func (app *entriesApp) CurrentCommandSet() meta.CommandSet {
	return app.currentView.CommandSet()
}

func (app *entriesApp) CurrentViewAllowsInsertMode() bool {
	return app.currentView.AllowsInsertMode()
}

func (app *entriesApp) AcceptedModels() map[meta.ModelType]struct{} {
	return app.currentView.AcceptedModels()
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
			TargetApp: meta.ENTRIESAPP,
			Model:     meta.ENTRYMODEL,
			Data:      items,
		}
	}
}

func (app *entriesApp) ReloadView() tea.Cmd {
	app.currentView = app.currentView.Reload()

	return app.currentView.Init()
}
