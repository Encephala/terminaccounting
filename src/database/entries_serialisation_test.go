package database_test

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	entry := database.Entry{
		Id:      1,
		Journal: 0,
		Notes:   meta.Notes{},
	}

	time1, err := time.Parse(database.DATE_FORMAT, "1234-05-06")
	require.NoError(t, err)

	time2, err := time.Parse(database.DATE_FORMAT, "7890-01-02")
	require.NoError(t, err)

	// SQLITE autoincrements from 1
	entryRows := []database.EntryRow{
		{
			Id:         1,
			Entry:      0,
			Date:       database.Date(time1),
			Ledger:     0,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
		{
			Id:         2,
			Entry:      0,
			Date:       database.Date(time2),
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

	var result database.Entry
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

	var row1, row2 database.EntryRow

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
