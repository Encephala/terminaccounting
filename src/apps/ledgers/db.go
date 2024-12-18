package ledgers

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type LedgerType string

const (
	Income    LedgerType = "INCOME"
	Expense   LedgerType = "EXPENSE"
	Asset     LedgerType = "ASSET"
	Liability LedgerType = "LIABILITY"
	Equity    LedgerType = "EQUITY"
)

func (lt LedgerType) String() string {
	return string(lt)
}

type Ledger struct {
	Id         int        `db:"id"`
	Name       string     `db:"name"`
	LedgerType LedgerType `db:"type"`
	Notes      meta.Notes `db:"notes"`
}

func setupSchema(db *sqlx.DB) (bool, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "ledgers")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS ledgers(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return true, err
}

func (l *Ledger) Insert(db *sqlx.DB) error {
	_, err := db.NamedExec(`INSERT INTO ledgers (name, type, notes) VALUES (:name, :type, :notes)`, l)

	return err
}

func SelectLedgers(db *sqlx.DB) ([]Ledger, error) {
	result := []Ledger{}

	err := db.Select(&result, `SELECT * FROM ledgers;`)

	return result, err
}
