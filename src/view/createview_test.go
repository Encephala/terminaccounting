package view

import (
	"errors"
	"testing"

	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"

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
