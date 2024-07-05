package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"terminaccounting/models"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	db *sqlx.DB
}

func main() {
	file, err := os.OpenFile("debug.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		slog.Error("Couldn't create logger: ", "error", err)
		os.Exit(1)
	}
	defer file.Close()
	log.SetOutput(file)

	slog.Info("Program started")

	db, err := sqlx.Open("sqlite3", "file:test.db?cache=shared&mode=rwc")
	if err != nil {
		slog.Error("Couldn't open database: ", "error", err)
		os.Exit(1)
	}

	m := model{
		db: db,
	}

	_, err = tea.NewProgram(m).Run()
	if err != nil {
		slog.Error("Program exited with error: ", "error", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	ctx := context.Background()

	err := models.SetupSchema(ctx, m.db)
	if err != nil {
		slog.Error("Failed to setup database: ", "error", err)

		return tea.Quit
	}

	slog.Info("Finished Init")

	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.KeyMsg:
		switch message.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	return ""
}
