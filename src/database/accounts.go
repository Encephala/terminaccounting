package database

import (
	"fmt"
	"local/bubbles/itempicker"
	"strconv"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Globally accessible list of available accounts
var AvailableAccounts []Account

func AvailableAccountsAsItempickerItems() []itempicker.Item {
	var result []itempicker.Item

	for _, account := range AvailableAccounts {
		result = append(result, &account)
	}

	return result
}

type AccountType string

const (
	DEBTOR   AccountType = "DEBTOR"
	CREDITOR AccountType = "CREDITOR"
)

func (at AccountType) CompareId() int {
	switch at {
	case DEBTOR:
		return 0
	case CREDITOR:
		return 1
	default:
		panic(fmt.Sprintf("unexpected database.AccountType: %#v", at))
	}
}

func (at AccountType) String() string {
	return string(at)
}

type Account struct {
	Id    int         `db:"id"`
	Name  string      `db:"name"`
	Type  AccountType `db:"type"`
	Notes meta.Notes  `db:"notes"`
}

func (a Account) FilterValue() string {
	var result strings.Builder

	result.WriteString(a.Name)
	result.WriteString(string(a.Type))
	result.WriteString(strings.Join(a.Notes, ";"))

	return result.String()
}

func (a Account) Title() string {
	return a.Name
}

func (a Account) Description() string {
	return a.Notes.Collapse()
}

// *Account because they're nullable for the sake of the itempicker
func (a *Account) String() string {
	if a == nil {
		return lipgloss.NewStyle().Italic(true).Render("None")
	}

	return a.Name + " (" + strconv.Itoa(a.Id) + ")"
}

func (a *Account) CompareId() int {
	// Since sqlite autoincrements from 1, -1 will never be a valid ID
	if a == nil {
		return -1
	}

	return a.Id
}

func MakeLoadAccountsDetailCmd(id int) tea.Cmd {
	return func() tea.Msg {
		account, err := SelectAccount(id)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ACCOUNT WITH ID %d: %#v", id, err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.ACCOUNTSAPP,
			Model:     meta.ACCOUNTMODEL,
			Data:      account,
		}
	}
}

func setupSchemaAccounts() (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp("accounts")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS accounts(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = DB.Exec(schema)
	return true, err
}

func (a Account) Insert() (int, error) {
	result, err := DB.NamedExec(`INSERT INTO accounts (name, type, notes) VALUES (:name, :type, :notes)`, a)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Update database cache
	_, err = SelectAccounts()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (a Account) Update() error {
	query := `UPDATE accounts SET
	name = :name,
	type = :type,
	notes = :notes
	WHERE id = :id;`

	_, err := DB.NamedExec(query, a)
	if err != nil {
		return err
	}

	// Update database cache
	_, err = SelectAccounts()

	return err
}

func SelectAccounts() ([]Account, error) {
	var result []Account

	err := DB.Select(&result, `SELECT * FROM accounts;`)
	if err == nil {
		AvailableAccounts = result
	}

	return result, err
}

func SelectAccount(id int) (Account, error) {
	var result Account

	err := DB.Get(&result, `SELECT * FROM accounts WHERE id = $1;`, id)

	return result, err
}

func DeleteAccount(accountId int) error {
	_, err := DB.Exec(`DELETE FROM accounts WHERE id = $1;`, accountId)
	if err != nil {
		return err
	}

	_, err = SelectAccounts()

	return err
}

func MakeSelectAccountsCmd(targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectAccounts()
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ACCOUNTS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.ACCOUNTMODEL,
			Data:      rows,
		}
	}
}
