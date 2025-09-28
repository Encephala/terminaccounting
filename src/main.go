package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"terminaccounting/database"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

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

	_, err = db.Exec(`PRAGMA foreign_keys = ON;`)
	if err != nil {
		slog.Error("Couldn't enable foreign keys")
		os.Exit(1)
	}

	database.DB = db

	commandInput := textinput.New()
	commandInput.Cursor.SetMode(cursor.CursorStatic)
	commandInput.Prompt = ":"

	motionSet := meta.DefaultMotionSet()
	commandSet := meta.DefaultCommandSet()

	apps := make([]meta.App, 2)
	apps[0] = NewLedgersApp()
	apps[1] = NewEntriesApp()
	// Commented while I'm refactoring a lot, to avoid having to reimplement various interfaces etc.
	// apps[meta.JOURNALS] = journals.New()
	// apps[meta.ACCOUNTS] = accounts.New()

	// Map the name(=type) of an app to its index in `apps`
	appIds := make(map[meta.AppType]int, 2)
	appIds[meta.LEDGERS] = 0
	appIds[meta.ENTRIES] = 1

	am := &appManager{
		activeApp: 0,
		apps:      apps,
		appIds:    appIds,
	}

	modal := &modalModel{}

	ta := &terminaccounting{
		appManager: am,
		modal:      modal,
		showModal:  false,

		inputMode:    meta.NORMALMODE,
		commandInput: commandInput,

		currentMotion: make(meta.Motion, 0),
		motionSet:     motionSet,

		commandSet: commandSet,
	}

	ta.overlay = newOverlay(ta)

	finalModel, err := tea.NewProgram(ta).Run()
	if err != nil {
		message := fmt.Sprintf("Bubbletea error: %v", err)
		slog.Error(message)
		fmt.Println(message)
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
