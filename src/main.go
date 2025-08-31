package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

func main() {
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	file, err := os.OpenFile("debug.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		slog.Error("Couldn't create logger: ", "error", err)
		os.Exit(1)
	}
	defer file.Close()
	log.SetOutput(file)

	db, err := sqlx.Connect("sqlite3", "file:test.db?cache=shared&mode=rwc")
	if err != nil {
		slog.Error("Couldn't connect to database: ", "error", err)
		os.Exit(1)
	}
	database.DB = db

	commandInput := textinput.New()
	commandInput.Cursor.SetMode(cursor.CursorStatic)
	commandInput.Prompt = ":"

	motionSet := meta.CompleteMotionSet{GlobalMotionSet: meta.GlobalMotions()}
	commandSet := meta.CompleteCommandSet{GlobalCommandSet: meta.GlobalCommands()}

	apps := make([]meta.App, 2)
	apps[0] = NewLedgersApp()
	apps[1] = NewEntriesApp()
	// Commented while I'm refactoring a lot, to avoid having to reimplement various interfaces etc.
	// apps[meta.JOURNALS] = journals.New()
	// apps[meta.ACCOUNTS] = accounts.New()

	// Map the name(=type) of an app to its index in `apps`
	appIds := make(map[meta.AppType]int, 2)
	appIds[meta.LEDGERS] = 0
	appIds[meta.ENTRIES] = 1

	m := &model{
		activeApp: 0,
		apps:      apps,
		appIds:    appIds,

		inputMode:    meta.NORMALMODE,
		commandInput: commandInput,

		currentMotion: make(meta.Motion, 0),
		motionSet:     motionSet,

		commandSet: commandSet,
	}

	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		message := fmt.Sprintf("Bubbletea error: %v", err)
		slog.Error(message)
		fmt.Println(message)
		os.Exit(1)
	}

	err = finalModel.(*model).fatalError
	if err != nil {
		message := fmt.Sprintf("Program exited with fatal error: %v", err)
		fmt.Println(message)
		os.Exit(1)
	}

	slog.Info("Exited gracefully")
	os.Exit(0)
}

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	for _, app := range m.apps {
		cmds = append(cmds, app.Init())
	}

	for i, app := range m.apps {
		model, cmd := app.Update(meta.SetupSchemaMsg{})
		m.apps[i] = model.(meta.App)
		cmds = append(cmds, cmd)
	}

	m.motionSet.ViewMotionSet = m.apps[m.activeApp].CurrentMotionSet()

	slog.Info("Initialised")

	return tea.Batch(cmds...)
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch message := message.(type) {
	case tea.QuitMsg:
		return m, tea.Quit

	case error:
		slog.Debug(fmt.Sprintf("Error: %v", message))
		m.displayedError = message
		return m, meta.ClearErrorAfterDelayCmd

	case meta.ClearErrorMsg:
		m.displayedError = nil
		return m, nil

	case meta.FatalErrorMsg:
		slog.Error(fmt.Sprintf("Fatal error: %v", message.Error))
		m.fatalError = message.Error
		return m, tea.Quit

	case meta.CommandMsg:
		return m, meta.MessageCmd(message(m.apps[m.activeApp]))

	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

		// -3 for the tabs and their borders
		// -1 for the status line
		remainingHeight := message.Height - 3 - 1
		for i, app := range m.apps {
			model, cmd := app.Update(tea.WindowSizeMsg{
				Width:  message.Width,
				Height: remainingHeight,
			})
			m.apps[i] = model.(meta.App)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKeyMsg(message)

	case meta.DataLoadedMsg:
		acceptedModels := m.appTypeToApp(message.TargetApp).AcceptedModels()

		if _, ok := acceptedModels[message.Model]; !ok {
			panic(fmt.Sprintf("Mismatch between target app %q and loaded model:\n%#v", m.appTypeToApp(message.TargetApp).Name(), message))
		}

		newApp, cmd := m.appTypeToApp(message.TargetApp).Update(message)
		m.apps[m.appIds[message.TargetApp]] = newApp.(meta.App)

		return m, cmd

	case meta.UpdateViewMotionSetMsg:
		m.motionSet.ViewMotionSet = message

		return m, nil

	case meta.UpdateViewCommandSetMsg:
		m.commandSet.ViewCommandSet = message

		return m, nil

	case meta.SwitchTabMsg:
		switch message.Direction {
		case meta.PREVIOUS:
			return m.setActiveApp(m.activeApp - 1)
		case meta.NEXT:
			return m.setActiveApp(m.activeApp + 1)
		default:
			panic(fmt.Sprintf("unexpected meta.Direction: %#v", message.Direction))
		}

	case meta.SwitchViewMsg:
		var cmd tea.Cmd
		if message.App != nil {
			m, cmd = m.setActiveApp(m.appIds[*message.App])
		}

		var cmdTwo tea.Cmd
		newApp, cmdTwo := m.apps[m.activeApp].Update(message)
		m.apps[m.activeApp] = newApp.(meta.App)

		return m, tea.Batch(cmd, cmdTwo)

	case meta.SwitchModeMsg:
		m.switchMode(message.InputMode)

		return m, nil

	case meta.ExecuteCommandMsg:
		command := m.commandInput.Value()

		return m.executeCommand(command)
	}

	app, cmd := m.apps[m.activeApp].Update(message)
	m.apps[m.activeApp] = app.(meta.App)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	result := []string{}

	if m.activeApp < 0 || m.activeApp >= len(m.apps) {
		panic(fmt.Sprintf("invalid tab index: %d", m.activeApp))
	}

	tabs := []string{}
	activeTabColour := m.apps[m.activeApp].Colours().Foreground
	for i, app := range m.apps {
		if i == m.activeApp {
			tabs = append(tabs, styles.ActiveTab(activeTabColour).Render(app.Name()))
		} else {
			tabs = append(tabs, styles.Tab(activeTabColour).Render(app.Name()))
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

	result = append(result, statusLineView(m))

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (m *model) appTypeToApp(appType meta.AppType) meta.App {
	return m.apps[m.appIds[appType]]
}

func (m *model) handleKeyMsg(message tea.KeyMsg) (*model, tea.Cmd) {
	// ctrl+c to reset the current motion can't be handled as a motion itself,
	// because then for instance ["g", "ctrl+c"] would be recognised as an invalid motion
	if m.inputMode == meta.NORMALMODE && message.Type == tea.KeyCtrlC {
		m.resetCurrentMotion()

		return m, nil
	}

	m.currentMotion = append(m.currentMotion, message.String())

	if !m.motionSet.ContainsPath(m.inputMode, m.currentMotion) {
		switch m.inputMode {
		case meta.NORMALMODE:
			cmd := meta.MessageCmd(fmt.Errorf("invalid motion: %s", m.currentMotion.View()))

			m.resetCurrentMotion()

			return m, cmd

		// In INSERT and COMMAND mode, a key stroke that isn't a motion gets sent to the appropriate input
		case meta.INSERTMODE:
			newApp, cmd := m.apps[m.activeApp].Update(message)
			m.apps[m.activeApp] = newApp.(meta.App)

			m.resetCurrentMotion()

			return m, cmd

		case meta.COMMANDMODE:
			var cmd tea.Cmd
			m.commandInput, cmd = m.commandInput.Update(message)

			m.resetCurrentMotion()

			return m, cmd
		}
	}

	completedMotionMsg, ok := m.motionSet.Get(m.inputMode, m.currentMotion)
	if !ok {
		// The currentMotion is the start of an existing motion, wait for more inputs
		return m, nil
	}

	m.resetCurrentMotion()

	newModel, cmd := m.Update(completedMotionMsg)
	m = newModel.(*model)

	return m, cmd
}

func (m *model) setActiveApp(appId int) (*model, tea.Cmd) {
	if appId < 0 {
		m.activeApp = len(m.apps) - 1
	} else if appId >= len(m.apps) {
		m.activeApp = 0
	} else {
		m.activeApp = appId
	}

	newModel, cmd := m.Update(meta.UpdateViewMotionSetMsg(m.apps[m.activeApp].CurrentMotionSet()))
	newModel, cmdTwo := newModel.Update(meta.UpdateViewCommandSetMsg(m.apps[m.activeApp].CurrentCommandSet()))

	return newModel.(*model), tea.Batch(cmd, cmdTwo)
}

func (m *model) resetCurrentMotion() {
	m.currentMotion = m.currentMotion[:0]
}

func (m *model) switchMode(newMode meta.InputMode) {
	if m.inputMode == meta.COMMANDMODE {
		m.commandInput.Reset()
		m.commandInput.Blur()
	}

	m.inputMode = newMode

	if newMode == meta.COMMANDMODE {
		m.commandInput.Focus()
	}
}

func (m *model) executeCommand(command string) (*model, tea.Cmd) {
	commandMsg, ok := m.commandSet.Get(strings.Split(command, ""))

	if !ok {
		cmd := meta.MessageCmd(fmt.Errorf("invalid command: %v", m.commandInput.Value()))

		m.switchMode(meta.NORMALMODE)

		return m, cmd
	}

	newModel, cmd := m.Update(commandMsg)
	m = newModel.(*model)

	m.switchMode(meta.NORMALMODE)

	return m, cmd
}
