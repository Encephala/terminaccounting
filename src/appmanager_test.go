package main

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppViewRouting(t *testing.T) {
	appTypes := []meta.AppType{
		meta.LEDGERSAPP,
		meta.ACCOUNTSAPP,
		meta.JOURNALSAPP,
		meta.ENTRIESAPP,
	}

	for _, appType := range appTypes {
		t.Run(string(appType), func(t *testing.T) {
			testAppViewRouting(t, appType)
		})
	}
}

func testAppViewRouting(t *testing.T, appType meta.AppType) {
	// Insert items before creating wrapper so the cache is populated on Init.
	DB := tat.SetupTestEnv(t)
	itemId := insertItemForApp(t, DB, appType)
	detailData := detailDataForApp(appType, itemId)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	t.Run("create view", func(t *testing.T) {
		tw.SwitchView(meta.CREATEVIEWTYPE)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.CREATEVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("detail view", func(t *testing.T) {
		tw.SwitchView(meta.DETAILVIEWTYPE, detailData)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.DETAILVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("update view", func(t *testing.T) {
		tw.SwitchView(meta.UPDATEVIEWTYPE, itemId)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.UPDATEVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("list view", func(t *testing.T) {
		tw.SwitchView(meta.LISTVIEWTYPE)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.LISTVIEWTYPE, ta.appManager.currentViewType())
		})
	})

	t.Run("delete view", func(t *testing.T) {
		tw.SwitchView(meta.DELETEVIEWTYPE, itemId)

		tw.Execute(t, func(ta *terminaccounting) {
			assert.Equal(t, meta.DELETEVIEWTYPE, ta.appManager.currentViewType())
		})
	})
}

func TestAppManager_ReloadViewMsg(t *testing.T) {
	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))

	tw.Send(meta.ReloadViewMsg{})

	assert.Contains(t, tw.LastCmdResults, meta.NotificationMessageMsg{Message: "Refreshed view"})
}

func insertItemForApp(t *testing.T, DB *sqlx.DB, appType meta.AppType) int {
	t.Helper()

	switch appType {
	case meta.LEDGERSAPP:
		ledgerId, err := (&database.Ledger{Name: "Test", Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
		return ledgerId

	case meta.ACCOUNTSAPP:
		accountId, err := (&database.Account{Name: "Test", Type: database.DEBTOR}).Insert(DB)
		require.NoError(t, err)
		return accountId

	case meta.JOURNALSAPP:
		journalId, err := (&database.Journal{Name: "Test", Type: database.GENERALJOURNAL}).Insert(DB)
		require.NoError(t, err)
		return journalId

	case meta.ENTRIESAPP:
		journalId, err := (&database.Journal{Name: "J", Type: database.GENERALJOURNAL}).Insert(DB)
		require.NoError(t, err)
		ledgerId, err := (&database.Ledger{Name: "L", Type: database.EXPENSELEDGER}).Insert(DB)
		require.NoError(t, err)
		require.NoError(t, database.UpdateCache(DB))
		entry := database.Entry{Journal: journalId}
		entryId, err := entry.Insert(DB, []database.EntryRow{
			{Ledger: ledgerId, Value: 100, Description: "row"},
		})
		require.NoError(t, err)
		return entryId

	default:
		t.Fatalf("unexpected meta.AppType: %#v", appType)
		return 0
	}
}

// Each app's detail view only reads the ID from the struct, so only that field needs to be set.
func detailDataForApp(appType meta.AppType, id int) any {
	switch appType {
	case meta.LEDGERSAPP:
		return database.Ledger{Id: id}
	case meta.ACCOUNTSAPP:
		return database.Account{Id: id}
	case meta.JOURNALSAPP:
		return database.Journal{Id: id}
	case meta.ENTRIESAPP:
		return database.Entry{Id: id}
	default:
		panic("unexpected meta.AppType")
	}
}
