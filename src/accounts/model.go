package accounts

import (
	"log/slog"
	"terminaccounting/meta"

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

func (a *app) SetupSchema(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "accounts")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	slog.Info("Creating `accounts` table")

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}
