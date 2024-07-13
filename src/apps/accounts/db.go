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

func setupSchema(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "accounts")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}
