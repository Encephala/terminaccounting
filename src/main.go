package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"terminaccounting/accounts"
	"terminaccounting/entries"
	"terminaccounting/journals"
	"terminaccounting/ledgers"
	"terminaccounting/meta"
	"terminaccounting/styles"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeTab int

	apps [4]meta.App
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

	m := model{
		db: db,

		viewWidth:  0,
		viewHeight: 0,

		activeTab: 0,
		apps: [...]meta.App{
			ledgers.Ledgers,
			journals.Journals,
			accounts.Accounts,
			entries.Entries,
		},
	}

	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		slog.Error("Exited with error: ", "error", err)
		os.Exit(1)
	}

	slog.Info("Exited gracefully")
	os.Exit(0)
}

func (m model) Init() tea.Cmd {
	err := meta.SetupSchema(m.db, m.apps[:])
	if err != nil {
		slog.Error("Failed to setup database: ", "error", err)
		return tea.Quit
	}

	slog.Info("Initialised")

	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyTab:
			m.activeTab = min(m.activeTab+1, len(m.apps)-1)

			return m, nil

		case tea.KeyShiftTab:
			m.activeTab = max(m.activeTab-1, 0)

			return m, nil
		}

	case tea.WindowSizeMsg:
		m.viewWidth = message.Width
		m.viewHeight = message.Height

	default:
		slog.Warn(fmt.Sprintf("Unhandled message: %+v", message))
	}

	return m, nil
}

func (m model) View() string {
	result := strings.Builder{}

	if m.activeTab < 0 || m.activeTab >= len(m.apps) {
		panic(fmt.Sprintf("Invalid tab index: %d", m.activeTab))
	}

	tabs := []string{}
	for i, view := range m.apps {
		if i == m.activeTab {
			style := styles.Tab().BorderForeground(view.AccentColour())
			tabs = append(tabs, style.Render(view.Name()))
		} else {
			tabs = append(tabs, styles.Tab().Render(view.Name()))
		}
	}
	tabsRendered := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	tabFill := strings.Repeat(" ", max(0, m.viewWidth-lipgloss.Width(tabsRendered)-1))

	row := lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		tabsRendered,
		tabFill,
	)
	result.WriteString(row + "\n")

	activeApp := m.apps[m.activeTab]

	// -2 for the borders on all sides
	bodyHeight := m.viewHeight - lipgloss.Height(row) - 2
	bodyWidth := m.viewWidth - 2
	bodyStyle := styles.Body(bodyWidth, bodyHeight, activeApp.AccentColour())
	result.WriteString(bodyStyle.Render(activeApp.Render()))

	return result.String()
}
