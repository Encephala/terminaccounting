package main

import (
	"fmt"
	"log/slog"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type LedgersApp struct {
	viewWidth, viewHeight int

	currentView view.View
}

func NewLedgersApp() meta.App {
	model := &LedgersApp{}

	model.currentView = view.NewListView(model)

	return model
}

func (m *LedgersApp) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m *LedgersApp) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(view.View)

		return m, cmd

	case meta.SetupSchemaMsg:
		changed, err := database.SetupSchemaLedgers()
		if err != nil {
			message := fmt.Errorf("COULD NOT CREATE `ledgers` TABLE: %v", err)
			return m, meta.MessageCmd(meta.FatalErrorMsg{Error: message})
		}

		if changed {
			slog.Info("Set up `ledgers` schema")
			return m, nil
		}

		return m, nil

	case meta.DataLoadedMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(view.View)

		return m, cmd

	case meta.SwitchViewMsg:
		switch message.ViewType {
		case meta.LISTVIEWTYPE:
			m.currentView = view.NewListView(m)

		case meta.DETAILVIEWTYPE:
			var ledger database.Ledger
			switch data := message.Data.(type) {
			case database.Ledger:
				ledger = data
			case nil:
				// TODO: I'd like for this to also take the ID from the message.Data,
				// but that's hard to do because the motion is created when the ListView is initialised,
				// but we need to know the selected item later when gd is pressed
				ledger = m.currentView.(*view.ListView).ListModel.SelectedItem().(database.Ledger)

			default:
				panic(fmt.Sprintf("unexpected Data: %#v", data))
			}

			m.currentView = view.NewDetailView(m, ledger.Id, ledger.Name)

		case meta.CREATEVIEWTYPE:
			m.currentView = view.NewLedgersCreateView(m.Colours())

		case meta.UPDATEVIEWTYPE:
			ledgerId := message.Data.(int)

			m.currentView = view.NewLedgersUpdateView(ledgerId, m.Colours())

		case meta.DELETEVIEWTYPE:
			ledgerId := message.Data.(int)

			m.currentView = view.NewLedgersDeleteView(ledgerId, m.Colours())

		default:
			panic(fmt.Sprintf("unexpected meta.ViewType: %#v", message.ViewType))
		}

		return m, m.currentView.Init()

	case meta.SwitchFocusMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(view.View)

		return m, cmd

	case meta.NavigateMsg:
		newView, cmd := m.currentView.Update(message)
		m.currentView = newView.(view.View)

		return m, cmd
	}

	newView, cmd := m.currentView.Update(message)
	m.currentView = newView.(view.View)

	return m, cmd
}

func (m *LedgersApp) View() string {
	style := styles.Body(m.viewWidth, m.viewHeight)

	return style.Render(m.currentView.View())
}

func (m *LedgersApp) Name() string {
	return "Ledgers"
}

func (m *LedgersApp) Colours() styles.AppColours {
	return styles.LEDGERSSTYLES
}

func (m *LedgersApp) CurrentMotionSet() *meta.MotionSet {
	return m.currentView.MotionSet()
}

func (m *LedgersApp) CurrentCommandSet() *meta.CommandSet {
	return m.currentView.CommandSet()
}

func (m *LedgersApp) AcceptedModels() map[meta.ModelType]struct{} {
	return map[meta.ModelType]struct{}{
		meta.LEDGER:   {},
		meta.ENTRYROW: {},
	}
}

func (m *LedgersApp) MakeLoadListCmd() tea.Cmd {
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

func (m *LedgersApp) MakeLoadRowsCmd() tea.Cmd {
	// Aren't closures just great
	ledgerId := m.currentView.(*view.DetailView).ModelId()

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
