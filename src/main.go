package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"terminaccounting/apps/entries"
	"terminaccounting/apps/ledgers"
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

	commandInput := textinput.New()
	commandInput.Cursor.SetMode(cursor.CursorStatic)
	commandInput.Prompt = ":"

	motionSet := meta.CompleteMotionSet{GlobalMotionSet: meta.GlobalMotions()}
	commandSet := meta.CompleteCommandSet{GlobalCommandSet: meta.GlobalCommands()}

	m := &model{
		db: db,

		activeApp: 0,
		apps: []meta.App{
			// Commented while I'm refactoring a lot, to avoid having to reimplement various interfaces etc.
			ledgers.New(db),
			// accounts.New(),
			// journals.New(),
			entries.New(db),
		},

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
		model, cmd := app.Update(meta.SetupSchemaMsg{Db: m.db})
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
		for i, app := range m.apps {
			if app.Name() == message.TargetApp {
				newApp, cmd := app.Update(message)
				m.apps[i] = newApp.(meta.App)
				return m, cmd
			}
		}

		return m, nil

	case meta.UpdateViewMotionSetMsg:
		m.motionSet.ViewMotionSet = message

		return m, nil

	case meta.UpdateViewCommandSetMsg:
		m.commandSet.ViewCommandSet = message

		return m, nil

	case meta.SwitchTabMsg:
		return m.handleTabSwitch(message.Direction)

	case meta.SwitchModeMsg:
		m.switchMode(message.InputMode)

		return m, nil

	case meta.ExecuteCommandMsg:
		return m.executeCommand()
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
			style := styles.ActiveTab(activeTabColour)
			tabs = append(tabs, style.Render(app.Name()))
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

func (m *model) handleKeyMsg(message tea.KeyMsg) (*model, tea.Cmd) {
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
		return m, nil
	}

	m.resetCurrentMotion()

	newModel, cmd := m.Update(completedMotionMsg)
	m = newModel.(*model)

	return m, cmd
}

func (m *model) handleTabSwitch(direction meta.Sequence) (*model, tea.Cmd) {
	switch direction {
	case meta.PREVIOUS:
		m.activeApp = (m.activeApp - 1)
		if m.activeApp < 0 {
			m.activeApp += len(m.apps)
		}

	case meta.NEXT:
		m.activeApp = (m.activeApp + 1) % len(m.apps)

	default:
		panic(fmt.Sprintf("unexpected meta.Sequence: %#v", direction))
	}

	newModel, cmd := m.Update(meta.UpdateViewMotionSetMsg(m.apps[m.activeApp].CurrentMotionSet()))
	newModel, cmdTwo := newModel.Update(meta.UpdateViewCommandSetMsg(m.apps[m.activeApp].CurrentCommandSet()))
	m = newModel.(*model)

	return m, tea.Batch(cmd, cmdTwo)
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

func (m *model) executeCommand() (*model, tea.Cmd) {
	completedCommandMsg, ok := m.commandSet.Get(strings.Split(m.commandInput.Value(), ""))

	if !ok {
		cmd := meta.MessageCmd(fmt.Errorf("invalid command: %v", m.commandInput.Value()))

		m.switchMode(meta.NORMALMODE)

		return m, cmd
	}

	switch completedCommandMsg.Type {
	case meta.QUIT:
		return m, tea.Quit
	}

	newApp, cmd := m.apps[m.activeApp].Update(completedCommandMsg)
	m.apps[m.activeApp] = newApp.(meta.App)

	m.switchMode(meta.NORMALMODE)

	return m, cmd
}
