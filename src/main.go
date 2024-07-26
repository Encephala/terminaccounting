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
	"terminaccounting/utils"
	"terminaccounting/vim"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
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
	commandInput.Prompt = ":"

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

		inputMode:    vim.NORMALMODE,
		commandInput: commandInput,

		currentMotion: make(vim.Motion, 0),
		motions:       vim.Motions(),
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
		message := fmt.Sprintf("Fatal error: %v", err)
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

	slog.Info("Initialised")

	return tea.Batch(cmds...)
}

func (m *model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch message := message.(type) {
	case error:
		slog.Warn(fmt.Sprintf("Error: %v", message))
		m.displayedError = message
		return m, utils.ClearErrorAfterDelayCmd

	case utils.ClearErrorMsg:
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
	switch message.Type {
	case tea.KeyCtrlC:
		m.inputMode = vim.NORMALMODE
		m.resetCurrentMotion()

		return m, nil
	}

	var cmd tea.Cmd
	switch m.inputMode {
	case vim.NORMALMODE:
		m.currentMotion = append(m.currentMotion, message.String())
		if !m.motions.ContainsPath(m.currentMotion) {
			cmd = utils.MessageCmd(fmt.Errorf("invalid motion: %s", m.currentMotion.View()))
			m.resetCurrentMotion()
			return m, cmd
		}

		message, ok := m.motions.Get(m.currentMotion)
		if !ok {
			return m, nil
		}

		m.resetCurrentMotion()

		switch message.Type {
		case vim.NAVIGATE:
			newApp, cmd := m.apps[m.activeApp].Update(message)
			m.apps[m.activeApp] = newApp.(meta.App)
			return m, cmd

		case vim.SWITCHMODE:
			newMode := message.Data.(vim.InputMode)

			if newMode == vim.INSERTMODE {
				m.inputMode = newMode
				return m, nil
			}

			if newMode == vim.COMMANDMODE {
				m.inputMode = newMode
				m.commandInput.Focus()
				return m, nil
			}

		case vim.SWITCHTAB:
			m.handleTabSwitch(message.Data.(vim.Direction))

		case vim.SWITCHVIEW:
			newApp, cmd := m.apps[m.activeApp].Update(message)
			m.apps[m.activeApp] = newApp.(meta.App)
			return m, cmd
		}

	case vim.COMMANDMODE:
		if message.Type == tea.KeyEnter {
			var cmd tea.Cmd
			if m.commandInput.Value() == "q" {
				cmd = tea.Quit
			} else {
				m.commandInput.Reset()
				m.inputMode = vim.NORMALMODE
			}

			return m, cmd
		}

		m.commandInput, cmd = m.commandInput.Update(message)
		return m, cmd
	}

	return m, nil
}

func (m *model) handleTabSwitch(direction vim.Direction) (*model, tea.Cmd) {
	var cmd tea.Cmd

	switch direction {
	case vim.RIGHT:
		m.activeApp = (m.activeApp + 1) % len(m.apps)

	case vim.LEFT:
		m.activeApp = (m.activeApp - 1)
		if m.activeApp < 0 {
			m.activeApp += len(m.apps)
		}

	default:
		panic(fmt.Sprintf("Invalid tab switch direction %q", direction))
	}

	return m, cmd
}

func (m *model) resetCurrentMotion() {
	m.currentMotion = m.currentMotion[:0]
}
