package database_test

import (
	"terminaccounting/database"
	"terminaccounting/tat"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshalJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	journal := database.Journal{
		Id:    1,
		Name:  "test",
		Type:  database.CASHFLOWJOURNAL,
		Notes: []string{"a note"},
	}

	insertedId, err := journal.Insert(DB)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, journal.Id)

	rows, err := DB.Queryx(`SELECT * FROM journals;`)
	assert.NoError(t, err)

	var result database.Journal
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)
	if count != 1 {
		t.Errorf("Invalid number of rows %d found, expected 1", count)
	}

	assert.Equal(t, result, journal)
}
