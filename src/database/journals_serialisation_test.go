package database

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDBJournals(t *testing.T) *sqlx.DB {
	t.Helper()

	DB := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaJournals(DB)

	require.NoError(t, err)

	return DB
}

func TestMarshalUnmarshalJournal(t *testing.T) {
	DB := setupDBJournals(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	journal := Journal{
		Id:    1,
		Name:  "test",
		Type:  CASHFLOWJOURNAL,
		Notes: []string{"a note"},
	}

	insertedId, err := journal.Insert(DB)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, journal.Id)

	rows, err := DB.Queryx(`SELECT * FROM journals;`)
	assert.NoError(t, err)

	var result Journal
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
