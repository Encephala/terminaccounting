package view

import (
	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/tat"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

// Mock helpers to create database structs since we can't rely on the DB being present
func makeTestLedger(id int, name string) database.Ledger {
	return database.Ledger{Id: id, Name: name}
}

func makeTestAccount(id int, name string) database.Account {
	return database.Account{Id: id, Name: name}
}

// Helper to create a rowCreator with populated itempickers
func newTestRowCreator(ledgers []database.Ledger, accounts []database.Account) *rowCreator {
	lItems := make([]itempicker.Item, len(ledgers))
	for i, v := range ledgers {
		lItems[i] = v
	}

	aItems := make([]itempicker.Item, len(accounts))
	for i := range accounts {
		acc := accounts[i]
		aItems[i] = &acc
	}

	rc := &rowCreator{
		dateInput:        textinput.New(),
		ledgerInput:      itempicker.New(lItems),
		accountInput:     itempicker.New(aItems),
		descriptionInput: textinput.New(),
		debitInput:       textinput.New(),
		creditInput:      textinput.New(),
	}

	return rc
}

func TestRowsMutateManager_CompileRows_Success(t *testing.T) {
	l1 := makeTestLedger(1, "Ledger 1")
	a1 := makeTestAccount(1, "Account 1")

	rc1 := newTestRowCreator([]database.Ledger{l1}, []database.Account{a1})
	rc1.dateInput.SetValue("2024-01-01")
	rc1.ledgerInput.SetValue(l1)
	rc1.accountInput.SetValue(&a1)
	rc1.descriptionInput.SetValue("Desc 1")
	rc1.debitInput.SetValue("10.00")

	rc2 := newTestRowCreator([]database.Ledger{l1}, []database.Account{a1})
	rc2.dateInput.SetValue("2024-01-01")
	rc2.ledgerInput.SetValue(l1)
	rc2.accountInput.SetValue(&a1)
	rc2.descriptionInput.SetValue("Desc 2")
	rc2.creditInput.SetValue("10.00")

	manager := &rowsMutateManager{
		rows: []*rowCreator{rc1, rc2},
	}

	rows, err := manager.compileRows()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(rows))
	}

	if rows[0].Value != 1000 {
		t.Errorf("Row 1 value mismatch: got %d, want 1000", rows[0].Value)
	}
	if rows[1].Value != -1000 {
		t.Errorf("Row 2 value mismatch: got %d, want -1000", rows[1].Value)
	}
	if rows[0].Description != "Desc 1" {
		t.Errorf("Row 1 description mismatch: got %q, want %q", rows[0].Description, "Desc 1")
	}
}

func TestRowsMutateManager_CompileRows_Unbalanced(t *testing.T) {
	l1 := makeTestLedger(1, "Ledger 1")
	a1 := makeTestAccount(1, "Account 1")

	rc1 := newTestRowCreator([]database.Ledger{l1}, []database.Account{a1})
	rc1.dateInput.SetValue("2024-01-01")
	rc1.ledgerInput.SetValue(l1)
	rc1.accountInput.SetValue(&a1)
	rc1.debitInput.SetValue("10.00")

	manager := &rowsMutateManager{
		rows: []*rowCreator{rc1},
	}

	_, err := manager.compileRows()
	if err == nil {
		t.Error("Expected error for unbalanced rows, got nil")
	}
}

func TestRowsMutateManager_CalculateCurrentTotal(t *testing.T) {
	rc1 := newTestRowCreator(nil, nil)
	rc1.debitInput.SetValue("10.50")

	rc2 := newTestRowCreator(nil, nil)
	rc2.creditInput.SetValue("5.25")

	manager := &rowsMutateManager{
		rows: []*rowCreator{rc1, rc2},
	}

	total, err := manager.calculateCurrentTotal()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// 10.50 - 5.25 = 5.25 -> 525
	expected := database.CurrencyValue(525)
	if total != expected {
		t.Errorf("Expected total %d, got %d", expected, total)
	}
}

func TestRowsMutateManager_AddRow(t *testing.T) {
	tat.SetupTestEnv(t)

	rc1 := newTestRowCreator(nil, nil)
	rc1.dateInput.SetValue("2024-05-20")

	manager := &rowsMutateManager{
		rows:        []*rowCreator{rc1},
		activeInput: 0, // Focus on first row
	}

	// Add row after
	manager.addRow(true)

	if len(manager.rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(manager.rows))
	}

	// Check if date was prefilled
	if manager.rows[1].dateInput.Value() != "2024-05-20" {
		t.Errorf("Expected new row to have prefilled date '2024-05-20', got '%s'", manager.rows[1].dateInput.Value())
	}
}

func TestRowsMutateManager_DeleteRow(t *testing.T) {
	rc1 := newTestRowCreator(nil, nil)
	rc2 := newTestRowCreator(nil, nil)
	rc3 := newTestRowCreator(nil, nil)

	manager := &rowsMutateManager{
		rows:        []*rowCreator{rc1, rc2, rc3},
		activeInput: 1 * 6, // Focus on second row (index 1)
	}

	manager.deleteRow()

	if len(manager.rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(manager.rows))
	}

	// We deleted the middle one, so we should have rc1 and rc3
	if manager.rows[0] != rc1 {
		t.Error("First row should still be rc1")
	}
	if manager.rows[1] != rc3 {
		t.Error("Second row should be rc3")
	}
}
