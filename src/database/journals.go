package database

import (
	"fmt"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
)

type JournalType string

const (
	INCOMEJOURNAL   JournalType = "INCOME"
	EXPENSEJOURNAL  JournalType = "EXPENSE"
	CASHFLOWJOURNAL JournalType = "CASHFLOW"
	GENERALJOURNAL  JournalType = "GENERAL"
)

type Journal struct {
	Id    int         `db:"id"`
	Name  string      `db:"name"`
	Type  JournalType `db:"type"`
	Notes meta.Notes  `db:"notes"`
}

func (j Journal) String() string {
	return j.Name
}

func (j Journal) CompareId() int {
	return j.Id
}

func (j Journal) FilterValue() string {
	var result strings.Builder

	result.WriteString(j.Name)
	result.WriteString(string(j.Type))
	result.WriteString(j.Notes.Collapse())

	return result.String()
}

func SetupSchemaJournals() (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp("journals")
	if err != nil {
		return false, err
	}
	if isSetUp {
		return false, nil
	}

	schema := `CREATE TABLE IF NOT EXISTS journals(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = DB.Exec(schema)
	return true, err
}

func (j *Journal) Insert() (int, error) {
	_, err := DB.NamedExec(`INSERT INTO journals (name, type, notes) VALUES (:name, :type, :notes)`, j)
	if err != nil {
		return 0, err
	}

	queryInsertedId := DB.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'journals';`)

	var insertedId int
	err = queryInsertedId.Scan(&insertedId)
	if err != nil {
		return 0, err
	}

	return insertedId, err
}

func SelectJournals() ([]Journal, error) {
	result := []Journal{}

	err := DB.Select(&result, `SELECT * FROM journals;`)

	return result, err
}

func SelectJournal(id int) (Journal, error) {
	result := Journal{}

	err := DB.Get(&result, `SELECT * FROM journals WHERE id = $1;`, id)

	return result, err
}

func MakeSelectJournalsCmd(targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectJournals()
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD JOURNALS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.JOURNAL,
			Data:      rows,
		}
	}
}
