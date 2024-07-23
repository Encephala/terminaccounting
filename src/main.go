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

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const LEADER = " "

type inputMode string

const NORMALMODE inputMode = "NORMAL"
const INSERTMODE inputMode = "INSERT"
const COMMANDMODE inputMode = "COMMAND"

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeApp int

	apps []meta.App

	// current vim-esque input mode
	inputMode

	// vim-esque command input
	commandInput textinput.Model

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

		inputMode:    NORMALMODE,
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

		// -2 for the tabs and their top borders
		// -1 for the status line
		remainingHeight := message.Height - 2 - 1
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

	result = append(result, m.statusLineView())

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

func (m *model) statusLineView() string {
	var result strings.Builder

	statusLineStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252"))

	switch m.inputMode {
	case NORMALMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("10")).Padding(0, 1)
		result.WriteString(modeStyle.Render("NORMAL"))

		result.WriteString(statusLineStyle.Render(" "))

		convertedStroke := make([]string, len(m.currentStroke))
		for _, stroke := range m.currentStroke {
			if stroke == LEADER {
				stroke = "<leader>"
			}
			convertedStroke = append(convertedStroke, stroke)
		}
		joinedStroke := strings.Join(convertedStroke, "")
		result.WriteString(statusLineStyle.Render(joinedStroke))

		numberOfTrailingEmptyCells := m.viewWidth - len(joinedStroke) - 1
		if numberOfTrailingEmptyCells >= 0 {
			// This has to be in if-statement because on initial render viewWidth is 0,
			// so subtracting 1 leaves negative Repeat count
			result.WriteString(statusLineStyle.Render(strings.Repeat(" ", numberOfTrailingEmptyCells)))
		}

	case INSERTMODE:
		modeStyle := lipgloss.NewStyle().Background(lipgloss.Color("12")).Padding(0, 1)
		result.WriteString(modeStyle.Render("INSERT"))

	case COMMANDMODE:
		result.WriteString(styles.Command.Render(m.commandInput.View()))

	default:
		panic(fmt.Sprintf("unexpected inputMode: %#v", m.inputMode))
	}

	return result.String()
}

func (m *model) handleKeyMsg(message tea.KeyMsg) (*model, tea.Cmd) {
	switch message.Type {
	case tea.KeyCtrlC:
		m.inputMode = NORMALMODE
		m.resetCurrentStroke()

		return m, nil
	}

	var cmd tea.Cmd
	switch m.inputMode {
	case NORMALMODE:
		m.currentStroke = append(m.currentStroke, message.String())

		switch {
		case m.currentStrokeEquals([]string{LEADER, "q"}):
			return m, tea.Quit

		case m.currentStrokeEquals([]string{"i"}):
			m.inputMode = INSERTMODE
			return m, nil

		case m.currentStrokeEquals([]string{"g", "t"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(NEXTTAB)
		case m.currentStrokeEquals([]string{"g", "T"}):
			m.resetCurrentStroke()
			return m.handleTabSwitch(PREVTAB)
		}

		// No case matched
		if len(m.currentStroke) == 3 {
			m.resetCurrentStroke()
		}

	case INSERTMODE:
		var app tea.Model
		app, cmd = m.apps[m.activeApp].Update(message)
		m.apps[m.activeApp] = app.(meta.App)

		return m, cmd

	case COMMANDMODE:
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
	m.currentStroke = make([]string, 0, 3)
}
