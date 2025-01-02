package ledgers

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMarshalUnmarshalLedger(t *testing.T) {
	db := setupDB(t)

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

	insertedId, err := ledger.insert(db)
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
	}

	if insertedId != 1 {
		t.Fatalf("Expected id of first inserted ledger to be %d, found %d", 1, insertedId)
	}

	rows, err := db.Queryx(`SELECT * FROM ledgers;`)
	if err != nil {
		t.Fatalf("Couldn't query rows from database: %v", err)
	}

	var result Ledger
	count := 0
	for rows.Next() {
		count++
		rows.StructScan(&result)
	}

	if count != 1 {
		t.Errorf("Invalid number of rows %d found, expected 1", count)
	}

	testLedgersEqual(t, result, ledger)
}

func setupDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchema(db)

	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}

	return db
}

func testLedgersEqual(t *testing.T, actual, expected Ledger) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Name != expected.Name {
		t.Errorf("Invalid name %q, expected %q", actual.Name, expected.Name)
	}

	if actual.Type != expected.Type {
		t.Errorf("Invalid type %q, expected %q", actual.Type, expected.Type)
	}

	if len(actual.Notes) != len(expected.Notes) {
		t.Errorf("Unequal notes lengths %d and %d", len(actual.Notes), len(expected.Notes))
		t.Logf("Actual notes %v, expected %v", actual.Notes, expected.Notes)
	}

	for i, note := range actual.Notes {
		if note != expected.Notes[i] {
			t.Errorf("Unexpected note %q at index %d, expected %q", note, i, expected.Notes[i])
		}
	}
}
