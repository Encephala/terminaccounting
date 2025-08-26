package database

import (
	"fmt"
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
	Fractional uint8
}

func ParseDecimalValue(input string) (DecimalValue, error) {
	parts := strings.Split(input, ".")

	if len(parts) == 1 {
		parsed, err := strconv.ParseInt(parts[0], 10, 64)

		return DecimalValue{
			Whole:      parsed,
			Fractional: 0,
		}, err
	}

	if len(parts) != 2 {
		return DecimalValue{}, fmt.Errorf("%s isn't a decimal value", input)
	}

	left := parts[0]
	right := parts[1]

	whole, err := strconv.ParseInt(left, 10, 64)
	if err != nil {
		return DecimalValue{}, err
	}

	var decimal uint64
	if len(right) == 0 {
		decimal = 0
	} else if len(right) == 1 {
		decimal, err = strconv.ParseUint(right, 10, 8)
		if err != nil {
			return DecimalValue{}, err
		}

		decimal = decimal * 10
	} else if len(right) == 2 {
		decimal, err = strconv.ParseUint(right, 10, 8)
		if err != nil {
			return DecimalValue{}, err
		}
	} else {
		return DecimalValue{}, fmt.Errorf("invalid decimal part %s", right)
	}

	return DecimalValue{
		Whole:      whole,
		Fractional: uint8(decimal),
	}, nil
}

func (dv DecimalValue) String() string {
	if dv.Whole >= 0 {
		return fmt.Sprintf("%d.%02d", dv.Whole, dv.Fractional)
	}

	if dv.Whole == -1 {
		return fmt.Sprintf("-0.%02d", 100-dv.Fractional)
	}

	return fmt.Sprintf("%d.%02d", dv.Whole+1, 100-dv.Fractional)
}

func (left DecimalValue) Add(right DecimalValue) DecimalValue {
	return DecimalValue{
		Whole:      left.Whole + right.Whole,
		Fractional: left.Fractional + right.Fractional,
	}
}

func (left DecimalValue) Subtract(right DecimalValue) DecimalValue {
	// Fix uint underflow
	if left.Fractional < right.Fractional {
		return DecimalValue{
			Whole:      left.Whole - right.Whole - 1,
			Fractional: left.Fractional - right.Fractional + 100,
		}
	}

	return DecimalValue{
		Whole:      left.Whole - right.Whole,
		Fractional: left.Fractional - right.Fractional,
	}
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
