package journals

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestMarshalUnmarshalJournal(t *testing.T) {
	db := setupDB(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	journal := Journal{
		Id:          1,
		Name:        "test",
		JournalType: CASHFLOW,
		Notes:       []string{"a note"},
	}

	insertedId, err := journal.Insert(db)
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
	}

	if insertedId != journal.Id {
		t.Fatalf("Expected id of first inserted journal to be %d, found %d", 1, insertedId)
	}

	rows, err := db.Queryx(`SELECT * FROM journals;`)
	if err != nil {
		t.Fatalf("Couldn't get rows from database: %v", err)
	}

	fmt.Println("dicks")

	var result Journal
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		if err != nil {
			t.Errorf("Failed to scan: %v", err)
		}
	}

	if count != 1 {
		t.Errorf("Invalid number of rows %d found, expected 1", count)
	}

	testJournalsEqual(t, result, journal)
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

func testJournalsEqual(t *testing.T, actual, expected Journal) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Name != expected.Name {
		t.Errorf("Invalid name %q, expected %q", actual.Name, expected.Name)
	}

	if actual.JournalType != expected.JournalType {
		t.Errorf("Invalid ID %q, expected %q", actual.JournalType, expected.JournalType)
	}

	// if len(actual.Notes) != len(expected.Notes) {
	// 	t.Errorf("Unequal notes lengths %d and %d", len(actual.Notes), len(expected.Notes))
	// 	t.Logf("Actual notes %v, expected %v", actual.Notes, expected.Notes)
	// }

	// for i, note := range actual.Notes {
	// 	if note != expected.Notes[i] {
	// 		t.Errorf("Invalid note %q at index %d, expected %q", actual.Notes, i, expected.Notes)
	// 	}
	// }
}
