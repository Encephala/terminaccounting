package database_test

import (
	"terminaccounting/database"
	tat "terminaccounting/tat"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshalAccount(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{
		Id:          1, // Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
		Name:        "testerino",
		Type:        database.DEBTOR,
		BankNumbers: []string{"NL02ABNA0123456789"},
		Notes:       []string{"a note"},
	}

	insertedId, err := account.Insert(DB)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, account.Id)

	rows, err := DB.Queryx(`SELECT * FROM accounts;`)
	assert.NoError(t, err)

	var result database.Account
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result, account)
}
