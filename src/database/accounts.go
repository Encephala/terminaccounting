package database

import (
	"fmt"
	"strconv"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AccountType string

const (
	DEBTOR   AccountType = "DEBTOR"
	CREDITOR AccountType = "CREDITOR"
)

type Account struct {
	Id          int         `db:"id"`
	Name        string      `db:"name"`
	AccountType AccountType `db:"type"`
	Notes       meta.Notes  `db:"notes"`
}

func (a Account) String() string {
	// Have id -1 to represent an invalid account
	if a.Id == -1 {
		return lipgloss.NewStyle().Italic(true).Render("None")
	}

	return a.Name + " (" + strconv.Itoa(a.Id) + ")"
}

func SetupSchemaAccounts() (bool, error) {
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

func (a *Account) Insert() (int, error) {
	_, err := DB.NamedExec(`INSERT INTO accounts (name, type, notes) VALUES (:name, :type, :notes)`, a)
	if err != nil {
		return -1, err
	}

	queryInsertedId := DB.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'accounts';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return -1, err
	}

	return insertedId, err
}

func SelectAccounts() ([]Account, error) {
	result := []Account{}

	err := DB.Select(&result, `SELECT * FROM accounts;`)

	return result, err
}

func MakeSelectAccountsCmd(targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectAccounts()
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD ACCOUNTS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.ACCOUNT,
			Data:      rows,
		}
	}
}
