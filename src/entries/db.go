package entries

import (
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type Entry struct {
	Id      int        `db:"id"`
	Journal int        `db:"journal"`
	Notes   meta.Notes `db:"notes"`
}

func setupSchemaEntries(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "entries")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS entries(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		journal INTEGER NOT NULL,
		notes TEXT,
		FOREIGN KEY (journal) REFERENCES journals(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}

type DecimalValue struct {
	Whole      int64
	Fractional uint64
}

type EntryRow struct {
	Id         int          `db:"id"`
	Entry      int          `db:"entry"`
	Ledger     int          `db:"ledger"`
	Account    *int         `db:"account"`
	Document   *string      `db:"document"`
	Value      DecimalValue `db:"value"`
	Reconciled bool         `db:"reconciled"`
}

func setupSchemaEntryRows(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "entryrows")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS entryrows(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry INTEGER NOT NULL,
		ledger INTEGER NOT NULL,
		account INTEGER,
		document TEXT,
		value BLOB NOT NULL,
		reconciled INTEGER NOT NULL,
		FOREIGN KEY (entry) REFERENCES entries(id),
		FOREIGN KEY (ledger) REFERENCES ledgers(id),
		FOREIGN KEY (account) REFERENCES accounts(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}

func InsertRows(db *sqlx.DB, rows []EntryRow) (int, error) {
	query := `INSERT INTO entryrows
	(entry, ledger, account, document, value, reconciled)
	VALUES
	(:entry, :ledger, :account, :document, :value, :reconciled);`

	result, err := db.NamedExec(query, rows)
	if err != nil {
		return 0, err
	}
	// I'm assuming this would yield the same error as the query itself?
	changed, _ := result.RowsAffected()

	return int(changed), err
}

func SelectEntries(db *sqlx.DB) ([]Entry, error) {
	result := []Entry{}

	err := db.Select(&result, `SELECT * FROM entries;`)

	return result, err
}

func SelectRows(db *sqlx.DB) ([]EntryRow, error) {
	result := []EntryRow{}

	err := db.Select(&result, `SELECT * FROM entryrows;`)

	return result, err
}

func SelectRowsByLedger(db *sqlx.DB, id int) ([]EntryRow, error) {
	result := []EntryRow{}

	err := db.Select(&result, `SELECT * FROM entryrows WHERE ledger = $1;`, id)

	return result, err
}

func SelectRowsByEntry(db *sqlx.DB, id int) ([]EntryRow, error) {
	result := []EntryRow{}

	err := db.Select(&result, `SELECT * FROM entryrows WHERE entry = $1;`, id)

	return result, err
}
