package database_test

import (
	"terminaccounting/database"
	tat "terminaccounting/tat"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshalLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Note: relying in sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	ledger := database.Ledger{
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

	var result database.Ledger
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result, ledger)
}
