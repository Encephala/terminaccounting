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
	viewWidth, viewHeight int

	activeApp int
	apps      []meta.App
	appIds    map[meta.AppType]int
}

func (am *appManager) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	for _, app := range am.apps {
		cmds = append(cmds, app.Init())
	}

	for i, app := range am.apps {
		model, cmd := app.Update(meta.SetupSchemaMsg{})
		am.apps[i] = model.(meta.App)
		cmds = append(cmds, cmd)
	}

	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(am.apps[am.activeApp].CurrentMotionSet())))
	cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(am.apps[am.activeApp].CurrentCommandSet())))

	slog.Info("Initialised")

	return tea.Batch(cmds...)
}

func (am *appManager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch message := message.(type) {
	case tea.WindowSizeMsg:
		am.viewWidth = message.Width
		am.viewHeight = message.Height

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
			return am.setActiveApp(am.activeApp - 1)
		case meta.NEXT:
			return am.setActiveApp(am.activeApp + 1)
		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

	case meta.SwitchViewMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		if message.App != nil {
			am, cmd = am.setActiveApp(am.appIds[*message.App])
			cmds = append(cmds, cmd)
		}

		newApp, cmd := am.apps[am.activeApp].Update(message)
		am.apps[am.activeApp] = newApp.(meta.App)
		cmds = append(cmds, cmd)

		cmds = append(cmds, meta.MessageCmd(meta.UpdateViewMotionSetMsg(am.apps[am.activeApp].CurrentMotionSet())))
		cmds = append(cmds, meta.MessageCmd(meta.UpdateViewCommandSetMsg(am.apps[am.activeApp].CurrentCommandSet())))

		return am, tea.Batch(cmds...)
	}

	app, cmd := am.apps[am.activeApp].Update(message)
	am.apps[am.activeApp] = app.(meta.App)
	cmds = append(cmds, cmd)

	return am, tea.Batch(cmds...)
}

func (m *appManager) View() string {
	result := []string{}

	if m.activeApp < 0 || m.activeApp >= len(m.apps) {
		panic(fmt.Sprintf("invalid tab index: %d", m.activeApp))
	}

	tabs := []string{}
	activeTabColour := m.apps[m.activeApp].Colours().Foreground
	for i, app := range m.apps {
		if i == m.activeApp {
			tabs = append(tabs, meta.ActiveTabStyle(activeTabColour).Render(app.Name()))
		} else {
			tabs = append(tabs, meta.TabStyle(activeTabColour).Render(app.Name()))
		}
	}

	// 14 is 12 (width of tab) + 2 (borders)
	numberOfTrailingEmptyCells := m.viewWidth - len(m.apps)*14
	if numberOfTrailingEmptyCells >= 0 {
		tabFill := strings.Repeat(" ", numberOfTrailingEmptyCells)
		style := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(activeTabColour)
		tabs = append(tabs, style.Render(tabFill))
	}

	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	result = append(result, tabsRendered)

	result = append(result, m.apps[m.activeApp].View())

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (m *appManager) appTypeToApp(appType meta.AppType) meta.App {
	return m.apps[m.appIds[appType]]
}

func (m *appManager) setActiveApp(appId int) (*appManager, tea.Cmd) {
	if appId < 0 {
		m.activeApp = len(m.apps) - 1
	} else if appId >= len(m.apps) {
		m.activeApp = 0
	} else {
		m.activeApp = appId
	}

	cmd := meta.MessageCmd(meta.UpdateViewMotionSetMsg(m.apps[m.activeApp].CurrentMotionSet()))
	cmdTwo := meta.MessageCmd(meta.UpdateViewCommandSetMsg(m.apps[m.activeApp].CurrentCommandSet()))

	return m, tea.Batch(cmd, cmdTwo)
}
