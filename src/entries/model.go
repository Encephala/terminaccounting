package entries

import (
	"log/slog"
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type Entry struct {
	id      int      `db:"id"`
	journal int      `db:"journal"`
	notes   []string `db:"notes"`
}

func (a *app) SetupSchema(db *sqlx.DB) (int, error) {
	changedEntries, err := SetupSchemaEntries(db)
	if err != nil {
		return 0, err
	}

	changedEntryRows, err := SetupSchemaEntryRows(db)
	if err != nil {
		return changedEntries, err
	}

	return changedEntries + changedEntryRows, nil
}

func SetupSchemaEntries(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "entries")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	slog.Info("Creating `entries` table")

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
	whole      int
	fractional int
}

type EntryRow struct {
	id         int          `db:"id"`
	entry      int          `db:"entry"`
	ledger     int          `db:"ledger"`
	account    int          `db:"account"`
	document   string       `db:"document"`
	value      DecimalValue `db:"value"`
	reconciled bool         `db:"reconciled"`
}

func SetupSchemaEntryRows(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "entryrows")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	slog.Info("Creating `entryrows` table")

	schema := `CREATE TABLE IF NOT EXISTS entryrows(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry INTEGER NOT NULL,
		ledger INTEGER NOT NULL,
		account INTEGER NOT NULL,
		document TEXT,
		FOREIGN KEY (entry) REFERENCES entries(id),
		FOREIGN KEY (ledger) REFERENCES ledgers(id),
		FOREIGN KEY (account) REFERENCES accounts(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}
