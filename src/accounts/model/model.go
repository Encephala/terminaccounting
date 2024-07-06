package model

import (
	"log/slog"
	modelUtils "terminaccounting/meta/models/utils"

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

func SetupSchema(db *sqlx.DB) error {
	isSetUp, err := modelUtils.TableIsSetUp(db, "accounts")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `accounts` table")

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		type INTEGER,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return err
}
