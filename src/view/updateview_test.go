package view

import (
	"fmt"
	"testing"

	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Like testGenericMutateView_Generic but clears input[0] before the delegation test,
// since update views pre-populate inputs from loaded data.
func testGenericUpdateView_Generic(t *testing.T, v genericMutateView, expectedTitle string, expectedInputNames []string) {
	t.Helper()

	tw := tat.NewTestWrapperSpecific(View(v))

	t.Run("Rendering", func(t *testing.T) {
		tw.AssertViewContains(t, expectedTitle)
		for _, name := range expectedInputNames {
			tw.AssertViewContains(t, name)
		}
	})

	t.Run("Focus Navigation", func(t *testing.T) {
		im := v.getInputManager()
		assert.Equal(t, 0, im.activeInput, "Initial active input should be 0")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 1, im.activeInput, "Active input should be 1 after NEXT")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, 0, im.activeInput, "Active input should be 0 after PREVIOUS")

		im.activeInput = len(im.inputs) - 1

		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		assert.Equal(t, 0, im.activeInput, "Active input should loop to 0 after NEXT from last input")

		tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
		assert.Equal(t, len(im.inputs)-1, im.activeInput, "Active input should loop to last input after PREVIOUS from 0")
	})

	t.Run("Input Delegation", func(t *testing.T) {
		im := v.getInputManager()
		im.activeInput = 0
		im.inputs[0].focus()

		require.NoError(t, im.inputs[0].setValue(""))
		tw.SendText("test")

		assert.Equal(t, "test", im.inputs[0].value())
	})
}

func TestAccountsUpdateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	uv := NewAccountsUpdateView(DB, accountId)
	testGenericUpdateView_Generic(t, uv, "Creating new account", []string{"Name", "Type", "Bank numbers", "Notes"})
}

func TestAccountsUpdateView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{
		Name:        "Test Account",
		Type:        database.DEBTOR,
		BankNumbers: meta.Notes{"IBAN123"},
		Notes:       meta.Notes{"Some notes"},
	}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)
	account.Id = accountId

	uv := NewAccountsUpdateView(DB, accountId)
	tat.NewTestWrapperSpecific(View(uv))

	assert.Equal(t, account.Name, uv.startingValue.Name)
	assert.Equal(t, account.Type, uv.startingValue.Type)
	assert.Equal(t, account.BankNumbers, uv.startingValue.BankNumbers)
	assert.Equal(t, account.Notes, uv.startingValue.Notes)

	assert.Equal(t, "Test Account", uv.inputManager.inputs[ACCOUNTSNAMEINPUT].value())
	assert.Equal(t, database.DEBTOR, uv.inputManager.inputs[ACCOUNTSTYPEINPUT].value())
	assert.Equal(t, "IBAN123", uv.inputManager.inputs[ACCOUNTSBANKNUMBERSINPUT].value())
	assert.Equal(t, "Some notes", uv.inputManager.inputs[ACCOUNTSNOTESINPUT].value())
}

func TestAccountsUpdateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	uv := NewAccountsUpdateView(DB, accountId)
	tw := tat.NewTestWrapperSpecific(View(uv),
		meta.NotificationMessageMsg{Message: `Successfully updated Account "Updated Account"`},
	)

	require.NoError(t, uv.inputManager.inputs[ACCOUNTSNAMEINPUT].setValue("Updated Account"))
	require.NoError(t, uv.inputManager.inputs[ACCOUNTSTYPEINPUT].setValue(database.CREDITOR))

	tw.Send(meta.CommitMsg{})

	accounts, err := database.SelectAccounts(DB)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "Updated Account", accounts[0].Name)
	assert.Equal(t, database.CREDITOR, accounts[0].Type)

	require.Len(t, tw.LastCmdResults, 1)
	assert.Equal(t, meta.NotificationMessageMsg{Message: `Successfully updated Account "Updated Account"`}, tw.LastCmdResults[0])
}

func TestAccountsUpdateView_ResetInputField(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	uv := NewAccountsUpdateView(DB, accountId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	require.NoError(t, uv.inputManager.inputs[ACCOUNTSNAMEINPUT].setValue("Modified Name"))
	assert.Equal(t, "Modified Name", uv.inputManager.inputs[ACCOUNTSNAMEINPUT].value())

	uv.inputManager.activeInput = ACCOUNTSNAMEINPUT
	tw.Send(meta.ResetInputFieldMsg{})

	assert.Equal(t, "Test Account", uv.inputManager.inputs[ACCOUNTSNAMEINPUT].value())
}

func TestLedgersUpdateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	uv := NewLedgersUpdateView(DB, ledgerId)
	testGenericUpdateView_Generic(t, uv, "Update Ledger: Test Ledger", []string{"Name", "Type", "Notes", "Is accounts ledger?"})
}

