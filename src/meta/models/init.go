package models

import (
	"log/slog"
	accountsModel "terminaccounting/accounts/model"
	entriesModel "terminaccounting/entries/model"
	journalsModel "terminaccounting/journals/model"
	ledgersModel "terminaccounting/ledgers/model"

	"github.com/jmoiron/sqlx"
)

// Does one-time database schema setup
func SetupSchema(db *sqlx.DB) error {
	setupFunctions := []func(*sqlx.DB) error{
		ledgersModel.SetupSchema,
		accountsModel.SetupSchema,
		journalsModel.SetupSchema,
		entriesModel.SetupSchemaEntries,
		entriesModel.SetupSchemaEntryRows,
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
