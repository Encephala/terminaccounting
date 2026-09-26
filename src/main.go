package main

import (
	"fmt"
	"log/slog"
	"os"
	"terminaccounting/database"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logFile, err := initSlog()
	if err != nil {
		slog.Error("Couldn't create logger:", "error", err)
		os.Exit(1)
	}
	defer logFile.Close()

	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("Couldn't load config:", "error", err)
		os.Exit(1)
	}

	DB, err := database.Connect(cfg.DBPath)
	if err != nil {
		slog.Error("Couldn't connect to database:", "error", err)
		os.Exit(1)
	}
	defer DB.Close()

	if cfg.Env == "prod" {
		err := runSSHServer(DB, cfg.SSHAddr)
		if err != nil {
			slog.Error("SSH server error", "error", err)
			os.Exit(1)
		}

		slog.Info("Exited gracefully")
		os.Exit(0)
	}

	ta := newTerminaccounting(DB)

	finalModel, err := tea.NewProgram(ta, tea.WithAltScreen()).Run()
	if err != nil {
		slog.Error("Bubbletea error", "error", err)
		fmt.Printf("Bubbletea error: %v\n", err)
		os.Exit(1)
	}

	err = finalModel.(*terminaccounting).fatalError
	if err != nil {
		message := fmt.Sprintf("Program exited with fatal error: %v", err)
		fmt.Println(message)
		os.Exit(1)
	}

	slog.Info("Exited gracefully")
	os.Exit(0)
}
