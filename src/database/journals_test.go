package database_test

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	journal := database.Journal{
		Id:    1,
		Name:  "test",
		Type:  database.CASHFLOWJOURNAL,
		Notes: meta.Notes{"a note"},
	}

	insertedId, err := journal.Insert(DB)
	require.NoError(t, err)
	assert.Equal(t, journal.Id, insertedId)

	rows, err := DB.Queryx(`SELECT * FROM journals;`)
	require.NoError(t, err)

	var result database.Journal
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, count)
	assert.Equal(t, journal, result)
}

func TestSelectJournals(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal1 := insertTestJournal(t, DB)
	journal2 := insertTestJournal(t, DB)

	result, err := database.SelectJournals(DB)
	require.NoError(t, err)

	assert.ElementsMatch(t, []database.Journal{journal1, journal2}, result)
}

func TestSelectJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)

	result, err := database.SelectJournal(DB, journal.Id)
	require.NoError(t, err)

	assert.Equal(t, journal, result)
}

func TestUpdateJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	journal.Name = "updated name"
	journal.Type = database.INCOMEJOURNAL
	journal.Notes = meta.Notes{"updated note"}

	err := journal.Update(DB)
	require.NoError(t, err)

	result, err := database.SelectJournal(DB, journal.Id)
	require.NoError(t, err)

	assert.Equal(t, journal, result)
}

func TestDeleteJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)

	err := database.DeleteJournal(DB, journal.Id)
	require.NoError(t, err)

	journals, err := database.SelectJournals(DB)
	require.NoError(t, err)

	assert.Empty(t, journals)
}

func TestJournalTypeSerialization(t *testing.T) {
	journalTypes := []database.JournalType{
		database.INCOMEJOURNAL,
		database.EXPENSEJOURNAL,
		database.CASHFLOWJOURNAL,
		database.GENERALJOURNAL,
	}

	for _, journalType := range journalTypes {
		t.Run(string(journalType), func(t *testing.T) {
			DB := tat.SetupTestEnv(t)

			journal := database.Journal{
				Name:  "test",
				Type:  journalType,
				Notes: meta.Notes{},
			}
			_, err := journal.Insert(DB)
			require.NoError(t, err)

			var result database.Journal
			err = DB.Get(&result, `SELECT * FROM journals;`)
			require.NoError(t, err)

			assert.Equal(t, journalType, result.Type)
		})
	}
}

func TestNotesSerialization(t *testing.T) {
	tests := []struct {
		name  string
		notes meta.Notes
	}{
		{"empty", meta.Notes{}},
		{"single", meta.Notes{"one note"}},
		{"multiple", meta.Notes{"first note", "second note", "third note"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			DB := tat.SetupTestEnv(t)

			journal := database.Journal{
				Name:  "test",
				Type:  database.GENERALJOURNAL,
				Notes: tc.notes,
			}
			_, err := journal.Insert(DB)
			require.NoError(t, err)

			var result database.Journal
			err = DB.Get(&result, `SELECT * FROM journals;`)
			require.NoError(t, err)

			assert.Equal(t, tc.notes, result.Notes)
		})
	}
}