func TestLedgersUpdateView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{
		Name:       "Test Ledger",
		Type:       database.ASSETLEDGER,
		Notes:      meta.Notes{"Some notes"},
		IsAccounts: false,
	}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)
	ledger.Id = ledgerId

	uv := NewLedgersUpdateView(DB, ledgerId)
	tat.NewTestWrapperSpecific(View(uv))

	assert.Equal(t, ledger.Name, uv.startingValue.Name)
	assert.Equal(t, ledger.Type, uv.startingValue.Type)
	assert.Equal(t, ledger.IsAccounts, uv.startingValue.IsAccounts)

	assert.Equal(t, "Test Ledger", uv.inputManager.inputs[0].value())
	assert.Equal(t, database.ASSETLEDGER, uv.inputManager.inputs[1].value())
	assert.Equal(t, false, uv.inputManager.inputs[3].value())
}

func TestLedgersUpdateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	uv := NewLedgersUpdateView(DB, ledgerId)
	tw := tat.NewTestWrapperSpecific(View(uv),
		meta.NotificationMessageMsg{Message: `Successfully updated Ledger "Updated Ledger"`},
	)

	require.NoError(t, uv.inputManager.inputs[0].setValue("Updated Ledger"))
	require.NoError(t, uv.inputManager.inputs[1].setValue(database.EXPENSELEDGER))

	tw.Send(meta.CommitMsg{})

	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)
	require.Len(t, ledgers, 1)
	assert.Equal(t, "Updated Ledger", ledgers[0].Name)
	assert.Equal(t, database.EXPENSELEDGER, ledgers[0].Type)

	require.Len(t, tw.LastCmdResults, 1)
	assert.Equal(t, meta.NotificationMessageMsg{Message: `Successfully updated Ledger "Updated Ledger"`}, tw.LastCmdResults[0])
}

func TestLedgersUpdateView_Commit_DuplicateAccountsLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	existingAccountsLedger := database.Ledger{Name: "Accounts Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	existingId, err := existingAccountsLedger.Insert(DB)
	require.NoError(t, err)
	existingAccountsLedger.Id = existingId

	otherLedger := database.Ledger{Name: "Other Ledger", Type: database.ASSETLEDGER, IsAccounts: false}
	otherId, err := otherLedger.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	uv := NewLedgersUpdateView(DB, otherId)
	tw := tat.NewTestWrapperSpecific(View(uv),
		fmt.Errorf("ledger %q already is accounts ledger, can't have multiple", &existingAccountsLedger),
	)

	require.NoError(t, uv.inputManager.inputs[3].setValue(true))

	tw.Send(meta.CommitMsg{})

	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))

	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)
	assert.False(t, ledgers[1].IsAccounts, "other ledger should still not be accounts ledger")
}

func TestLedgersUpdateView_ResetInputField(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	uv := NewLedgersUpdateView(DB, ledgerId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	require.NoError(t, uv.inputManager.inputs[0].setValue("Modified Ledger"))
	assert.Equal(t, "Modified Ledger", uv.inputManager.inputs[0].value())

	uv.inputManager.activeInput = 0
	tw.Send(meta.ResetInputFieldMsg{})

	assert.Equal(t, "Test Ledger", uv.inputManager.inputs[0].value())
}

func TestJournalsUpdateView_Generic(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	uv := NewJournalsUpdateView(DB, journalId)
	testGenericUpdateView_Generic(t, uv, "Creating new journal", []string{"Name", "Type", "Notes"})
}

func TestJournalsUpdateView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{
		Name:  "Test Journal",
		Type:  database.GENERALJOURNAL,
		Notes: meta.Notes{"Some notes"},
	}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	uv := NewJournalsUpdateView(DB, journalId)
	tat.NewTestWrapperSpecific(View(uv))

	assert.Equal(t, journal.Name, uv.startingValue.Name)
	assert.Equal(t, journal.Type, uv.startingValue.Type)
	assert.Equal(t, journal.Notes, uv.startingValue.Notes)

	assert.Equal(t, "Test Journal", uv.inputManager.inputs[0].value())
	assert.Equal(t, database.GENERALJOURNAL, uv.inputManager.inputs[1].value())
	assert.Equal(t, "Some notes", uv.inputManager.inputs[2].value())
}

func TestJournalsUpdateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	uv := NewJournalsUpdateView(DB, journalId)
	tw := tat.NewTestWrapperSpecific(View(uv),
		meta.NotificationMessageMsg{Message: `Successfully updated Journal "Updated Journal"`},
	)

	require.NoError(t, uv.inputManager.inputs[0].setValue("Updated Journal"))
	require.NoError(t, uv.inputManager.inputs[1].setValue(database.INCOMEJOURNAL))

	tw.Send(meta.CommitMsg{})

	journals, err := database.SelectJournals(DB)
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Equal(t, "Updated Journal", journals[0].Name)
	assert.Equal(t, database.INCOMEJOURNAL, journals[0].Type)

	require.Len(t, tw.LastCmdResults, 1)
	assert.Equal(t, meta.NotificationMessageMsg{Message: `Successfully updated Journal "Updated Journal"`}, tw.LastCmdResults[0])
}

