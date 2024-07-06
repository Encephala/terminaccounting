package ledgers

import (
	"log/slog"
	"terminaccounting/models/utils"

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

type Notes []string

type Ledger struct {
	Id   int    `db:"id"`
	Name string `db:"name"`

	// TODO: Can this work with sqlx? I think not, because some type conversion is needed?
	// But then the whole marshalling into struct isn't going to work and I'd have to do either two queries,
	// or just ditch the whole idea altogether and completely lose the convenience.
	// Ah well, we'll see
	LedgerType LedgerType `db:"type"`
	Notes      Notes      `db:"notes"`
}

func SetupSchema(db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(db, "ledgers")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `ledgers` table")

	schema := `CREATE TABLE IF NOT EXISTS ledgers(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return err
}

func Insert(db *sqlx.DB, ledger *Ledger) error {
	_, err := db.NamedExec(`INSERT INTO ledgers VALUES (:id, :name, :type, :notes)`, ledger)

	return err
}

func SelectAll(db *sqlx.DB) ([]Ledger, error) {
	result := make([]Ledger, 0)

	err := db.Select(&result, `SELECT * FROM ledgers;`)

	return result, err
}
