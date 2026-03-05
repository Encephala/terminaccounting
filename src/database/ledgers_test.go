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

func TestInsertLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	ledger := database.Ledger{
		Id:   1,
		Name: "test",
		Type: database.INCOMELEDGER,
		Notes: meta.Notes{
			"First note",
			"Second note",
		},
	}

	insertedId, err := ledger.Insert(DB)
	require.NoError(t, err)
	assert.Equal(t, ledger.Id, insertedId)

	rows, err := DB.Queryx(`SELECT * FROM ledgers;`)
	require.NoError(t, err)

	var result database.Ledger
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, count)
	assert.Equal(t, ledger, result)
}

func TestSelectLedgers(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger1 := insertTestLedger(t, DB)
	ledger2 := insertTestLedger(t, DB)

	result, err := database.SelectLedgers(DB)
	require.NoError(t, err)

	assert.ElementsMatch(t, []database.Ledger{ledger1, ledger2}, result)
}

func TestSelectLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := insertTestLedger(t, DB)

	result, err := database.SelectLedger(DB, ledger.Id)
	require.NoError(t, err)

	assert.Equal(t, ledger, result)
}

func TestUpdateLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := insertTestLedger(t, DB)
	ledger.Name = "updated name"
	ledger.Type = database.EXPENSELEDGER
	ledger.Notes = meta.Notes{"a new note"}

	err := ledger.Update(DB)
	require.NoError(t, err)

	result, err := database.SelectLedger(DB, ledger.Id)
	require.NoError(t, err)

	assert.Equal(t, ledger, result)
}

func TestDeleteLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := insertTestLedger(t, DB)

	err := database.DeleteLedger(DB, ledger.Id)
	require.NoError(t, err)

	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)

	assert.Empty(t, ledgers)
}

func TestLedgerTypeSerialization(t *testing.T) {
	ledgerTypes := []database.LedgerType{
		database.INCOMELEDGER,
		database.EXPENSELEDGER,
		database.ASSETLEDGER,
		database.LIABILITYLEDGER,
		database.EQUITYLEDGER,
	}

	for _, ledgerType := range ledgerTypes {
		t.Run(string(ledgerType), func(t *testing.T) {
			DB := tat.SetupTestEnv(t)

			ledger := database.Ledger{
				Name:       "test",
				Type:       ledgerType,
				Notes:      meta.Notes{},
				IsAccounts: false,
			}
			_, err := ledger.Insert(DB)
			require.NoError(t, err)

			var result database.Ledger
			err = DB.Get(&result, `SELECT * FROM ledgers;`)
			require.NoError(t, err)

			assert.Equal(t, ledgerType, result.Type)
		})
	}
}