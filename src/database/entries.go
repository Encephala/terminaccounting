package database

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"terminaccounting/meta"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
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

func (e Entry) CompareId() int {
	return e.Id
}

func CalculateTotal(rows []EntryRow) CurrencyValue {
	sum := CurrencyValue(0)

	for _, row := range rows {
		sum = sum.Add(row.Value)
	}

	return sum
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

func (e Entry) Insert(rows []EntryRow) (int, error) {
	transaction, err := DB.Beginx()
	defer transaction.Rollback()

	if err != nil {
		return 0, err
	}

	res, err := transaction.NamedExec(`INSERT INTO entries (journal, notes) VALUES (:journal, :notes)`, e)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	for i, row := range rows {
		row.Entry = int(id)
		rows[i] = row
	}

	_, err = insertRows(transaction, rows)
	if err != nil {
		return 0, err
	}

	err = transaction.Commit()
	if err != nil {
		return int(id), err
	}

	return int(id), nil
}

func (e Entry) Update(rows []EntryRow) error {
	entryQuery := `UPDATE entries SET
	journal = :journal,
	notes = :notes
	WHERE id = :id;`

	tx := DB.MustBegin()
	defer tx.Rollback()

	totalChanged := 0

	res, err := tx.NamedExec(entryQuery, e)
	if err != nil {
		return err
	}
	changedMain, err := res.RowsAffected()
	if err != nil {
		return err
	}
	totalChanged += int(changedMain)

	// Delete all rows and then create new ones
	// Brute force, but robust way to handle deleting and creating rows
	res, err = tx.Exec(`DELETE FROM entryrows WHERE entry = $1;`, e.Id)
	if err != nil {
		return err
	}
	changedDelete, err := res.RowsAffected()
	if err != nil {
		return err
	}
	totalChanged += int(changedDelete)

	// Inject the ID's into the new rows
	for i := range rows {
		rows[i].Entry = e.Id
	}

	changedInsert, err := insertRows(tx, rows)
	if err != nil {
		return err
	}
	totalChanged += changedInsert

	tx.Commit()

	slog.Debug(fmt.Sprintf("Updated entry %d, changed %d rows", e.Id, totalChanged))

	return nil
}

type CurrencyValue int64

func ParseCurrencyValue(input string) (CurrencyValue, error) {
	parts := strings.Split(input, ".")

	if len(parts) == 1 {
		// No decimal part provided
		parsed, err := strconv.ParseInt(parts[0], 10, 64)

		return CurrencyValue(parsed * 100), err
	}

	if len(parts) != 2 {
		return 0, fmt.Errorf("%s isn't a decimal value", input)
	}

	left := parts[0]
	right := parts[1]

	whole, err := strconv.ParseInt(left, 10, 64)
	if err != nil {
		return 0, err
	}

	var decimal uint64
	if len(right) == 0 {
		decimal = 0
	} else if len(right) == 1 {
		decimal, err = strconv.ParseUint(right, 10, 8)
		if err != nil {
			return 0, err
		}

		decimal = decimal * 10
	} else if len(right) == 2 {
		decimal, err = strconv.ParseUint(right, 10, 8)
		if err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("invalid decimal part %s", right)
	}

	if whole >= 0 {
		return CurrencyValue(whole*100 + int64(decimal)), nil
	} else {
		panic("Why did a negative value get parsed?")
	}
}

func (cv CurrencyValue) String() string {
	whole := cv / 100
	decimal := cv % 100

	if cv >= 0 {
		return fmt.Sprintf("%d.%02d", whole, decimal)
	} else if cv <= -100 {
		// Negate the decimal to avoid `-6.-90`
		return fmt.Sprintf("%d.%02d", whole, -decimal)
	} else {
		// Negate the decimal and insert a minus because the whole is 0 which doesn't get stringified as -0
		return fmt.Sprintf("-%d.%02d", whole, -decimal)
	}
}

func (left CurrencyValue) Add(right CurrencyValue) CurrencyValue {
	return left + right
}

func (left CurrencyValue) Subtract(right CurrencyValue) CurrencyValue {
	return left - right
}

type Date time.Time

func (d Date) String() string {
	return time.Time(d).Format(DATE_FORMAT)
}

type EntryRow struct {
	Id         int           `db:"id"`
	Entry      int           `db:"entry"`
	Date       Date          `db:"date"`
	Ledger     int           `db:"ledger"`
	Account    *int          `db:"account"`
	Document   *string       `db:"document"`
	Value      CurrencyValue `db:"value"`
	Reconciled bool          `db:"reconciled"`
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
		date TEXT NOT NULL,
		ledger INTEGER NOT NULL,
		account INTEGER,
		document TEXT,
		value INTEGER NOT NULL,
		reconciled INTEGER NOT NULL,
		FOREIGN KEY (entry) REFERENCES entries(id),
		FOREIGN KEY (ledger) REFERENCES ledgers(id),
		FOREIGN KEY (account) REFERENCES accounts(id)
	) STRICT;`

	_, err = DB.Exec(schema)
	return true, err
}

func insertRows(transaction *sqlx.Tx, rows []EntryRow) (int, error) {
	query := `INSERT INTO entryrows
	(entry, date, ledger, account, document, value, reconciled)
	VALUES
	(:entry, :date, :ledger, :account, :document, :value, :reconciled);`

	result, err := transaction.NamedExec(query, rows)
	if err != nil {
		return 0, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(changed), err
}

func SelectEntries() ([]Entry, error) {
	result := []Entry{}

	err := DB.Select(&result, `SELECT * FROM entries;`)

	return result, err
}

func SelectEntry(id int) (Entry, error) {
	var result Entry

	err := DB.Get(&result, `SELECT * FROM entries WHERE id = $1;`, id)

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

func MakeSelectEntryCmd(entryId int, targetApp meta.AppType) tea.Cmd {
	// Shoutout to closures
	return func() tea.Msg {
		rows, err := SelectEntry(entryId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ENTRY: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.ENTRY,
			Data:      rows,
		}
	}
}

func MakeSelectEntryRowsCmd(entryId int, targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectRowsByEntry(entryId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ENTRY ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.ENTRYROW,
			Data:      rows,
		}
	}
}
