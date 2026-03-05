package database_test

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	account := database.Account{
		Id:          1,
		Name:        "testerino",
		Type:        database.DEBTOR,
		BankNumbers: meta.Notes{"NL02ABNA0123456789"},
		Notes:       meta.Notes{"a note"},
	}

	insertedId, err := account.Insert(DB)
	require.NoError(t, err)
	assert.Equal(t, account.Id, insertedId)

	rows, err := DB.Queryx(`SELECT * FROM accounts;`)
	require.NoError(t, err)

	var result database.Account
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, 1, count)
	assert.Equal(t, account, result)
}

func TestSelectAccounts(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account1 := insertTestAccount(t, DB)
	account2 := insertTestAccount(t, DB)

	result, err := database.SelectAccounts(DB)
	require.NoError(t, err)

	assert.ElementsMatch(t, []database.Account{account1, account2}, result)
}

func TestSelectAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := insertTestAccount(t, DB)

	result, err := database.SelectAccount(DB, account.Id)
	require.NoError(t, err)

	assert.Equal(t, account, result)
}

func TestUpdateAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := insertTestAccount(t, DB)
	account.Name = "updated name"
	account.Type = database.CREDITOR
	account.BankNumbers = meta.Notes{"NL02ABNA0123456789"}
	account.Notes = meta.Notes{"updated note"}

	err := account.Update(DB)
	require.NoError(t, err)

	result, err := database.SelectAccount(DB, account.Id)
	require.NoError(t, err)

	assert.Equal(t, account, result)
}

func TestDeleteAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := insertTestAccount(t, DB)

	err := database.DeleteAccount(DB, account.Id)
	require.NoError(t, err)

	accounts, err := database.SelectAccounts(DB)
	require.NoError(t, err)

	assert.Empty(t, accounts)
}

func TestAccountTypeSerialization(t *testing.T) {
	accountTypes := []database.AccountType{
		database.DEBTOR,
		database.CREDITOR,
	}

	for _, accountType := range accountTypes {
		t.Run(string(accountType), func(t *testing.T) {
			DB := tat.SetupTestEnv(t)

			account := database.Account{
				Name:        "test",
				Type:        accountType,
				BankNumbers: meta.Notes{},
				Notes:       meta.Notes{},
			}
			_, err := account.Insert(DB)
			require.NoError(t, err)

			var result database.Account
			err = DB.Get(&result, `SELECT * FROM accounts;`)
			require.NoError(t, err)

			assert.Equal(t, accountType, result.Type)
		})
	}
}