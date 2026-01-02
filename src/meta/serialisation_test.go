package meta_test

import (
	"terminaccounting/meta"
	tat "terminaccounting/tatesting"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestMarshalUnmarshalNotes(t *testing.T) {
	db := tat.SetupTestDB(t)

	_, err := db.Exec(`CREATE TABLE test(id INTEGER NOT NULL, notes TEXT NOT NULL) STRICT;`)
	assert.NoError(t, err)

	type Test struct {
		SomeId int        `db:"id"`
		Notes  meta.Notes `db:"notes"`
	}

	expected := Test{
		SomeId: 69,
		Notes: meta.Notes{
			"a note",
			"another one",
		},
	}

	_, err = db.NamedExec(`INSERT INTO test VALUES (:id, :notes);`, expected)
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
