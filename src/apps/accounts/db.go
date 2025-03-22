package accounts

import (
	"terminaccounting/meta"
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

func setupSchema() (bool, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(DB, "accounts")
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

	_, err = DB.Exec(schema)
	return true, err
}
