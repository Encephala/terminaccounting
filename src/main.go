package main

import (
	"log"
	"log/slog"
	"os"
	"strings"
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

	activeTab int

	apps [1]meta.App
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

		activeTab: 0,
		apps: [1]meta.App{
			ledgers.Ledgers,
		},
	}

	_, err = tea.NewProgram(m).Run()
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
			m.activeTab++
			return m, nil

		case tea.KeyShiftTab:
			m.activeTab--
			return m, nil
		}
	}

	return m, nil
}

func (m model) View() string {
	result := strings.Builder{}

	tabs := []string{}
	for i, view := range m.apps {
		if i == m.activeTab {
			tabs = append(tabs, styles.ActiveTab().Render(view.TabName()))
		} else {
			tabs = append(tabs, styles.Tab().Render(view.TabName()))
		}
	}

	result.WriteString(lipgloss.JoinHorizontal(
		lipgloss.Top,
		tabs...,
	))

	return result.String()
}
