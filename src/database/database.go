package database

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

func Connect() error {
	var err error
	DB, err = sqlx.Connect("sqlite3", "file:test.db?cache=shared&mode=rwc&_foreign_keys=on")
	if err != nil {
		return err
	}

	err = InitSchemas()

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

func InitSchemas() error {
	changed, err := setupSchemaLedgers()
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `ledgers` schema")
	}

	changed, err = setupSchemaAccounts()
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `accounts` schema")
	}

	changed, err = setupSchemaJournals()
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `journals` schema")
	}

	changed, err = setupSchemaEntries()
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `entries` schema")
	}

	changed, err = setupSchemaEntryRows()
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `entryrows` schema")
	}

	return nil
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
