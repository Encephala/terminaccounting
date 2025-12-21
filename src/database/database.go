package database

import (
	"fmt"
	"log/slog"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

func Connect() error {
	var err error
	DB, err = sqlx.Connect("sqlite3", "file:test.db?cache=shared&mode=rwc&_foreign_keys=on")

	return err
}

func DatabaseTableIsSetUp(name string) (bool, error) {
	// Kinda hacky, whatever
	result, err := DB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name=$1", name)
	if err != nil {
		return false, fmt.Errorf("FAILED TO CHECK IF DATABASE IS NEW: %v", err)
	}
	defer result.Close()
	nextRowAvailable := result.Next()

	if nextRowAvailable {
		return true, nil
	}

	return false, nil
}

func InitSchemas() tea.Cmd {
	var makeFatalErrorCmd = func(err error) tea.Cmd {
		return meta.MessageCmd(meta.FatalErrorMsg{Error: err})
	}

	return func() tea.Msg {
		changed, err := setupSchemaLedgers()
		if err != nil {
			return makeFatalErrorCmd(fmt.Errorf("COULD NOT CREATE `ledgers` TABLE: %v", err))
		}
		if changed {
			slog.Info("Set up `ledgers` schema")
		}

		changed, err = setupSchemaAccounts()
		if err != nil {
			return makeFatalErrorCmd(fmt.Errorf("COULD NOT CREATE `accounts` TABLE: %v", err))
		}
		if changed {
			slog.Info("Set up `accounts` schema")
		}

		changed, err = setupSchemaJournals()
		if err != nil {
			return makeFatalErrorCmd(fmt.Errorf("COULD NOT CREATE `journals` TABLE: %v", err))
		}
		if changed {
			slog.Info("Set up `journals` schema")
		}

		changed, err = setupSchemaEntries()
		if err != nil {
			return makeFatalErrorCmd(fmt.Errorf("COULD NOT CREATE `entries` schema: %v", err))
		}
		if changed {
			slog.Info("Set up `entries` schema")
		}

		changed, err = setupSchemaEntryRows()
		if err != nil {
			return makeFatalErrorCmd(fmt.Errorf("COULD NOT CREATE `entryrows` schema: %v", err))
		}
		if changed {
			slog.Info("Set up `entryrows` schema")
		}

		return nil
	}
}

func UpdateCache() error {
	err := UpdateLedgersCache()
	if err != nil {
		return err
	}

	err = UpdateAccountsCache()
	if err != nil {
		return err
	}

	err = UpdateJournalsCache()
	if err != nil {
		return err
	}

	return nil
}
