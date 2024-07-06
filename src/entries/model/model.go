package model

import (
	"log/slog"
	modelUtils "terminaccounting/meta/models/utils"

	"github.com/jmoiron/sqlx"
)

type Entry struct {
	id      int      `db:"id"`
	journal int      `db:"journal"`
	notes   []string `db:"notes"`
}

func SetupSchemaEntries(db *sqlx.DB) error {
	isSetUp, err := modelUtils.TableIsSetUp(db, "entries")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `entries` table")

	schema := `CREATE TABLE IF NOT EXISTS entries(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		journal INTEGER,
		notes TEXT,
		FOREIGN KEY (journal) REFERENCES journals(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return err
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

func SetupSchemaEntryRows(db *sqlx.DB) error {
	isSetUp, err := modelUtils.TableIsSetUp(db, "entryrows")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `entryrows` table")

	schema := `CREATE TABLE IF NOT EXISTS entryrows(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry INTEGER,
		ledger INTEGER,
		account INTEGER,
		document TEXT,
		FOREIGN KEY (entry) REFERENCES entries(id),
		FOREIGN KEY (ledger) REFERENCES ledgers(id),
		FOREIGN KEY (account) REFERENCES accounts(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return err
}
