package ledgers

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMarshalUnmarshalLedger(t *testing.T) {
	db := setupDB(t)

	ledger := Ledger{
		Id:         1,
		Name:       "test",
		LedgerType: "INCOME",
		Notes:      []string{},
	}

	err := Insert(db, &ledger)
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
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
	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Name != expected.Name {
		t.Errorf("Invalid name %q, expected %q", actual.Name, expected.Name)
	}

	if actual.LedgerType != expected.LedgerType {
		t.Errorf("Invalid type %q, expected %q", actual.LedgerType, expected.LedgerType)
	}

	if len(actual.Notes) != len(expected.Notes) {
		t.Errorf("Unequal notes lengths %d and %d", len(actual.Notes), len(expected.Notes))
	}

	for i, note := range actual.Notes {
		if note != expected.Notes[i] {
			t.Errorf("Unexpected note %q at index %d, expected %q", note, i, expected.Notes[i])
		}
	}
}
