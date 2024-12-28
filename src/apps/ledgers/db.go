package ledgers

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type LedgerType string

const (
	INCOME    LedgerType = "INCOME"
	EXPENSE   LedgerType = "EXPENSE"
	ASSET     LedgerType = "ASSET"
	LIABILITY LedgerType = "LIABILITY"
	EQUITY    LedgerType = "EQUITY"
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

func (l *Ledger) Insert(db *sqlx.DB) (int, error) {
	transaction := db.MustBegin()
	defer transaction.Rollback() // If already committed, this is a noop

	_, err := transaction.NamedExec(`INSERT INTO ledgers (name, type, notes) VALUES (:name, :type, :notes);`, l)
	if err != nil {
		return -1, err
	}

	queryInsertedId := transaction.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'ledgers';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return -1, err
	}

	err = transaction.Commit()

	return insertedId, err
}

func (l *Ledger) Update(db *sqlx.DB) error {
	query := `UPDATE ledgers SET
	name = :name,
	type = :type,
	notes = :notes
	WHERE id = :id;`

	_, err := db.NamedExec(query, l)

	return err
}

func SelectLedgers(db *sqlx.DB) ([]Ledger, error) {
	result := []Ledger{}

	err := db.Select(&result, `SELECT * FROM ledgers;`)

	return result, err
}
