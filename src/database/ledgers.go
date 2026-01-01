package database

import (
	"fmt"
	"strconv"
	"strings"
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
)

// Globally accessible list of available ledgers
var AvailableLedgers []Ledger

func AvailableLedgersAsItempickerItems() []itempicker.Item {
	var result []itempicker.Item

	for _, ledger := range AvailableLedgers {
		result = append(result, ledger)
	}

	return result
}

type LedgerType string

const (
	INCOMELEDGER    LedgerType = "INCOME"
	EXPENSELEDGER   LedgerType = "EXPENSE"
	ASSETLEDGER     LedgerType = "ASSET"
	LIABILITYLEDGER LedgerType = "LIABILITY"
	EQUITYLEDGER    LedgerType = "EQUITY"
)

// Listen, it's a great hash function
func (lt LedgerType) CompareId() int {
	switch lt {
	case INCOMELEDGER:
		return 0
	case EXPENSELEDGER:
		return 1
	case ASSETLEDGER:
		return 2
	case LIABILITYLEDGER:
		return 3
	case EQUITYLEDGER:
		return 4
	default:
		panic(fmt.Sprintf("unexpected database.AccountType: %#v", lt))
	}
}

func (lt LedgerType) String() string {
	return string(lt)
}

type Ledger struct {
	Id         int        `db:"id"`
	Name       string     `db:"name"`
	Type       LedgerType `db:"type"`
	Notes      meta.Notes `db:"notes"`
	IsAccounts bool       `db:"is_accounts"`
}

func (l Ledger) FilterValue() string {
	var result strings.Builder

	result.WriteString(l.Name)
	result.WriteString(string(l.Type))
	result.WriteString(l.Notes.Collapse())

	return result.String()
}

func (l Ledger) Title() string {
	return l.Name
}

func (l Ledger) Description() string {
	return l.Notes.Collapse()
}

func (l Ledger) String() string {
	return l.Name + " (" + strconv.Itoa(l.Id) + ")"
}

func (l Ledger) CompareId() int {
	return l.Id
}

func MakeLoadLedgersDetailCmd(id int) tea.Cmd {
	return func() tea.Msg {
		ledger, err := SelectLedger(id)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER WITH ID %d: %#v", id, err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.LEDGERSAPP,
			Model:     meta.LEDGERMODEL,
			Data:      ledger,
		}
	}
}

func MakeLoadLedgersRowsCmd(ledgerId int) tea.Cmd {
	// Aren't closures just great
	return func() tea.Msg {
		rows, err := SelectRowsByLedger(ledgerId)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD LEDGER ROWS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.LEDGERSAPP,
			Model:     meta.ENTRYROWMODEL,
			Data:      rows,
		}
	}
}

func setupSchemaLedgers() (bool, error) {
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
		notes TEXT,
		is_accounts INTEGER NOT NULL
	) STRICT;`

	_, err = DB.Exec(schema)
	return true, err
}

func (l *Ledger) Insert() (int, error) {
	result, err := DB.NamedExec(
		`INSERT INTO ledgers (name, type, notes, is_accounts)
		VALUES (:name, :type, :notes, :is_accounts);`,
		l)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	err = UpdateLedgersCache()
	if err != nil {
		return int(id), err
	}

	return int(id), nil
}

func (l Ledger) Update() error {
	query := `UPDATE ledgers SET
	name = :name,
	type = :type,
	notes = :notes,
	is_accounts = :is_accounts
	WHERE id = :id;`

	_, err := DB.NamedExec(query, l)
	if err != nil {
		return err
	}

	err = UpdateLedgersCache()

	return err
}

func SelectLedgers() ([]Ledger, error) {
	var result []Ledger

	err := DB.Select(&result, `SELECT * FROM ledgers;`)

	return result, err
}

func UpdateLedgersCache() error {
	ledgers, err := SelectLedgers()
	if err != nil {
		return err
	}

	AvailableLedgers = ledgers

	return nil
}

func SelectLedger(ledgerId int) (Ledger, error) {
	var result Ledger

	err := DB.Get(&result, `SELECT * FROM ledgers WHERE id = $1;`, ledgerId)

	return result, err
}

func DeleteLedger(ledgerId int) error {
	_, err := DB.Exec(`DELETE FROM ledgers WHERE id = $1;`, ledgerId)
	if err != nil {
		return err
	}

	err = UpdateLedgersCache()

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
			Model:     meta.LEDGERMODEL,
			Data:      rows,
		}
	}
}
