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

	DB, err := database.Connect()
	if err != nil {
		slog.Error("Couldn't connect to database:", "error", err)
		os.Exit(1)
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
