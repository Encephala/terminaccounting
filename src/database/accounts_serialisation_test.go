package database

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDBAccounts(t *testing.T) *sqlx.DB {
	t.Helper()

	DB := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaAccounts(DB)
	require.NoError(t, err)

	return DB
}

func TestMarshalUnmarshalAccount(t *testing.T) {
	DB := setupDBAccounts(t)

	account := Account{
		Id:          1, // Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
		Name:        "testerino",
		Type:        DEBTOR,
		BankNumbers: []string{"NL02ABNA0123456789"},
		Notes:       []string{"a note"},
	}

	insertedId, err := account.Insert(DB)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, account.Id)

	rows, err := DB.Queryx(`SELECT * FROM accounts;`)
	assert.NoError(t, err)

	var result Account
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result, account)
}
