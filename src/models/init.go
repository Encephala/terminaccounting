package models

import (
	"log/slog"
	"terminaccounting/models/accounts"
	"terminaccounting/models/entries"
	"terminaccounting/models/journals"
	"terminaccounting/models/ledgers"

	"github.com/jmoiron/sqlx"
)

// Does one-time database schema setup
func SetupSchema(db *sqlx.DB) error {
	setupFunctions := []func(*sqlx.DB) error{
		ledgers.SetupSchema,
		accounts.SetupSchema,
		journals.SetupSchema,
		entries.SetupSchemaEntries,
		entries.SetupSchemaEntryRows,
	}

	var err error
	for _, function := range setupFunctions {
		err = function(db)
		if err != nil {
			return err
		}
	}

	slog.Info("Database schema setup completed")

	return nil
}
