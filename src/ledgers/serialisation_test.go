package ledgers

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMarshalUnmarshalLedger(t *testing.T) {
	db := setupDB(t)

	ledger := Ledger{
		id:         0,
		name:       "test",
		ledgerType: "INCOME",
		notes:      []string{},
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

	app := app{}

	db := sqlx.MustConnect("sqlite3", ":memory:")
	_, err := app.SetupSchema(db)

	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}

	return db
}

func testLedgersEqual(t *testing.T, actual, expected Ledger) {
	if actual.id != expected.id {
		t.Errorf("Invalid ID %d, expected %d", actual.id, expected.id)
	}

	if actual.name != expected.name {
		t.Errorf("Invalid name %q, expected %q", actual.name, expected.name)
	}

	if actual.ledgerType != expected.ledgerType {
		t.Errorf("Invalid type %q, expected %q", actual.ledgerType, expected.ledgerType)
	}

	if len(actual.notes) != len(expected.notes) {
		t.Errorf("Unequal notes lengths %d and %d", len(actual.notes), len(expected.notes))
	}

	for i, note := range actual.notes {
		if note != expected.notes[i] {
			t.Errorf("Unexpected note %q at index %d, expected %q", note, i, expected.notes[i])
		}
	}
}
