package database

import (
	"strconv"
	"strings"
	"terminaccounting/meta"
)

type Entry struct {
	Id      int        `db:"id"`
	Journal int        `db:"journal"`
	Notes   meta.Notes `db:"notes"`
}

func (e Entry) FilterValue() string {
	var result strings.Builder
	result.WriteString(strconv.Itoa(e.Id))
	result.WriteString(strconv.Itoa(e.Journal))
	result.WriteString(strings.Join(e.Notes, ";"))
	return result.String()
}

func (e Entry) Title() string {
	return strconv.Itoa(e.Id)
}

func (e Entry) Description() string {
	return strings.Join(e.Notes, "; ")
}

func (er EntryRow) FilterValue() string {
	var result strings.Builder

	result.WriteString(strconv.Itoa(er.Id))

	// TODO: Get entry name, ledger name, account name etc.
	// Maybe I do want to maintain a `[]Ledger` array in ledgers app etc.,
	// for this. Makes sense maybe.
	// Then again, import loops and all. Maybe the main program needs a way to query these things?
	// Or a just a bunch of DB queries.
	// I mean I guess they're just lookups by primary key, that's fiiiine?
	// Probably runs every time the search box updates, maybe it's not "fiiiine".
	result.WriteString(strconv.Itoa(er.Entry))
	result.WriteString(strconv.Itoa(er.Ledger))
	result.WriteString(strconv.Itoa(*er.Account))

	result.WriteString(strconv.Itoa(int(er.Value.Whole)))
	result.WriteString(strconv.Itoa(int(er.Value.Fractional)))

	return result.String()
}

func (er EntryRow) Title() string {
	return strconv.Itoa(er.Id)
}

func (er EntryRow) Description() string {
	return strconv.Itoa(er.Id)
}

func SetupSchemaEntries() (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp("entries")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS entries(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		journal INTEGER NOT NULL,
		notes TEXT,
		FOREIGN KEY (journal) REFERENCES journals(id)
	) STRICT;`

	_, err = DB.Exec(schema)
	return true, err
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

func SetupSchemaEntryRows() (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp("entryrows")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
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

	_, err = DB.Exec(schema)
	return true, err
}

func InsertRows(rows []EntryRow) (int, error) {
	query := `INSERT INTO entryrows
	(entry, ledger, account, document, value, reconciled)
	VALUES
	(:entry, :ledger, :account, :document, :value, :reconciled);`

	result, err := DB.NamedExec(query, rows)
	if err != nil {
		return 0, err
	}
	// I'm assuming this would yield the same error as the query itself?
	changed, _ := result.RowsAffected()

	return int(changed), err
}

func SelectEntries() ([]Entry, error) {
	result := []Entry{}

	err := DB.Select(&result, `SELECT * FROM entries;`)

	return result, err
}

func SelectRows() ([]EntryRow, error) {
	result := []EntryRow{}

	err := DB.Select(&result, `SELECT * FROM entryrows;`)

	return result, err
}

func SelectRowsByLedger(id int) ([]EntryRow, error) {
	result := []EntryRow{}

	err := DB.Select(&result, `SELECT * FROM entryrows WHERE ledger = $1;`, id)

	return result, err
}

func SelectRowsByEntry(id int) ([]EntryRow, error) {
	result := []EntryRow{}

	err := DB.Select(&result, `SELECT * FROM entryrows WHERE entry = $1;`, id)

	return result, err
}
