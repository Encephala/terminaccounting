package accounts

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type AccountType string

const (
	Debtor   AccountType = "DEBTOR"
	Creditor AccountType = "CREDITOR"
)

type Account struct {
	Id          int         `db:"id"`
	Name        string      `db:"name"`
	AccountType AccountType `db:"type"`
	Notes       []string    `db:"notes"`
}

func setupSchema(db *sqlx.DB) (bool, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "accounts")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return true, err
}
