package models

import (
	"context"
	"log/slog"
	"terminaccounting/models/accounts"
	"terminaccounting/models/entries"
	"terminaccounting/models/journals"
	"terminaccounting/models/ledgers"

	"github.com/jmoiron/sqlx"
)

// Does one-time database schema setup
func SetupSchema(ctx context.Context, db *sqlx.DB) error {
	err := ledgers.SetupSchema(ctx, db)
	if err != nil {
		return err
	}

	err = accounts.SetupSchema(ctx, db)
	if err != nil {
		return err
	}

	err = journals.SetupSchema(ctx, db)
	if err != nil {
		return err
	}

	err = entries.SetupSchemaEntries(ctx, db)
	if err != nil {
		return err
	}

	err = entries.SetupSchemaEntryRows(ctx, db)
	if err != nil {
		return err
	}

	slog.Info("Database schema setup completed")

	return nil
}
