package database_test

import (
	"terminaccounting/database"
	"testing"

	"github.com/jmoiron/sqlx"
)

func setupDBAccounts(t *testing.T) {
	t.Helper()

	database.DB = sqlx.MustConnect("sqlite3", ":memory:")
	_, err := database.SetupSchemaAccounts()

	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}
}

func TestMarshalUnmarshalAccount(t *testing.T) {
	setupDBAccounts(t)

	// Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
	account := database.Account{
		Id:          1,
		Name:        "testerino",
		AccountType: database.DEBTOR,
		Notes:       []string{"a note"},
	}

	insertedId, err := account.Insert()
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
	}

	if insertedId != account.Id {
		t.Fatalf("Expected id of first inserted account to be %d, found %d", account.Id, insertedId)
	}

	rows, err := database.DB.Queryx(`SELECT * FROM accounts;`)
	if err != nil {
		t.Fatalf("Couldn't get rows from database: %v", err)
	}

	var result database.Account
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

	testAccountsEqual(t, result, account)
}

func testAccountsEqual(t *testing.T, actual, expected database.Account) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Name != expected.Name {
		t.Errorf("Invalid name %q, expected %q", actual.Name, expected.Name)
	}

	if actual.AccountType != expected.AccountType {
		t.Errorf("Invalid ID %q, expected %q", actual.AccountType, expected.AccountType)
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
