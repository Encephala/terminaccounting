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
	commandInput.Placeholder = "command"
	commandInput.Prompt = ""

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
	}

	m.resetCurrentStroke()

	_, err = tea.NewProgram(m).Run()
	if err != nil {
		slog.Error(fmt.Sprintf("Exited with error: %v", err))
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
		return m, nil

	case meta.FatalErrorMsg:
		slog.Error(fmt.Sprintf("Fatal error: %v", message.Error))
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
	}

	app, cmd := m.apps[m.activeApp].Update(message)
	m.apps[m.activeApp] = app.(meta.App)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	result := []string{}

	if m.activeApp < 0 || m.activeApp >= len(m.apps) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.activeApp))
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
		m.resetCurrentStroke()

		return m, nil
	}

	var cmd tea.Cmd
	switch m.inputMode {
	case vim.NORMALMODE:
		m.currentStroke = append(m.currentStroke, message.String())

		switch {
		case m.currentStrokeEquals([]string{vim.LEADER, "q"}):
			return m, tea.Quit

		case m.currentStrokeEquals([]string{"i"}):
			m.inputMode = vim.INSERTMODE
			return m, nil

		case m.currentStrokeEquals([]string{"g", "t"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(NEXTTAB)
		case m.currentStrokeEquals([]string{"g", "T"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(PREVTAB)
		}

		// No case matched
		// if len(m.CurrentStroke) == 3 {
		// 	m.resetCurrentStroke()
		// }

	case vim.INSERTMODE:
		var app tea.Model
		app, cmd = m.apps[m.activeApp].Update(message)
		m.apps[m.activeApp] = app.(meta.App)

		return m, cmd

	case vim.COMMANDMODE:
		m.commandInput, cmd = m.commandInput.Update(message)
		return m, cmd
	}

	return m, nil
}

const NEXTTAB = "NEXTTAB"
const PREVTAB = "PREVTAB"

func (m *model) handleTabSwitch(switchTo string) (*model, tea.Cmd) {
	var cmd tea.Cmd

	switch switchTo {
	case NEXTTAB:
		m.activeApp = (m.activeApp + 1) % len(m.apps)

	case PREVTAB:
		m.activeApp = (m.activeApp - 1)
		if m.activeApp < 0 {
			m.activeApp += len(m.apps)
		}

	default:
		panic(fmt.Sprintf("Handling tab switchTo was %q", switchTo))
	}

	return m, cmd
}

func (m *model) currentStrokeEquals(other []string) bool {
	if len(m.currentStroke) != len(other) {
		return false
	}

	for i, s := range m.currentStroke {
		if s != other[i] {
			return false
		}
	}

	return true
}

func (m *model) resetCurrentStroke() {
	m.currentStroke = make([]string, 0)
}
