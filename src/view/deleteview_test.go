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

func testGenericDeleteView(t *testing.T, v View, expectedTitle string, expectedInputNames []string, notificationMsg meta.NotificationMessageMsg) {
	t.Helper()

	tw := tat.NewTestWrapperSpecific(v,
		notificationMsg,
		meta.SwitchAppViewMsg{ViewType: meta.LISTVIEWTYPE},
	)

	t.Run("Rendering", func(t *testing.T) {
		tw.AssertViewContains(t, expectedTitle)
		for _, name := range expectedInputNames {
			tw.AssertViewContains(t, name)
		}
		tw.AssertViewContains(t, ":w")
	})

	t.Run("Commit", func(t *testing.T) {
		tw.Send(meta.CommitMsg{})

		require.Len(t, tw.LastCmdResults, 2)
		assert.IsType(t, meta.NotificationMessageMsg{}, tw.LastCmdResults[0])

		switchMsg := tw.LastCmdResults[1].(meta.SwitchAppViewMsg)
		assert.Equal(t, meta.LISTVIEWTYPE, switchMsg.ViewType)
	})
}

func TestAccountsDeleteView(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	accountId, err := account.Insert(DB)
	require.NoError(t, err)
	account.Id = accountId

	dv := NewAccountsDeleteView(DB, accountId)

	testGenericDeleteView(t, View(dv),
		fmt.Sprintf("Delete account: %s", account.String()),
		[]string{"Name", "Type", "Bank numbers", "Notes"},
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully deleted Account %q", account.Name)},
	)

	accounts, err := database.SelectAccounts(DB)
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

func TestLedgersDeleteView(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)
	ledger.Id = ledgerId

	dv := NewLedgersDeleteView(DB, ledgerId)

	testGenericDeleteView(t, View(dv),
		fmt.Sprintf("Delete ledger %s", ledger.String()),
		[]string{"Name", "Type", "Notes", "Is accounts ledger"},
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully deleted Ledger %q", ledger.Name)},
	)

	ledgers, err := database.SelectLedgers(DB)
	require.NoError(t, err)
	assert.Empty(t, ledgers)
}

func TestJournalsDeleteView(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	dv := NewJournalsDeleteView(DB, journalId)

	testGenericDeleteView(t, View(dv),
		fmt.Sprintf("Delete journal: %s", journal.String()),
		[]string{"Name", "Type", "Notes"},
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully deleted Journal %q", journal.Name)},
	)

	journals, err := database.SelectJournals(DB)
	require.NoError(t, err)
	assert.Empty(t, journals)
}

func TestEntryDeleteView(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	journalId, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = journalId

	ledger := database.Ledger{Name: "Test Ledger", Type: database.EXPENSELEDGER}
	ledgerId, err := ledger.Insert(DB)
	require.NoError(t, err)
	ledger.Id = ledgerId

	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	entry := database.Entry{Journal: journalId}
	entryRows := []database.EntryRow{
		{Ledger: ledgerId, Date: date, Value: 1000},
		{Ledger: ledgerId, Date: date, Value: -1000},
	}
	entryId, err := entry.Insert(DB, entryRows)
	require.NoError(t, err)
	entry.Id = entryId

	dv := NewEntryDeleteView(DB, entryId)

	testGenericDeleteView(t, View(dv),
		fmt.Sprintf("Delete entry %s", entry.String()),
		[]string{"Journal", "Notes", "# rows", "Entry size"},
		meta.NotificationMessageMsg{Message: fmt.Sprintf("Successfully deleted entry \"%d\"", entryId)},
	)

	entries, err := database.SelectEntries(DB)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
