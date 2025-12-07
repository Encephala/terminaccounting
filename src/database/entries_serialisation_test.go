package database

import (
	"terminaccounting/meta"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func setupDBEntries(t *testing.T) {
	t.Helper()

	DB = sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaEntries()
	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}

	_, err = setupSchemaEntryRows()
	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}
}

func TestMarshalUnmarshalEntry(t *testing.T) {
	setupDBEntries(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	entry := Entry{
		Id:      1,
		Journal: 0,
		Notes:   meta.Notes{},
	}

	time1, err := time.Parse(DATE_FORMAT, "1234-05-06")
	if err != nil {
		panic(err)
	}
	time2, err := time.Parse(DATE_FORMAT, "7890-01-02")
	if err != nil {
		panic(err)
	}

	// SQLITE autoincrements from 1
	entryRows := []EntryRow{
		{
			Id:         1,
			Entry:      0,
			Date:       Date(time1),
			Ledger:     0,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
		{
			Id:         2,
			Entry:      0,
			Date:       Date(time2),
			Ledger:     1,
			Account:    nil,
			Document:   nil,
			Value:      5,
			Reconciled: false,
		},
	}

	insertedId, err := entry.Insert(entryRows)
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
	}

	if insertedId != entry.Id {
		t.Fatalf("Expected id of first inserted journal to be %d, found %d", entry.Id, insertedId)
	}

	rows, err := DB.Queryx(`SELECT * FROM entries;`)
	if err != nil {
		t.Fatalf("Couldn't get rows from database: %v", err)
	}

	var result Entry
	count := 0
	for rows.Next() {
		count++
		err = rows.StructScan(&result)
		if err != nil {
			t.Errorf("Failed to scan: %v", err)
		}
	}

	if count != 1 {
		t.Errorf("Invalid number of rows %d found, expected %d", count, 1)
	}

	testEntriesEqual(t, result, entry)

	rowsEntryRow, err := DB.Queryx(`SELECT * FROM entryrows;`)
	if err != nil {
		t.Fatalf("Couldn't get rows from database: %v", err)
	}

	var row1, row2 EntryRow

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row1)
	if err != nil {
		t.Errorf("Failed to scan: %v", err)
	}

	rowsEntryRow.Next()
	err = rowsEntryRow.StructScan(&row2)
	if err != nil {
		t.Errorf("Failed to scan: %v", err)
	}

	// Check if more rows exist
	count = 2
	for rowsEntryRow.Next() {
		count++
	}
	if count != 2 {
		t.Errorf("Invalid number of rows %d found, expected %d", count, 2)
	}

	testEntryRowsEqual(t, row1, entryRows[0])
	testEntryRowsEqual(t, row2, entryRows[1])
}

func testEntriesEqual(t *testing.T, actual, expected Entry) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Journal != expected.Journal {
		t.Errorf("Invalid Journal %d, expected %d", actual.Journal, expected.Journal)
	}

	if len(actual.Notes) != len(expected.Notes) {
		t.Errorf("Unequal notes lengths %d and %d", len(actual.Notes), len(expected.Notes))
		t.Logf("Actual notes %v, expected %v", actual.Notes, expected.Notes)
	}

	for i, note := range actual.Notes {
		if note != expected.Notes[i] {
			t.Errorf("Invalid note %q at index %d, expected %q", actual.Notes, i, expected.Notes)
		}
	}
}

func testEntryRowsEqual(t *testing.T, actual, expected EntryRow) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Entry != expected.Entry {
		t.Errorf("Invalid Entry %d, expected %d", actual.Entry, expected.Entry)
	}

	if !time.Time(actual.Date).Equal(time.Time(expected.Date)) {
		t.Errorf("Invalid Date %v, expected %v", actual.Date, expected.Date)
	}

	if actual.Ledger != expected.Ledger {
		t.Errorf("Invalid Ledger %d, expected %d", actual.Ledger, expected.Ledger)
	}

	if (actual.Account == nil) != (expected.Account == nil) ||
		(actual.Account != nil && expected.Account != nil && *actual.Account != *expected.Account) {
		t.Errorf("Invalid Account %v, expected %v", actual.Account, expected.Account)
	}

	if (actual.Document == nil) != (expected.Document == nil) ||
		(actual.Document != nil && expected.Document != nil && *actual.Document != *expected.Document) {
		t.Errorf("Invalid Document %v, expected %v", actual.Document, expected.Document)
	}

	if actual.Value != expected.Value {
		t.Errorf("Invalid Value %d, expected %d", actual.Value, expected.Value)
	}

	if actual.Reconciled != expected.Reconciled {
		t.Errorf("Invalid Reconciled %v, expected %v", actual.Reconciled, expected.Reconciled)
	}
}
