package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"terminaccounting/apps/entries"
	"terminaccounting/apps/ledgers"
	"terminaccounting/meta"
	"terminaccounting/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const LEADER = " "

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeApp int

	apps []meta.App

	// vim-esque command input
	commandInput  textinput.Model
	commandActive bool

	// current vim-esque key stroke
	currentStroke []string
}

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

		commandInput:  commandInput,
		commandActive: false,
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

		// -2 for the tabs and their top borders
		remainingHeight := message.Height - 2
		if m.commandActive {
			remainingHeight -= 1
		}
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

	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	result := []string{}

	if m.activeApp < 0 || m.activeApp >= len(m.apps) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.activeApp))
	}

	tabs := []string{}
	for i, app := range m.apps {
		if i == m.activeApp {
			style := styles.Tab.BorderForeground(app.Colours().Foreground)
			tabs = append(tabs, style.Render(app.Name()))
		} else {
			tabs = append(tabs, styles.Tab.Render(app.Name()))
		}
	}
	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	result = append(result, tabsRendered)

	result = append(result, m.apps[m.activeApp].View())

	if m.commandActive {
		result = append(result, styles.Command.Render(m.commandInput.View()))
	}

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (m *model) handleKeyMsg(message tea.KeyMsg) (*model, tea.Cmd) {
	switch message.Type {
	case tea.KeyCtrlC:
		m.currentStroke = make([]string, 0, 3)

		m.commandInput.Reset()
		m.commandActive = false

		return m, nil
	}

	m.currentStroke = append(m.currentStroke, message.String())

	switch {
	case m.currentStrokeEquals([]string{LEADER, "q"}):
		return m, tea.Quit

	case m.currentStrokeEquals([]string{"g", "t"}):
		m.resetCurrentStroke()
		return m.handleTabSwitch(NEXTTAB)
	case m.currentStrokeEquals([]string{"g", "T"}):
		m.resetCurrentStroke()
		return m.handleTabSwitch(PREVTAB)
	}

	// TODO: This shouldn't always happen, I have to think about how to differentiate when a key is part of a stroke,
	// versus when a key is to control some item on the screen.
	var cmd tea.Cmd
	if m.commandActive {
		m.commandInput, cmd = m.commandInput.Update(message)
	} else {
		var app tea.Model
		app, cmd = m.apps[m.activeApp].Update(message)
		m.apps[m.activeApp] = app.(meta.App)
	}

	return m, cmd
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
	m.currentStroke = make([]string, 0, 3)
}
