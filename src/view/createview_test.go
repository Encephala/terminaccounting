package view

import (
	"errors"
	"testing"

	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountsCreateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewAccountsCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(v), meta.NotificationMessageMsg{
		Message: "Successfully created Account \"Test Account\"",
	}, meta.SwitchAppViewMsg{
		ViewType: meta.UPDATEVIEWTYPE,
		Data:     1,
	})

	// Set inputs
	im := v.getInputManager()
	require.NoError(t, im.inputs[0].setValue("Test Account"))
	require.NoError(t, im.inputs[1].setValue(database.DEBTOR))
	require.NoError(t, im.inputs[2].setValue("IBAN123"))
	require.NoError(t, im.inputs[3].setValue("Some notes"))

	// Commit
	tw.Send(meta.CommitMsg{})

	// Verify DB
	accounts, err := database.SelectAccounts(DB)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "Test Account", accounts[0].Name)
	assert.Equal(t, database.DEBTOR, accounts[0].Type)
	assert.Equal(t, meta.Notes{"IBAN123"}, accounts[0].BankNumbers)
	assert.Equal(t, meta.Notes{"Some notes"}, accounts[0].Notes)

	// Verify messages
	require.Len(t, tw.LastCmdResults, 2)
	assert.IsType(t, meta.NotificationMessageMsg{}, tw.LastCmdResults[0])
	assert.IsType(t, meta.SwitchAppViewMsg{}, tw.LastCmdResults[1])

	switchMsg := tw.LastCmdResults[1].(meta.SwitchAppViewMsg)
	assert.Equal(t, meta.UPDATEVIEWTYPE, switchMsg.ViewType)
	assert.Equal(t, accounts[0].Id, switchMsg.Data)
}

func TestLedgersCreateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewLedgersCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(v), meta.NotificationMessageMsg{
		Message: "Successfully created Ledger \"Test Ledger\"",
	}, meta.SwitchAppViewMsg{
		ViewType: meta.UPDATEVIEWTYPE,
		Data:     1,
	})

	// Set inputs
	im := v.getInputManager()
	require.NoError(t, im.inputs[0].setValue("Test Ledger"))
	require.NoError(t, im.inputs[1].setValue(database.ASSETLEDGER))
	require.NoError(t, im.inputs[2].setValue("Ledger notes"))
	require.NoError(t, im.inputs[3].setValue(true))

	// Commit
	tw.Send(meta.CommitMsg{})

	// Verify DB
	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)
	require.Len(t, ledgers, 1)
	assert.Equal(t, "Test Ledger", ledgers[0].Name)
	assert.Equal(t, database.ASSETLEDGER, ledgers[0].Type)
	assert.True(t, ledgers[0].IsAccounts)

	// Verify messages
	require.Len(t, tw.LastCmdResults, 2)
	assert.IsType(t, meta.NotificationMessageMsg{}, tw.LastCmdResults[0])
	assert.IsType(t, meta.SwitchAppViewMsg{}, tw.LastCmdResults[1])
}

func TestLedgersCreateView_Commit_DuplicateAccountsLedger(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	// Create existing accounts ledger
	existing := database.Ledger{Name: "Existing", Type: database.ASSETLEDGER, IsAccounts: true}
	_, err := existing.Insert(DB)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	v := NewLedgersCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(v),
		errors.New("ledger \"Existing (1)\" already is accounts ledger, can't have multiple"),
	)

	// Set inputs to try creating another accounts ledger
	im := v.getInputManager()
	require.NoError(t, im.inputs[0].setValue("New Ledger"))
	require.NoError(t, im.inputs[1].setValue(database.ASSETLEDGER))
	require.NoError(t, im.inputs[3].setValue(true))

	// Commit
	tw.Send(meta.CommitMsg{})

	// Verify error
	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))

	// Verify DB unchanged count
	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)
	assert.Len(t, ledgers, 1)
}

func TestJournalsCreateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	v := NewJournalsCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(v), meta.NotificationMessageMsg{
		Message: "Successfully created Journal \"Test Journal\"",
	}, meta.SwitchAppViewMsg{
		ViewType: meta.UPDATEVIEWTYPE,
		Data:     1,
	})

	// Set inputs
	im := v.getInputManager()
	require.NoError(t, im.inputs[0].setValue("Test Journal"))
	require.NoError(t, im.inputs[1].setValue(database.GENERALJOURNAL))
	require.NoError(t, im.inputs[2].setValue("Journal notes"))

	// Commit
	tw.Send(meta.CommitMsg{})

	// Verify DB
	journals, err := database.SelectJournals(DB)
	require.NoError(t, err)
	require.Len(t, journals, 1)
	assert.Equal(t, "Test Journal", journals[0].Name)
	assert.Equal(t, database.GENERALJOURNAL, journals[0].Type)

	// Verify messages
	require.Len(t, tw.LastCmdResults, 2)
	assert.IsType(t, meta.NotificationMessageMsg{}, tw.LastCmdResults[0])
	assert.IsType(t, meta.SwitchAppViewMsg{}, tw.LastCmdResults[1])
}

func TestEntryCreateView_Rendering(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	tw.AssertViewContains(t, "Journal")
	tw.AssertViewContains(t, "Notes")
}

func TestEntryCreateView_FocusNavigation(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	assert.Equal(t, ENTRIESJOURNALINPUT, cv.activeInput, "initial active input should be journal")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESNOTESINPUT, cv.activeInput, "after NEXT from journal should be notes")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESROWINPUT, cv.activeInput, "after NEXT from notes should be rows")

	// PREVIOUS from rows at the first cell (0, 0) returns to notes
	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESNOTESINPUT, cv.activeInput, "PREVIOUS from rows first cell should return to notes")

	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESJOURNALINPUT, cv.activeInput, "PREVIOUS from notes should return to journal")
}

func TestEntryCreateView_FocusNavigation_WrapsAround(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	// PREVIOUS from journal wraps to rows, focused at last cell
	tw.Send(meta.SwitchFocusMsg{Direction: meta.PREVIOUS})
	assert.Equal(t, ENTRIESROWINPUT, cv.activeInput, "PREVIOUS from journal should wrap to rows")

	// NEXT from the last cell in rows wraps back to journal
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	assert.Equal(t, ENTRIESJOURNALINPUT, cv.activeInput, "NEXT from last rows cell should wrap to journal")
}

func TestEntryCreateView_Commit(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)
	ledger.Id = ledgerId

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)
	account.Id = accountId

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		meta.NotificationMessageMsg{Message: "Successfully created Entry \"1\""},
		meta.SwitchAppViewMsg{ViewType: meta.UPDATEVIEWTYPE, Data: 1},
	)

	cv.journalInput.SetValue(journal)
	cv.notesInput.SetValue("Test Notes")

	cv.entryRowsManager.rowMutators[0].dateInput.SetValue("2024-01-01")
	cv.entryRowsManager.rowMutators[0].ledgerInput.SetValue(ledger)
	cv.entryRowsManager.rowMutators[0].accountInput.SetValue(&account)
	cv.entryRowsManager.rowMutators[0].debitInput.SetValue("50.00")

	cv.entryRowsManager.rowMutators[1].dateInput.SetValue("2024-01-01")
	cv.entryRowsManager.rowMutators[1].ledgerInput.SetValue(ledger)
	cv.entryRowsManager.rowMutators[1].accountInput.SetValue(&account)
	cv.entryRowsManager.rowMutators[1].creditInput.SetValue("50.00")

	tw.Send(meta.CommitMsg{})

	// Verify DB
	entries, err := database.SelectEntries(DB)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, journalId, entries[0].Journal)
	assert.Equal(t, meta.Notes{"Test Notes"}, entries[0].Notes)

	rows, err := database.SelectRowsByEntry(DB, entries[0].Id)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, database.CurrencyValue(5000), rows[0].Value)
	assert.Equal(t, database.CurrencyValue(-5000), rows[1].Value)
	assert.Equal(t, ledgerId, rows[0].Ledger)

	// Verify messages
	require.Len(t, tw.LastCmdResults, 2)
	assert.IsType(t, meta.NotificationMessageMsg{}, tw.LastCmdResults[0])
	assert.IsType(t, meta.SwitchAppViewMsg{}, tw.LastCmdResults[1])

	switchMsg := tw.LastCmdResults[1].(meta.SwitchAppViewMsg)
	assert.Equal(t, meta.UPDATEVIEWTYPE, switchMsg.ViewType)
	assert.Equal(t, entries[0].Id, switchMsg.Data)
}

