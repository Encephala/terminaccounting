package journals

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type JournalType string

const (
	INCOME   JournalType = "INCOME"
	EXPENSE  JournalType = "EXPENSE"
	CASHFLOW JournalType = "CASHFLOW"
	GENERAL  JournalType = "GENERAL"
)

type Journal struct {
	Id          int         `db:"id"`
	Name        string      `db:"name"`
	JournalType JournalType `db:"type"`
	Notes       meta.Notes  `db:"notes"`
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

func (j *Journal) Insert(db *sqlx.DB) (int, error) {
	_, err := db.NamedExec(`INSERT INTO journals (name, type, notes) VALUES (:name, :type, :notes)`, j)
	if err != nil {
		return -1, err
	}

	queryInsertedId := db.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'journals';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return -1, err
	}

	return insertedId, err
}

func SelectJournals(db *sqlx.DB) ([]Journal, error) {
	result := []Journal{}

	err := db.Select(&result, `SELECT * FROM journals;`)

	return result, err
}
