package journals

import (
	"terminaccounting/meta"
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

func setupSchema() (bool, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(DB, "journals")
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

	_, err = DB.Exec(schema)
	return true, err
}

func SelectJournals() ([]Journal, error) {
	result := []Journal{}

	err := DB.Select(&result, `SELECT * FROM journals;`)

	return result, err
}