func TestJournalsUpdateView_ResetInputField(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	uv := NewJournalsUpdateView(DB, journalId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	require.NoError(t, uv.inputManager.inputs[0].setValue("Modified Journal"))
	assert.Equal(t, "Modified Journal", uv.inputManager.inputs[0].value())

	uv.inputManager.activeInput = 0
	tw.Send(meta.ResetInputFieldMsg{})

	assert.Equal(t, "Test Journal", uv.inputManager.inputs[0].value())
}

func setupEntryUpdateViewTest(t *testing.T) (DB interface{ Close() error }, journalId, ledgerId, accountId, entryId int) {
	t.Helper()

	dbConn := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	lId, err := ledger.Insert(dbConn)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	aId, err := account.Insert(dbConn)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jId, err := journal.Insert(dbConn)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(dbConn))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: jId, Notes: meta.Notes{"Original notes"}}
	entryRows := []database.EntryRow{
		{Ledger: lId, Account: &aId, Date: date, Value: 5000},
		{Ledger: lId, Account: &aId, Date: date, Value: -5000},
	}
	eId, err := entry.Insert(dbConn, entryRows)
	require.NoError(t, err)

	return dbConn, jId, lId, aId, eId
}

func TestEntryUpdateView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId, Notes: meta.Notes{"Original notes"}}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Account: &accountId, Date: date, Value: 5000},
		{Ledger: ledgerId, Account: &accountId, Date: date, Value: -5000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tat.NewTestWrapperSpecific(View(uv))

	assert.Equal(t, entryId, uv.startingEntry.Id)
	assert.Equal(t, "Original notes", uv.notesInput.Value())
	assert.NotNil(t, uv.journalInput.Value(), "journal input should be populated")
	assert.Equal(t, journalId, uv.journalInput.Value().(database.Journal).Id)
	assert.Len(t, uv.entryRowsManager.rows, 2)
	assert.Len(t, uv.startingEntryRows, 2)
}

func TestEntryUpdateView_FocusNavigation(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Date: date, Value: 1000},
		{Ledger: ledgerId, Date: date, Value: -1000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	assert.Equal(t, ENTRIESJOURNALINPUT, uv.activeInput, "initial active input should be journal")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESNOTESINPUT, uv.activeInput, "after NEXT from journal should be notes")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESROWINPUT, uv.activeInput, "after NEXT from notes should be rows")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESNOTESINPUT, uv.activeInput, "PREVIOUS from rows first cell should return to notes")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESJOURNALINPUT, uv.activeInput, "PREVIOUS from notes should return to journal")
}

func TestEntryUpdateView_FocusNavigation_WrapsAround(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Date: date, Value: 1000},
		{Ledger: ledgerId, Date: date, Value: -1000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESROWINPUT, uv.activeInput, "PREVIOUS from journal should wrap to rows")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESJOURNALINPUT, uv.activeInput, "NEXT from last rows cell should wrap to journal")
}

func TestEntryUpdateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId, Notes: meta.Notes{"Original notes"}}
	originalRows := []database.EntryRow{
		{Ledger: ledgerId, Account: &accountId, Date: date, Value: 5000},
		{Ledger: ledgerId, Account: &accountId, Date: date, Value: -5000},
	}
	entryId, err := entry.Insert(DB, originalRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tw := tat.NewTestWrapperSpecific(View(uv),
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully updated Entry \"%d\"", entryId)},
	)

	uv.notesInput.SetValue("Updated notes")

	tw.Send(meta.CommitMsg{})

	entries, err := database.SelectEntries(DB)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, meta.Notes{"Updated notes"}, entries[0].Notes)

	require.Len(t, tw.LastCmdResults, 1)
	assert.Equal(t,
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully updated Entry \"%d\"", entryId)},
		tw.LastCmdResults[0],
	)
}

func TestEntryUpdateView_ResetInputField_Notes(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId, Notes: meta.Notes{"Original notes"}}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Date: date, Value: 1000},
		{Ledger: ledgerId, Date: date, Value: -1000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tw := tat.NewTestWrapperSpecific(View(uv))

	uv.notesInput.SetValue("Modified notes")
	assert.Equal(t, "Modified notes", uv.notesInput.Value())

	uv.activeInput = ENTRIESNOTESINPUT
	tw.Send(meta.ResetInputFieldMsg{})

	assert.Equal(t, "Original notes", uv.notesInput.Value())
}

func TestEntryUpdateView_ResetInputField_Rows(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Date: date, Value: 1000},
		{Ledger: ledgerId, Date: date, Value: -1000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)

	uv := NewEntryUpdateView(DB, entryId)
	tw := tat.NewTestWrapperSpecific(View(uv), meta.SwitchModeMsg{InputMode: meta.INSERTMODE})

	// Navigate to rows, then add an extra row
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	require.Equal(t, ENTRIESROWINPUT, uv.activeInput)

	tw.Send(CreateEntryRowMsg{After: true})
	require.Len(t, uv.entryRowsManager.rows, 3)

	uv.activeInput = ENTRIESROWINPUT
	tw.Send(meta.ResetInputFieldMsg{})

	assert.Len(t, uv.entryRowsManager.rows, 2, "rows should be reset to original count")
}
