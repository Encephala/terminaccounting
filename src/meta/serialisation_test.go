package meta

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalNotes(t *testing.T) {
	db := setupDB(t)

	type Test struct {
		SomeId int   `db:"id"`
		Notes  Notes `db:"notes"`
	}

	expected := Test{
		SomeId: 69,
		Notes: Notes{
			"a note",
			"another one",
		},
	}

	_, err := db.NamedExec(`INSERT INTO test VALUES (:id, :notes);`, expected)
	assert.NoError(t, err)

	rows, err := db.Queryx(`SELECT * FROM test;`)
	assert.NoError(t, err)

	var result Test
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		assert.NoError(t, err)
	}

	assert.Equal(t, count, 1)

	assert.Equal(t, result.Notes, expected.Notes)
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := db.Exec(`CREATE TABLE test(id INTEGER NOT NULL, notes TEXT NOT NULL) STRICT;`)

	require.NoError(t, err)

	return db
}
