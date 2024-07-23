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
	"terminaccounting/model"
	"terminaccounting/styles"
	"terminaccounting/view"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type mainModel model.Model

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

	m := &mainModel{
		Db: db,

		ActiveApp: 0,
		Apps: []meta.App{
			// Commented while I'm refactoring a lot, to avoid having to reimplement various interfaces etc.
			ledgers.New(db),
			// accounts.New(),
			// journals.New(),
			entries.New(db),
		},

		InputMode:    model.NORMALMODE,
		CommandInput: commandInput,
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

func (m *mainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	for _, app := range m.Apps {
		cmds = append(cmds, app.Init())
	}

	for i, app := range m.Apps {
		model, cmd := app.Update(meta.SetupSchemaMsg{Db: m.Db})
		m.Apps[i] = model.(meta.App)
		cmds = append(cmds, cmd)
	}

	slog.Info("Initialised")

	return tea.Batch(cmds...)
}

func (m *mainModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch message := message.(type) {
	case error:
		slog.Warn(fmt.Sprintf("Error: %v", message))
		return m, nil

	case meta.FatalErrorMsg:
		slog.Error(fmt.Sprintf("Fatal error: %v", message.Error))
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.ViewWidth = message.Width
		m.ViewHeight = message.Height

		// -3 for the tabs and their borders
		// -1 for the status line
		remainingHeight := message.Height - 3 - 1
		for i, app := range m.Apps {
			model, cmd := app.Update(tea.WindowSizeMsg{
				Width:  message.Width,
				Height: remainingHeight,
			})
			m.Apps[i] = model.(meta.App)
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKeyMsg(message)
	}

	app, cmd := m.Apps[m.ActiveApp].Update(message)
	m.Apps[m.ActiveApp] = app.(meta.App)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *mainModel) View() string {
	result := []string{}

	if m.ActiveApp < 0 || m.ActiveApp >= len(m.Apps) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.ActiveApp))
	}

	tabs := []string{}
	activeTabColour := m.Apps[m.ActiveApp].Colours().Foreground
	for i, app := range m.Apps {
		if i == m.ActiveApp {
			style := styles.ActiveTab(activeTabColour)
			tabs = append(tabs, style.Render(app.Name()))
		} else {
			tabs = append(tabs, styles.Tab(activeTabColour).Render(app.Name()))
		}
	}

	numberOfTrailingEmptyCells := m.ViewWidth - len(m.Apps)*12
	if numberOfTrailingEmptyCells >= 0 {
		tabFill := strings.Repeat(" ", numberOfTrailingEmptyCells)
		style := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(activeTabColour)
		tabs = append(tabs, style.Render(tabFill))
	}

	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	result = append(result, tabsRendered)

	result = append(result, m.Apps[m.ActiveApp].View())

	result = append(result, view.StatusLineView((*model.Model)(m)))

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (m *mainModel) handleKeyMsg(message tea.KeyMsg) (*mainModel, tea.Cmd) {
	switch message.Type {
	case tea.KeyCtrlC:
		m.InputMode = model.NORMALMODE
		m.resetCurrentStroke()

		return m, nil
	}

	var cmd tea.Cmd
	switch m.InputMode {
	case model.NORMALMODE:
		m.CurrentStroke = append(m.CurrentStroke, message.String())

		switch {
		case m.currentStrokeEquals([]string{model.LEADER, "q"}):
			return m, tea.Quit

		case m.currentStrokeEquals([]string{"i"}):
			m.InputMode = model.INSERTMODE
			return m, nil

		case m.currentStrokeEquals([]string{"g", "t"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(NEXTTAB)
		case m.currentStrokeEquals([]string{"g", "T"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(PREVTAB)
		}

		// No case matched
		if len(m.CurrentStroke) == 3 {
			m.resetCurrentStroke()
		}

	case model.INSERTMODE:
		var app tea.Model
		app, cmd = m.Apps[m.ActiveApp].Update(message)
		m.Apps[m.ActiveApp] = app.(meta.App)

		return m, cmd

	case model.COMMANDMODE:
		m.CommandInput, cmd = m.CommandInput.Update(message)
		return m, cmd
	}

	return m, nil
}

const NEXTTAB = "NEXTTAB"
const PREVTAB = "PREVTAB"

func (m *mainModel) handleTabSwitch(switchTo string) (*mainModel, tea.Cmd) {
	var cmd tea.Cmd

	switch switchTo {
	case NEXTTAB:
		m.ActiveApp = (m.ActiveApp + 1) % len(m.Apps)

	case PREVTAB:
		m.ActiveApp = (m.ActiveApp - 1)
		if m.ActiveApp < 0 {
			m.ActiveApp += len(m.Apps)
		}

	default:
		panic(fmt.Sprintf("Handling tab switchTo was %q", switchTo))
	}

	return m, cmd
}

func (m *mainModel) currentStrokeEquals(other []string) bool {
	if len(m.CurrentStroke) != len(other) {
		return false
	}

	for i, s := range m.CurrentStroke {
		if s != other[i] {
			return false
		}
	}

	return true
}

func (m *mainModel) resetCurrentStroke() {
	m.CurrentStroke = make([]string, 0, 3)
}
