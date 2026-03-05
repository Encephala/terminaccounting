package main

import (
	"fmt"
	"testing"

	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func insertListItems(t *testing.T, DB *sqlx.DB, appType meta.AppType) []string {
	t.Helper()

	names := []string{"Alpha", "Beta", "Gamma"}

	switch appType {
	case meta.LEDGERSAPP:
		for _, name := range names {
			_, err := (&database.Ledger{Name: name, Type: database.ASSETLEDGER}).Insert(DB)
			require.NoError(t, err)
		}

	case meta.ACCOUNTSAPP:
		for _, name := range names {
			_, err := (&database.Account{Name: name, Type: database.DEBTOR}).Insert(DB)
			require.NoError(t, err)
		}

	case meta.JOURNALSAPP:
		for _, name := range names {
			_, err := (&database.Journal{Name: name, Type: database.GENERALJOURNAL}).Insert(DB)
			require.NoError(t, err)
		}

	case meta.ENTRIESAPP:
		jID, err := (&database.Journal{Name: "Journal", Type: database.GENERALJOURNAL}).Insert(DB)
		require.NoError(t, err)

		for _, note := range names {
			entry := database.Entry{Journal: jID, Notes: meta.Notes{note}}
			_, err = entry.Insert(DB, []database.EntryRow{})
			require.NoError(t, err)
		}

	default:
		panic(fmt.Sprintf("unexpected meta.AppType: %#v", appType))
	}

	return names
}

func TestListViewGeneric(t *testing.T) {
	allApps := []meta.AppType{
		meta.ENTRIESAPP,
		meta.LEDGERSAPP,
		meta.ACCOUNTSAPP,
		meta.JOURNALSAPP,
	}

	for _, appType := range allApps {
		t.Run(string(appType), func(t *testing.T) {
			t.Parallel()

			testListView_ItemsDisplayed(t, appType)
			testListView_NavigateDown(t, appType)
			testListView_NavigateUp(t, appType)
			testListView_GoToCreate(t, appType)
			testListView_GoToDetail(t, appType)
			testListView_GoToDetail_NoItems(t, appType)
		})
	}
}

func testListView_ItemsDisplayed(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	itemNames := insertListItems(t, DB, appType)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	for _, name := range itemNames {
		tw.AssertViewContains(t, name)
	}

	tw.Execute(t, func(ta *terminaccounting) {
		assert.Equal(t, meta.LISTVIEWTYPE, ta.appManager.currentViewType())
	})
}

func testListView_NavigateDown(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	insertListItems(t, DB, appType)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	tw.SendText("j")

	assert.Equal(t, meta.NavigateMsg{Direction: meta.DOWN}, tw.LastCmdResults[0])
}

func testListView_NavigateUp(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	insertListItems(t, DB, appType)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	tw.SendText("j") // move down first so there's room to move up
	tw.SendText("k")

	assert.Equal(t, meta.NavigateMsg{Direction: meta.UP}, tw.LastCmdResults[0])
}

func testListView_GoToCreate(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	tw.SendText("gc")

	require.IsType(t, meta.SwitchAppViewMsg{}, tw.LastCmdResults[0])
	switchMsg := tw.LastCmdResults[0].(meta.SwitchAppViewMsg)
	assert.Equal(t, meta.CREATEVIEWTYPE, switchMsg.ViewType)
}

func testListView_GoToDetail(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	insertListItems(t, DB, appType)

	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	tw.SendText("gd")

	// LastCmdResults: [tea.Cmd (the motion's stored cmd), SwitchAppViewMsg, ...DataLoadedMsgs from detail view init]
	require.GreaterOrEqual(t, len(tw.LastCmdResults), 2)
	switchMsg, ok := tw.LastCmdResults[1].(meta.SwitchAppViewMsg)
	require.True(t, ok, "expected SwitchAppViewMsg, got %T", tw.LastCmdResults[1])
	assert.Equal(t, meta.DETAILVIEWTYPE, switchMsg.ViewType)
	assert.NotNil(t, switchMsg.Data)
}

func testListView_GoToDetail_NoItems(t *testing.T, appType meta.AppType) {
	t.Helper()

	DB := tat.SetupTestEnv(t)
	tw := tat.NewTestWrapperGeneric(newTerminaccounting(DB))
	tw.GoToTab(appType)

	tw.SendText("gd")

	// LastCmdResults: [tea.Cmd, error]
	require.Len(t, tw.LastCmdResults, 2)
	assert.Error(t, tw.LastCmdResults[1].(error))
}
