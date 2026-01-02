package database

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDBLedgers(t *testing.T) *sqlx.DB {
	t.Helper()

	DB := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaLedgers(DB)
	require.NoError(t, err)

	return DB
}

func TestMarshalUnmarshalLedger(t *testing.T) {
	DB := setupDBLedgers(t)

	// Note: relying in sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	ledger := Ledger{
		Id:   1,
		Name: "test",
		Type: "INCOME",
		Notes: []string{
			"First note",
			"Second note",
		},
	}

	insertedId, err := ledger.Insert(DB)
	assert.NoError(t, err)

	assert.Equal(t, insertedId, ledger.Id)

	rows, err := DB.Queryx(`SELECT * FROM ledgers;`)
	assert.NoError(t, err)

	var result Ledger
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result, ledger)
}
