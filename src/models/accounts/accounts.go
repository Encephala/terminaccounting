package accounts

import (
	"context"
	"log/slog"
	"terminaccounting/models/utils"

	"github.com/jmoiron/sqlx"
)

type AccountType string

const (
	Debtor   AccountType = "DEBTOR"
	Creditor AccountType = "CREDITOR"
)

type Account struct {
	id          int         `db:"id"`
	name        string      `db:"name"`
	accountType AccountType `db:"type"`
	notes       []string    `db:"notes"`
}

func SetupSchema(ctx context.Context, db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(ctx, db, "accounts")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `accounts` table")

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(69),
		type INTEGER,
		notes JSONB
	);`

	_, err = db.ExecContext(ctx, schema)
	return err
}
