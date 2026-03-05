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

func TestInsertEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)

	time1, err := time.Parse(database.DATE_FORMAT, "1234-05-06")
	require.NoError(t, err)

	time2, err := time.Parse(database.DATE_FORMAT, "7890-01-02")
	require.NoError(t, err)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	entry := database.Entry{
		Id:      1,
		Journal: journal.Id,
		Notes:   meta.Notes{},
	}
	entryRows := []database.EntryRow{
		{
			Id:         1,
			Date:       database.Date(time1),
			Ledger:     ledger.Id,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
		{
			Id:         2,
			Date:       database.Date(time2),
			Ledger:     ledger.Id,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
	}

	insertedId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)
	assert.Equal(t, entry.Id, insertedId)

	rows, err := DB.Queryx(`SELECT * FROM entries;`)
	require.NoError(t, err)

	var result database.Entry
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, count)
	assert.Equal(t, entry, result)

	rowsEntryRow, err := DB.Queryx(`SELECT * FROM entryrows;`)
	require.NoError(t, err)

	var row1, row2 database.EntryRow

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row1)
	assert.NoError(t, err)

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row2)
	assert.NoError(t, err)

	count = 2
	for rowsEntryRow.Next() {
		count++
	}
	assert.Equal(t, 2, count)

	assert.Equal(t, entryRows[0], row1)
	assert.Equal(t, entryRows[1], row2)
}

func TestSelectEntries(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry1 := insertTestEntry(t, DB, journal.Id, ledger.Id)
	entry2 := insertTestEntry(t, DB, journal.Id, ledger.Id)

	result, err := database.SelectEntries(DB)
	require.NoError(t, err)

	assert.ElementsMatch(t, []database.Entry{entry1, entry2}, result)
}

func TestSelectEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry := insertTestEntry(t, DB, journal.Id, ledger.Id)

	result, err := database.SelectEntry(DB, entry.Id)
	require.NoError(t, err)

	assert.Equal(t, entry, result)
}

func TestUpdateEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry := insertTestEntry(t, DB, journal.Id, ledger.Id)

	updatedEntry := database.Entry{
		Id:      entry.Id,
		Journal: journal.Id,
		Notes:   meta.Notes{"updated note"},
	}
	newDate, err := database.ToDate("2025-06-15")
	require.NoError(t, err)

	updatedRows := []database.EntryRow{
		{
			Date:        newDate,
			Ledger:      ledger.Id,
			Account:     nil,
			Description: "updated row",
			Document:    nil,
			Value:       2000,
			Reconciled:  false,
		},
	}

	err = updatedEntry.Update(DB, updatedRows)
	require.NoError(t, err)

	resultEntry, err := database.SelectEntry(DB, entry.Id)
	require.NoError(t, err)
	assert.Equal(t, updatedEntry, resultEntry)

	resultRows, err := database.SelectRowsByEntry(DB, entry.Id)
	require.NoError(t, err)
	require.Len(t, resultRows, 1)
	assert.Equal(t, entry.Id, resultRows[0].Entry)
	assert.Equal(t, newDate, resultRows[0].Date)
	assert.Equal(t, database.CurrencyValue(2000), resultRows[0].Value)
	assert.Equal(t, "updated row", resultRows[0].Description)
}

func TestDeleteEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry := insertTestEntry(t, DB, journal.Id, ledger.Id)

	err := database.DeleteEntry(DB, entry.Id)
	require.NoError(t, err)

	entries, err := database.SelectEntries(DB)
	require.NoError(t, err)

	assert.Empty(t, entries)
}
