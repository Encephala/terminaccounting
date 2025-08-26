package database

import (
	"fmt"
	"strconv"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
)

type LedgerType string

const (
	INCOMELEDGER    LedgerType = "INCOME"
	EXPENSELEDGER   LedgerType = "EXPENSE"
	ASSETLEDGER     LedgerType = "ASSET"
	LIABILITYLEDGER LedgerType = "LIABILITY"
	EQUITYLEDGER    LedgerType = "EQUITY"
)

func (l Ledger) FilterValue() string {
	var result strings.Builder
	result.WriteString(l.Name)
	result.WriteString(strings.Join(l.Notes, ";"))
	return result.String()
}

func (l Ledger) Title() string {
	return l.Name
}

func (l Ledger) Description() string {
	return l.Name
}

func (l Ledger) String() string {
	return l.Name + " (" + strconv.Itoa(l.Id) + ")"
}

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

func MakeLoadLedgersDetailCmd(id int) tea.Cmd {
	return func() tea.Msg {
		ledger, err := SelectLedger(id)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER WITH ID %d: %#v", id, err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.LEDGERS,
			Model:     meta.LEDGER,
			Data:      ledger,
		}
	}
}

func SetupSchemaLedgers() (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp("ledgers")
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

	_, err = DB.Exec(schema)
	return true, err
}

func (l *Ledger) Insert() (int, error) {
	_, err := DB.NamedExec(`INSERT INTO ledgers (name, type, notes) VALUES (:name, :type, :notes);`, l)
	if err != nil {
		return -1, err
	}

	queryInsertedId := DB.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'ledgers';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return -1, err
	}

	return insertedId, err
}

func (l *Ledger) Update() error {
	query := `UPDATE ledgers SET
	name = :name,
	type = :type,
	notes = :notes
	WHERE id = :id;`

	_, err := DB.NamedExec(query, l)

	return err
}

func SelectLedgers() ([]Ledger, error) {
	result := []Ledger{}

	err := DB.Select(&result, `SELECT * FROM ledgers;`)

	return result, err
}

func SelectLedger(ledgerId int) (Ledger, error) {
	var result Ledger

	err := DB.Get(&result, `SELECT * FROM ledgers WHERE id = :id`, ledgerId)

	return result, err
}

func DeleteLedger(ledgerId int) error {
	_, err := DB.Exec(`DELETE FROM ledgers WHERE id = :id`, ledgerId)

	return err
}

func MakeSelectLedgersCmd(targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectLedgers()
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGERS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.LEDGER,
			Data:      rows,
		}
	}
}
