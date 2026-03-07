package main

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/apps"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/jmoiron/sqlx"
)

type appManager struct {
	width, height    int
	xscroll, yscroll int

	activeApp int
	apps      []meta.App
	appIds    map[meta.AppType]int
}

func newAppManager(DB *sqlx.DB) *appManager {
	a := make([]meta.App, 4)
	a[0] = apps.NewEntriesApp(DB)
	a[1] = apps.NewLedgersApp(DB)
	a[2] = apps.NewAccountsApp(DB)
	a[3] = apps.NewJournalsApp(DB)

	// Map the name(=type) of an app to its index in `apps`
	appIds := make(map[meta.AppType]int, 4)
	appIds[meta.ENTRIESAPP] = 0
	appIds[meta.LEDGERSAPP] = 1
	appIds[meta.ACCOUNTSAPP] = 2
	appIds[meta.JOURNALSAPP] = 3

	return &appManager{
		apps:   a,
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
		var cmd tea.Cmd
		am.apps[am.activeApp], cmd = am.apps[am.activeApp].Update(message)

		return am, cmd

	case meta.DataLoadedMsg:
		app := am.appTypeToApp(message.TargetApp)

		acceptedModels := app.AcceptedModels()

		if _, ok := acceptedModels[message.Model]; !ok {
			appName := am.appTypeToApp(message.TargetApp).Name()
			viewType := app.CurrentViewType()
			message := fmt.Sprintf("Mismatch between target app %q (%q) and loaded model:\n%#v", appName, viewType, message)
			panic(message)
		}

		app, cmd := app.Update(message)

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

	case meta.ScrollVerticalMsg:
		// -3 for top tabs, -3 for the title, -1 to leave some bottom padding
		// TODO: This should scroll to the content length, not the body height minus stuff
		bodyHeight := am.height - 3 - 3 - 1

		switch {
		case !message.Up && !message.ToEnd:
			am.yscroll = min(am.yscroll+1, bodyHeight-1)
		case !message.Up && message.ToEnd:
			am.yscroll = bodyHeight - 1
		case message.Up && !message.ToEnd:
			am.yscroll = max(am.yscroll-1, 0)
		case message.Up && message.ToEnd:
			am.yscroll = 0
		}

		return am, nil

	case meta.ScrollHorizontalMsg:
		// -4 for horizontal padding
		// TODO: This should scroll to the content width, not the body width minus stuff
		bodyWidth := am.width - 4

		switch {
		case !message.Left && !message.ToEnd:
			am.xscroll = min(am.xscroll+1, bodyWidth-1)
		case !message.Left && message.ToEnd:
			am.xscroll = bodyWidth - 1
		case message.Left && !message.ToEnd:
			am.xscroll = max(am.xscroll-1, 0)
		case message.Left && message.ToEnd:
			am.xscroll = 0
		}

		return am, nil

	case meta.SwitchAppViewMsg:
		if message.App != nil {
			am.setActiveApp(am.appIds[*message.App])
		}

		var updateCmd tea.Cmd
		am.apps[am.activeApp], updateCmd = am.apps[am.activeApp].Update(message)

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

	var cmd tea.Cmd
	am.apps[am.activeApp], cmd = am.apps[am.activeApp].Update(message)

	return am, cmd
}

func (am *appManager) View() string {
	// Render tabs
	if am.activeApp < 0 || am.activeApp >= len(am.apps) {
		panic(fmt.Sprintf("invalid tab index: %d", am.activeApp))
	}

	tabs := []string{}
	activeTabColour := am.apps[am.activeApp].Colour()
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

		style := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(activeTabColour)

		tabs = append(tabs, style.Render(tabFill))
	}

	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)

	// Render title
	titleRendered := lipgloss.NewStyle().
		Margin(1, 2).
		Render(am.apps[am.activeApp].CurrentTitle())

	// Render body
	body := am.apps[am.activeApp].View()

	var bodyLines []string
	for i, line := range strings.Split(body, "\n") {
		// Apply vertical scrolling
		if i < am.yscroll {
			continue
		}

		// Apply horizontal scrolling
		line = ansi.TruncateLeft(line, am.xscroll, "")

		bodyLines = append(bodyLines, line)
	}

	bodyRendered := lipgloss.NewStyle().
		// -4 for vertical horizontal margin
		Width(am.width-4).
		// -3 for the tabs, -3 for the title
		Height(am.height-3-3).
		Margin(0, 2).
		Render(strings.Join(bodyLines, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, tabsRendered, titleRendered, bodyRendered)
}

func (am *appManager) CurrentMotionSet() meta.MotionSet {
	result := am.apps[am.activeApp].CurrentMotionSet()

	result.Normal.Insert([]string{"z", "j"}, meta.ScrollVerticalMsg{Up: false})
	result.Normal.Insert([]string{"z", "J"}, meta.ScrollVerticalMsg{Up: false, ToEnd: true})
	result.Normal.Insert([]string{"z", "k"}, meta.ScrollVerticalMsg{Up: true})
	result.Normal.Insert([]string{"z", "K"}, meta.ScrollVerticalMsg{Up: true, ToEnd: true})
	result.Normal.Insert([]string{"z", "l"}, meta.ScrollHorizontalMsg{Left: false})
	result.Normal.Insert([]string{"z", "L"}, meta.ScrollHorizontalMsg{Left: false, ToEnd: true})
	result.Normal.Insert([]string{"z", "h"}, meta.ScrollHorizontalMsg{Left: true})
	result.Normal.Insert([]string{"z", "H"}, meta.ScrollHorizontalMsg{Left: true, ToEnd: true})

	return result
}

func (am *appManager) CurrentCommandSet() meta.CommandSet {
	return am.apps[am.activeApp].CurrentCommandSet()
}

func (am *appManager) currentViewAllowsInsertMode() bool {
	return am.apps[am.activeApp].CurrentViewAllowsInsertMode()
}

func (am *appManager) currentViewType() meta.ViewType {
	return am.apps[am.activeApp].CurrentViewType()
}

func (am *appManager) updateAppsViewSize(message tea.WindowSizeMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// -4 for horizontal margins
	remainingWidth := message.Width - 4
	// -3 for the tabs, -3 for the title, -1 for bottom margin
	remainingHeight := message.Height - 3 - 3 - 1

	for i, app := range am.apps {
		var cmd tea.Cmd
		am.apps[i], cmd = app.Update(tea.WindowSizeMsg{
			Width:  remainingWidth,
			Height: remainingHeight,
		})

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
