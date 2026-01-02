package database

import (
	"terminaccounting/meta"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDBEntries(t *testing.T) *sqlx.DB {
	t.Helper()

	DB := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaEntries(DB)
	require.NoError(t, err)

	_, err = setupSchemaEntryRows(DB)
	require.NoError(t, err)

	return DB
}

func TestMarshalUnmarshalEntry(t *testing.T) {
	DB := setupDBEntries(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	entry := Entry{
		Id:      1,
		Journal: 0,
		Notes:   meta.Notes{},
	}

	time1, err := time.Parse(DATE_FORMAT, "1234-05-06")
	require.NoError(t, err)

	time2, err := time.Parse(DATE_FORMAT, "7890-01-02")
	require.NoError(t, err)

	// SQLITE autoincrements from 1
	entryRows := []EntryRow{
		{
			Id:         1,
			Entry:      0,
			Date:       Date(time1),
			Ledger:     0,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
		{
			Id:         2,
			Entry:      0,
			Date:       Date(time2),
			Ledger:     1,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
	}

	insertedId, err := entry.Insert(DB, entryRows)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, entry.Id)

	rows, err := DB.Queryx(`SELECT * FROM entries;`)
	assert.NoError(t, err)

	var result Entry
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result, entry)

	rowsEntryRow, err := DB.Queryx(`SELECT * FROM entryrows;`)
	assert.NoError(t, err)

	var row1, row2 EntryRow

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row1)
	assert.NoError(t, err)

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row2)
	assert.NoError(t, err)

	// Check if more rows exist
	count = 2
	for rowsEntryRow.Next() {
		count++
	}
	assert.Equal(t, count, 2)

	assert.Equal(t, row1, entryRows[0])
	assert.Equal(t, row2, entryRows[1])
}
