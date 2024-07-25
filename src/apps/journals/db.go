package journals

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type JournalType string

const (
	Income   JournalType = "INCOME"
	Expense  JournalType = "EXPENSE"
	CashFlow JournalType = "CASHFLOW"
	General  JournalType = "GENERAL"
)

type Journal struct {
	Id          int         `db:"id"`
	Name        string      `db:"name"`
	JournalType JournalType `db:"type"`
	Notes       []string    `db:"notes"`
}

func setupSchema(db *sqlx.DB) (bool, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "journals")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS journals(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return true, err
}
