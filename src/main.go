package main

import (
	"fmt"
	"log/slog"
	"os"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/modals"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logLevel = slog.LevelDebug
	}

	file, err := os.OpenFile("debug.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		slog.Error("Couldn't create logger:", "error", err)
		os.Exit(1)
	}
	defer file.Close()
	slog.SetDefault(slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: logLevel})))

	err = database.Connect()
	if err != nil {
		slog.Error("Couldn't connect to database:", "error", err)
		os.Exit(1)
	}

	commandInput := textinput.New()
	commandInput.Cursor.SetMode(cursor.CursorStatic)
	commandInput.Prompt = ":"

	apps := make([]meta.App, 4)
	apps[0] = NewLedgersApp()
	apps[1] = NewEntriesApp()
	apps[2] = NewAccountsApp()
	apps[3] = NewJournalsApp()

	// Map the name(=type) of an app to its index in `apps`
	appIds := make(map[meta.AppType]int, 4)
	appIds[meta.LEDGERSAPP] = 0
	appIds[meta.ENTRIESAPP] = 1
	appIds[meta.ACCOUNTSAPP] = 2
	appIds[meta.JOURNALSAPP] = 3

	am := &appManager{
		activeApp: 0,
		apps:      apps,
		appIds:    appIds,
	}

	mm := &modals.ModalManager{}

	ta := &terminaccounting{
		appManager:   am,
		modalManager: mm,
		showModal:    false,

		inputMode:    meta.NORMALMODE,
		commandInput: commandInput,

		currentMotion: make(meta.Motion, 0),
	}

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
