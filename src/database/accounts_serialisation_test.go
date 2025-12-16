package database

import (
	"slices"
	"testing"

	"github.com/jmoiron/sqlx"
)

func setupDBAccounts(t *testing.T) {
	t.Helper()

	DB = sqlx.MustConnect("sqlite3", ":memory:")
	_, err := setupSchemaAccounts()

	if err != nil {
		t.Fatalf("Couldn't setup db: %v", err)
	}
}

func TestMarshalUnmarshalAccount(t *testing.T) {
	setupDBAccounts(t)

	account := Account{
		Id:          1, // Note: relying on sqlite default behaviour of starting PRIMARY KEY AUTOINCREMENT at 1
		Name:        "testerino",
		Type:        DEBTOR,
		BankNumbers: []string{"NL02ABNA0123456789"},
		Notes:       []string{"a note"},
	}

	insertedId, err := account.Insert()
	if err != nil {
		t.Fatalf("Couldn't insert into database: %v", err)
	}

	if insertedId != account.Id {
		t.Fatalf("Expected id of first inserted account to be %d, found %d", account.Id, insertedId)
	}

	rows, err := DB.Queryx(`SELECT * FROM accounts;`)
	if err != nil {
		t.Fatalf("Couldn't get rows from database: %v", err)
	}

	var result Account
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

func testAccountsEqual(t *testing.T, actual, expected Account) {
	t.Helper()

	if actual.Id != expected.Id {
		t.Errorf("Invalid ID %d, expected %d", actual.Id, expected.Id)
	}

	if actual.Name != expected.Name {
		t.Errorf("Invalid name %q, expected %q", actual.Name, expected.Name)
	}

	if actual.Type != expected.Type {
		t.Errorf("Invalid ID %q, expected %q", actual.Type, expected.Type)
	}

	if len(actual.BankNumbers) != len(expected.BankNumbers) {
		t.Logf("Unequal bank numbers lengths %d and %d", len(actual.BankNumbers), len(expected.BankNumbers))
	}
	if !slices.Equal(actual.BankNumbers, expected.BankNumbers) {
		t.Errorf("Actual bank numbers %v, expected %v", actual.BankNumbers, expected.BankNumbers)
	}

	if len(actual.Notes) != len(expected.Notes) {
		t.Logf("Unequal notes lengths %d and %d", len(actual.Notes), len(expected.Notes))
	}
	if !slices.Equal(actual.Notes, expected.Notes) {
		t.Errorf("Actual notes %v, expected %v", actual.Notes, expected.Notes)
	}
}