func TestEntryCreateView_Commit_NoJournal(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		errors.New("no journal selected (none available)"),
	)

	tw.Send(meta.CommitMsg{})

	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))
}

func TestEntryCreateView_Commit_UnbalancedRows(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		errors.New("entry has nonzero total value 20.00"),
	)

	cv.journalInput.SetValue(journal)
	cv.entryRowsManager.rowMutators[0].debitInput.SetValue("50.00")
	cv.entryRowsManager.rowMutators[1].creditInput.SetValue("30.00")

	tw.Send(meta.CommitMsg{})

	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))
}

func TestEntryCreateView_CreateRow(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		meta.SwitchModeMsg{InputMode: meta.INSERTMODE},
	)

	// Navigate to rows input
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	require.Equal(t, ENTRIESROWINPUT, cv.activeInput)

	initialRowCount := len(cv.entryRowsManager.rowMutators)

	tw.Send(CreateEntryRowMsg{After: true})

	assert.Len(t, cv.entryRowsManager.rowMutators, initialRowCount+1, "row count should increase by 1")
}

func TestEntryCreateView_DeleteRow(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	// Navigate to rows input
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
	require.Equal(t, ENTRIESROWINPUT, cv.activeInput)

	initialRowCount := len(cv.entryRowsManager.rowMutators)
	require.Greater(t, initialRowCount, 1, "initial state must have more than one row to allow deletion")

	tw.Send(DeleteEntryRowMsg{})

	assert.Len(t, cv.entryRowsManager.rowMutators, initialRowCount-1, "row count should decrease by 1")
}

func TestEntryCreateView_CreateRow_WhenNotInRows(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		errors.New("no entry row highlighted while trying to create one"),
	)

	require.Equal(t, ENTRIESJOURNALINPUT, cv.activeInput)

	tw.Send(CreateEntryRowMsg{After: true})

	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))
}

func TestEntryCreateView_DeleteRow_WhenNotInRows(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv),
		errors.New("no entry row highlighted while trying to delete one"),
	)

	require.Equal(t, ENTRIESJOURNALINPUT, cv.activeInput)

	tw.Send(DeleteEntryRowMsg{})

	require.Len(t, tw.LastCmdResults, 1)
	assert.Error(t, tw.LastCmdResults[0].(error))
}

func TestEntryCreateView_SmallWindowSize_DoesNotPanic(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	// A negative viewport height causes a panic in the viewport package, ensure an excessively low height
	// doesn't cause a panic.
	tw.Send(tea.WindowSizeMsg{Width: 80, Height: 1})

	assert.NotPanics(t, func() { tw.Execute(t, func(view View) { view.View() }) })
}

func TestEntryCreateView_InputDelegation(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	cv := NewEntryCreateView(DB)
	tw := tat.NewTestWrapperSpecific(View(cv))

	t.Run("notes input", func(t *testing.T) {
		// Navigate to notes - SwitchFocusMsg calls notesInput.Focus()
		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		require.Equal(t, ENTRIESNOTESINPUT, cv.activeInput)

		tw.SendText("some notes")

		assert.Equal(t, "some notes", cv.notesInput.Value())
	})

	t.Run("rows description input", func(t *testing.T) {
		// Navigate to rows - rows start focused at col 0 (date)
		tw.Send(meta.SwitchFocusMsg{Direction: meta.NEXT})
		require.Equal(t, ENTRIESROWINPUT, cv.activeInput)

		// Move right three times to reach col 3 (description)
		tw.Send(
			meta.NavigateMsg{Direction: meta.RIGHT},
			meta.NavigateMsg{Direction: meta.RIGHT},
			meta.NavigateMsg{Direction: meta.RIGHT},
		)
		_, activeCol := cv.entryRowsManager.getActiveCoords()
		require.Equal(t, 3, activeCol)

		tw.SendText("row description")

		assert.Equal(t, "row description", cv.entryRowsManager.rowMutators[0].descriptionInput.Value())
	})
}
