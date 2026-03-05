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

func TestParseCurrencyValue(t *testing.T) {
	tests := []struct {
		input    string
		expected database.CurrencyValue
		hasError bool
	}{
		{"0", 0, false},
		{"1", 100, false},
		{"100", 10000, false},
		{"1.", 100, false},
		{"1.0", 100, false},
		{"1.5", 150, false},
		{"1.50", 150, false},
		{"1.09", 109, false},
		{"0.99", 99, false},
		{"abc", 0, true},
		{"1.abc", 0, true},
		{"1.123", 0, true},
		{"1.2.3", 0, true},
	}

	for _, testCase := range tests {
		result, err := database.ParseCurrencyValue(testCase.input)
		if testCase.hasError {
			assert.Error(t, err, "input: %q", testCase.input)
		} else {
			require.NoError(t, err, "input: %q", testCase.input)
			assert.Equal(t, testCase.expected, result, "input: %q", testCase.input)
		}
	}
}

func TestCurrencyValueString(t *testing.T) {
	tests := []struct {
		value    database.CurrencyValue
		expected string
	}{
		{0, "0.00"},
		{100, "1.00"},
		{150, "1.50"},
		{109, "1.09"},
		{99, "0.99"},
		{10000, "100.00"},
		{-100, "-1.00"},
		{-150, "-1.50"},
		{-99, "-0.99"},
	}

	for _, testCase := range tests {
		assert.Equal(t, testCase.expected, testCase.value.String())
	}
}

func TestCurrencyValueAddSubtract(t *testing.T) {
	assert.Equal(t, database.CurrencyValue(300), database.CurrencyValue(100).Add(database.CurrencyValue(200)))
	assert.Equal(t, database.CurrencyValue(100), database.CurrencyValue(300).Subtract(database.CurrencyValue(200)))
	assert.Equal(t, database.CurrencyValue(-100), database.CurrencyValue(100).Subtract(database.CurrencyValue(200)))
	assert.Equal(t, database.CurrencyValue(0), database.CurrencyValue(100).Add(database.CurrencyValue(-100)))
}

func TestCalculateTotal(t *testing.T) {
	rows := []*database.EntryRow{
		{Value: 1000},
		{Value: -500},
		{Value: 250},
	}

	assert.Equal(t, database.CurrencyValue(750), database.CalculateTotal(rows))
}

func TestCalculateSize(t *testing.T) {
	rows := []*database.EntryRow{
		{Value: 1000},
		{Value: -500},
		{Value: 250},
	}

	assert.Equal(t, database.CurrencyValue(1250), database.CalculateSize(rows))
}

func TestSelectRowsByEntry(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry1 := insertTestEntry(t, DB, journal.Id, ledger.Id)
	entry2 := insertTestEntry(t, DB, journal.Id, ledger.Id)

	rows1, err := database.SelectRowsByEntry(DB, entry1.Id)
	require.NoError(t, err)
	require.Len(t, rows1, 1)
	assert.Equal(t, entry1.Id, rows1[0].Entry)

	rows2, err := database.SelectRowsByEntry(DB, entry2.Id)
	require.NoError(t, err)
	require.Len(t, rows2, 1)
	assert.Equal(t, entry2.Id, rows2[0].Entry)
}

func TestSelectRowsByLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	ledger1 := insertTestLedger(t, DB)
	ledger2 := insertTestLedger(t, DB)
	insertTestEntry(t, DB, journal.Id, ledger1.Id)
	insertTestEntry(t, DB, journal.Id, ledger2.Id)

	rows, err := database.SelectRowsByLedger(DB, ledger1.Id)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, ledger1.Id, rows[0].Ledger)
}

func TestSelectRowsByJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal1 := insertTestJournal(t, DB)
	journal2 := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	entry1 := insertTestEntry(t, DB, journal1.Id, ledger.Id)
	insertTestEntry(t, DB, journal2.Id, ledger.Id)

	rows, err := database.SelectRowsByJournal(DB, journal1.Id)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, entry1.Id, rows[0].Entry)
}

func TestSetReconciled(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)
	insertTestEntry(t, DB, journal.Id, ledger.Id)

	rows, err := database.SelectRows(DB)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.False(t, rows[0].Reconciled)

	rows[0].Reconciled = true
	changed, err := database.SetReconciled(DB, []*database.EntryRow{&rows[0]})
	require.NoError(t, err)
	assert.Equal(t, 1, changed)

	updatedRows, err := database.SelectRows(DB)
	require.NoError(t, err)
	require.Len(t, updatedRows, 1)
	assert.True(t, updatedRows[0].Reconciled)
}

func TestSetReconciledMultipleRows(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journal.Id, Notes: meta.Notes{}}
	_, err = entry.Insert(DB, []database.EntryRow{
		{Date: date, Ledger: ledger.Id, Value: 1000},
		{Date: date, Ledger: ledger.Id, Value: -1000},
	})
	require.NoError(t, err)

	rows, err := database.SelectRows(DB)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	rows[0].Reconciled = true
	rows[1].Reconciled = true
	changed, err := database.SetReconciled(DB, []*database.EntryRow{&rows[0], &rows[1]})
	require.NoError(t, err)
	assert.Equal(t, 2, changed)

	updatedRows, err := database.SelectRows(DB)
	require.NoError(t, err)
	assert.True(t, updatedRows[0].Reconciled)
	assert.True(t, updatedRows[1].Reconciled)
}

func TestSetReconciledReturnsChangedCount(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	ledger := insertTestLedger(t, DB)

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journal.Id, Notes: meta.Notes{}}
	_, err = entry.Insert(DB, []database.EntryRow{
		{Date: date, Ledger: ledger.Id, Value: 500},
		{Date: date, Ledger: ledger.Id, Value: -500},
	})
	require.NoError(t, err)

	rows, err := database.SelectRows(DB)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	// Only reconcile one of two rows
	rows[0].Reconciled = true
	changed, err := database.SetReconciled(DB, []*database.EntryRow{&rows[0]})
	require.NoError(t, err)
	assert.Equal(t, 1, changed)
}

func TestSelectRowsByAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	journal := insertTestJournal(t, DB)
	account := insertTestAccount(t, DB)

	accountsLedger := database.Ledger{
		Name:       "accounts ledger",
		Type:       database.INCOMELEDGER,
		Notes:      meta.Notes{},
		IsAccounts: true,
	}
	accountsLedgerId, err := accountsLedger.Insert(DB)
	require.NoError(t, err)

	regularLedger := insertTestLedger(t, DB)

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entryWithAccount := database.Entry{Journal: journal.Id, Notes: meta.Notes{}}
	_, err = entryWithAccount.Insert(DB, []database.EntryRow{
		{Date: date, Ledger: accountsLedgerId, Account: &account.Id, Value: 500},
	})
	require.NoError(t, err)

	// Row on a non-accounts ledger should not appear
	entryWithoutAccountsLedger := database.Entry{Journal: journal.Id, Notes: meta.Notes{}}
	_, err = entryWithoutAccountsLedger.Insert(DB, []database.EntryRow{
		{Date: date, Ledger: regularLedger.Id, Account: &account.Id, Value: 300},
	})
	require.NoError(t, err)

	rows, err := database.SelectRowsByAccount(DB, account.Id)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, accountsLedgerId, rows[0].Ledger)
	assert.Equal(t, &account.Id, rows[0].Account)
}
