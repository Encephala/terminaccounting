package database

import (
	"fmt"
	"strings"
	"sync/atomic"
	"terminaccounting/meta"

	"terminaccounting/bubbles/itempicker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

// Globally accessible list of available journals
// Atomic for parallel tests
var journalsCache atomic.Pointer[[]Journal]

func AvailableJournals() []Journal {
	return *journalsCache.Load()
}

func AvailableJournalsAsItempickerItems() []itempicker.Item {
	var result []itempicker.Item

	for _, journal := range *journalsCache.Load() {
		result = append(result, journal)
	}

	return result
}

type JournalType string

const (
	INCOMEJOURNAL   JournalType = "INCOME"
	EXPENSEJOURNAL  JournalType = "EXPENSE"
	CASHFLOWJOURNAL JournalType = "CASHFLOW"
	GENERALJOURNAL  JournalType = "GENERAL"
)

func (jt JournalType) String() string {
	return string(jt)
}

func (jt JournalType) CompareId() int {
	switch jt {
	case INCOMEJOURNAL:
		return 0
	case EXPENSEJOURNAL:
		return 1
	case CASHFLOWJOURNAL:
		return 2
	case GENERALJOURNAL:
		return 3
	default:
		panic(fmt.Sprintf("unexpected database.AccountType: %#v", jt))
	}
}

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

func (j Journal) Title() string {
	return j.Name
}

func (j Journal) Description() string {
	return j.Notes.Collapse()
}

func MakeLoadJournalsDetailCmd(DB *sqlx.DB, id int) tea.Cmd {
	return func() tea.Msg {
		journal, err := SelectJournal(DB, id)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD JOURNAL WITH ID %d: %#v", id, err)
		}

		return meta.DataLoadedMsg{
			TargetApp: meta.JOURNALSAPP,
			Model:     meta.JOURNALMODEL,
			Data:      journal,
		}
	}
}

func setupSchemaJournals(DB *sqlx.DB) (bool, error) {
	isSetUp, err := DatabaseTableIsSetUp(DB, "journals")
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

func (j *Journal) Insert(DB *sqlx.DB) (int, error) {
	_, err := DB.NamedExec(`INSERT INTO journals (name, type, notes) VALUES (:name, :type, :notes)`, j)
	if err != nil {
		return 0, err
	}

	queryId := DB.QueryRowx(`SELECT seq FROM sqlite_sequence WHERE name = 'journals';`)

	var id int
	err = queryId.Scan(&id)
	if err != nil {
		return 0, err
	}

	err = UpdateJournalsCache(DB)
	if err != nil {
		return int(id), err
	}

	return id, nil
}

func (j *Journal) Update(DB *sqlx.DB) error {
	query := `UPDATE journals SET
	name = :name,
	type = :type,
	notes = :notes
	WHERE id = :id;`

	_, err := DB.NamedExec(query, j)
	if err != nil {
		return err
	}

	err = UpdateJournalsCache(DB)

	return err
}

func SelectJournals(DB *sqlx.DB) ([]Journal, error) {
	var result []Journal

	err := DB.Select(&result, `SELECT * FROM journals;`)

	return result, err
}

func UpdateJournalsCache(DB *sqlx.DB) error {
	journals, err := SelectJournals(DB)
	if err != nil {
		return err
	}

	journalsCache.Store(&journals)

	return nil
}

func SelectJournal(DB *sqlx.DB, id int) (Journal, error) {
	result := Journal{}

	err := DB.Get(&result, `SELECT * FROM journals WHERE id = $1;`, id)

	return result, err
}

func DeleteJournal(DB *sqlx.DB, id int) error {
	_, err := DB.Exec(`DELETE FROM journals WHERE id = $1;`, id)
	if err != nil {
		return err
	}

	err = UpdateJournalsCache(DB)

	return err
}

func MakeSelectJournalsCmd(DB *sqlx.DB, targetApp meta.AppType) tea.Cmd {
	return func() tea.Msg {
		rows, err := SelectJournals(DB)
		if err != nil {
			return fmt.Errorf("FAILED TO LOAD JOURNALS: %v", err)
		}

		return meta.DataLoadedMsg{
			TargetApp: targetApp,
			Model:     meta.JOURNALMODEL,
			Data:      rows,
		}
	}
}
