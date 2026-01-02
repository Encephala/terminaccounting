package database

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

func Connect() (*sqlx.DB, error) {
	DB, err := sqlx.Connect("sqlite3", "file:test.db?cache=shared&mode=rwc&_foreign_keys=on")
	if err != nil {
		return DB, err
	}

	err = InitSchemas(DB)

	return DB, err
}

func DatabaseTableIsSetUp(DB *sqlx.DB, name string) (bool, error) {
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

func InitSchemas(DB *sqlx.DB) error {
	changed, err := setupSchemaLedgers(DB)
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `ledgers` schema")
	}

	changed, err = setupSchemaAccounts(DB)
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `accounts` schema")
	}

	changed, err = setupSchemaJournals(DB)
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `journals` schema")
	}

	changed, err = setupSchemaEntries(DB)
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `entries` schema")
	}

	changed, err = setupSchemaEntryRows(DB)
	if err != nil {
		return err
	}
	if changed {
		slog.Info("Set up `entryrows` schema")
	}

	return nil
}

func UpdateCache(DB *sqlx.DB) error {
	err := UpdateLedgersCache(DB)
	if err != nil {
		return err
	}

	err = UpdateAccountsCache(DB)
	if err != nil {
		return err
	}

	err = UpdateJournalsCache(DB)
	if err != nil {
		return err
	}

	return nil
}
