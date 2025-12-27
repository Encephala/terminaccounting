package main

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appManager struct {
	width, height int

	activeApp int
	apps      []meta.App
	appIds    map[meta.AppType]int
}

func newAppManager() *appManager {
	apps := make([]meta.App, 4)
	apps[0] = NewEntriesApp()
	apps[1] = NewLedgersApp()
	apps[2] = NewAccountsApp()
	apps[3] = NewJournalsApp()

	// Map the name(=type) of an app to its index in `apps`
	appIds := make(map[meta.AppType]int, 4)
	appIds[meta.ENTRIESAPP] = 0
	appIds[meta.LEDGERSAPP] = 1
	appIds[meta.ACCOUNTSAPP] = 2
	appIds[meta.JOURNALSAPP] = 3

	return &appManager{
		apps:   apps,
		appIds: appIds,
	}
}

func (am *appManager) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	for _, app := range am.apps {
		cmds = append(cmds, app.Init())
	}

	slog.Info("Initialising")

	return tea.Batch(cmds...)
}

func (am *appManager) Update(message tea.Msg) (*appManager, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		am.width = message.Width
		am.height = message.Height

		cmds := am.updateAppsViewSize(message)

		return am, tea.Batch(cmds...)

	case tea.KeyMsg:
		newApp, cmd := am.apps[am.activeApp].Update(message)
		am.apps[am.activeApp] = newApp.(meta.App)

		return am, cmd

	case meta.DataLoadedMsg:
		app := am.appTypeToApp(message.TargetApp)

		acceptedModels := app.AcceptedModels()

		if _, ok := acceptedModels[message.Model]; !ok {
			panic(fmt.Sprintf("Mismatch between target app %q and loaded model:\n%#v", am.appTypeToApp(message.TargetApp).Name(), message))
		}

		newApp, cmd := app.Update(message)
		am.apps[am.appIds[message.TargetApp]] = newApp.(meta.App)

		return am, cmd

	case meta.SwitchTabMsg:
		switch message.Direction {
		case meta.PREVIOUS:
			am.setActiveApp(am.activeApp - 1)
			return am, nil

		case meta.NEXT:
			am.setActiveApp(am.activeApp + 1)
			return am, nil

		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

	case meta.SwitchViewMsg:
		if message.App != nil {
			am.setActiveApp(am.appIds[*message.App])
		}

		newApp, updateCmd := am.apps[am.activeApp].Update(message)
		am.apps[am.activeApp] = newApp.(meta.App)

		windowSizeCmds := am.updateAppsViewSize(tea.WindowSizeMsg{Width: am.width, Height: am.height})

		cmds := append(windowSizeCmds, updateCmd)

		return am, tea.Batch(cmds...)

	case meta.ReloadViewMsg:
		reloadCmd := am.apps[am.activeApp].ReloadView()

		windowSizeCmds := am.updateAppsViewSize(tea.WindowSizeMsg{Width: am.width, Height: am.height})

		notificationCmd := meta.MessageCmd(meta.NotificationMessageMsg{Message: "Refreshed view"})

		cmds := append(windowSizeCmds, reloadCmd, notificationCmd)

		return am, tea.Batch(cmds...)
	}

	var cmds []tea.Cmd
	app, cmd := am.apps[am.activeApp].Update(message)
	am.apps[am.activeApp] = app.(meta.App)
	cmds = append(cmds, cmd)

	return am, tea.Batch(cmds...)
}

func (am *appManager) View() string {
	result := []string{}

	if am.activeApp < 0 || am.activeApp >= len(am.apps) {
		panic(fmt.Sprintf("invalid tab index: %d", am.activeApp))
	}

	tabs := []string{}
	activeTabColour := am.apps[am.activeApp].Colours().Foreground
	for i, app := range am.apps {
		if i == am.activeApp {
			tabs = append(tabs, meta.ActiveTabStyle(activeTabColour).Render(app.Name()))
		} else {
			tabs = append(tabs, meta.TabStyle(activeTabColour).Render(app.Name()))
		}
	}

	// 14 is 12 (width of tab) + 2 (borders)
	numberOfTrailingEmptyCells := am.width - len(am.apps)*14
	if numberOfTrailingEmptyCells >= 0 {
		tabFill := strings.Repeat(" ", numberOfTrailingEmptyCells)
		style := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(activeTabColour)
		tabs = append(tabs, style.Render(tabFill))
	}

	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	result = append(result, tabsRendered)

	result = append(result, am.apps[am.activeApp].View())

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (am *appManager) CurrentMotionSet() meta.MotionSet {
	return am.apps[am.activeApp].CurrentMotionSet()
}

func (am *appManager) CurrentCommandSet() meta.CommandSet {
	return am.apps[am.activeApp].CurrentCommandSet()
}

func (am *appManager) updateAppsViewSize(message tea.WindowSizeMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// -3 for the tabs and their borders
	remainingHeight := message.Height - 3
	for i, app := range am.apps {
		model, cmd := app.Update(tea.WindowSizeMsg{
			Width:  message.Width,
			Height: remainingHeight,
		})
		am.apps[i] = model.(meta.App)
		cmds = append(cmds, cmd)
	}

	return cmds
}

func (am *appManager) appTypeToApp(appType meta.AppType) meta.App {
	return am.apps[am.appIds[appType]]
}

func (am *appManager) setActiveApp(appId int) {
	if appId < 0 {
		am.activeApp = len(am.apps) - 1
	} else if appId >= len(am.apps) {
		am.activeApp = 0
	} else {
		am.activeApp = appId
	}
}
