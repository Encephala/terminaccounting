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
	Id    int        `db:"id"`
	Name  string     `db:"name"`
	Type  LedgerType `db:"type"`
	Notes meta.Notes `db:"notes"`
}

// TODO: All database interactions should be done asynchronously through tea.Cmds, so all these functions should
// return a command that does the interaction.
// Not actually relevant for a sqlite database, but out of principle and for the sake of learning.
// Classid async struggles.

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
	_, err := db.NamedExec(`INSERT INTO ledgers (name, type, notes) VALUES (:name, :type, :notes);`, l)
	if err != nil {
		return -1, err
	}

	queryInsertedId := db.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'ledgers';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return -1, err
	}

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

func SelectLedger(db *sqlx.DB, ledgerId int) (Ledger, error) {
	var result Ledger

	err := db.Get(&result, `SELECT * FROM ledgers WHERE id = :id`, ledgerId)

	return result, err
}

func DeleteLedger(db *sqlx.DB, ledgerId int) error {
	_, err := db.Exec(`DELETE FROM ledgers WHERE id = :id`, ledgerId)

	return err
}
