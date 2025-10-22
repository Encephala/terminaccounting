package meta

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMarshalUnmarshalNotes(t *testing.T) {
	db := setupDB(t)

	type Test struct {
		someId int   `db:"id"`
		notes  Notes `db:"notes"`
	}

	expected := Test{
		someId: 69,
		notes: Notes{
			"a note",
			"another one",
		},
	}

	_, err := db.NamedExec(`INSERT INTO test VALUES (:id, :notes);`, expected)

	if err != nil {
		t.Fatalf("Failed to insert notes into database: %v", err)
	}

	rows, err := db.Queryx(`SELECT * FROM test;`)
	if err != nil {
		t.Fatalf("Couldn't query rows from database: %v", err)
	}

	var result Test
	count := 0
	for rows.Next() {
		count++
		rows.StructScan(&result)
		if err != nil {
			t.Errorf("Failed to scan: %v", err)
		}
	}

	if count != 1 {
		t.Errorf("Invalid number of rows %d found, expected 1", count)
	}

	if len(result.notes) != len(expected.notes) {
		t.Errorf("Unequal notes lengths %d and %d", len(result.notes), len(expected.notes))
		t.Logf("Actual notes %v, expected %v", result, expected)
	}

	for i, note := range result.notes {
		if note != expected.notes[i] {
			t.Errorf("Unexpected note %q at index %d, expected %q", note, i, expected.notes[i])
		}
	}
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := db.Exec(`CREATE TABLE test(id INTEGER NOT NULL, notes TEXT NOT NULL) STRICT;`)

	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}

	return db
}
