package view

import (
	"testing"

	"terminaccounting/bubbles/itempicker"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/stretchr/testify/assert"
)

func testGenericMutateView_Generic(t *testing.T, v genericMutateView, expectedTitle string, expectedInputNames []string) {
	tw := tat.NewTestWrapperSpecific(View(v))

	// Generic Rendering
	t.Run("Rendering", func(t *testing.T) {
		tw.AssertViewContains(t, expectedTitle)
		for _, name := range expectedInputNames {
			tw.AssertViewContains(t, name)
		}
	})

	// Focus Navigation
	t.Run("Focus Navigation", func(t *testing.T) {
		im := v.getInputManager()
		assert.Equal(t, 0, im.activeInput, "Initial active input should be 0")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 1, im.activeInput, "Active input should be 1 after NEXT")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, 0, im.activeInput, "Active input should be 0 after PREVIOUS")

		// Test looping
		im.activeInput = len(im.inputs) - 1

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 0, im.activeInput, "Active input should loop to 0 after NEXT from last input")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, len(im.inputs)-1, im.activeInput, "Active input should loop to last input after PREVIOUS from 0")
	})

	// Input Delegation
	t.Run("Input Delegation", func(t *testing.T) {
		im := v.getInputManager()
		// Ensure we are at the first input (Name)
		im.activeInput = 0
		im.inputs[0].focus()

		tw.SendText("test")

		val := im.inputs[0].value()
		assert.Equal(t, "test", val)
	})
}

func TestAccountsCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewAccountsCreateView(DB)
	testGenericMutateView_Generic(t, v, "Creating new account", []string{"Name", "Type", "Bank numbers", "Notes"})
}

func TestLedgersCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewLedgersCreateView(DB)
	testGenericMutateView_Generic(t, v, "Creating new ledger", []string{"Name", "Type", "Notes", "Is accounts ledger?"})
}

func TestJournalsCreateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewJournalsCreateView(DB)
	testGenericMutateView_Generic(t, v, "Creating new journal", []string{"Name", "Type", "Notes"})
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
	l1 := database.Ledger{Name: "Ledger 1"}
	a1 := database.Account{Name: "Account 1"}

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
	l1 := database.Ledger{Name: "Ledger 1"}
	a1 := database.Account{Name: "Account 1"}

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
