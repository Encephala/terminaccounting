package view

import (
	"testing"
	"time"

	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/tat"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericDetailView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)

	entry := database.Entry{Journal: jID}
	rows := []database.EntryRow{
		{
			Date:    database.Date(time.Now()),
			Ledger:  lID,
			Account: &aID,
			Value:   database.CurrencyValue(100),
		},
	}
	_, err = entry.Insert(DB, rows)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	dv := NewAccountsDetailView(DB, aID)
	tw := tat.NewTestWrapperSpecific(View(dv))

	tw.Execute(t, func(view View) {
		v := view.(*accountsDetailView)

		assert.Equal(t, v.model.Id, aID)

		require.Len(t, v.viewer.rows, 1)
		assert.Equal(t, v.viewer.rows[0].Value, database.CurrencyValue(100))
	})
}

func TestGenericDetailView_Reconciliation(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)

	entry := database.Entry{Journal: jID}
	rows := []database.EntryRow{
		{
			Date:       database.Date(time.Now()),
			Ledger:     lID,
			Account:    &aID,
			Value:      database.CurrencyValue(100),
			Reconciled: false,
		},
		{
			Date:       database.Date(time.Now()),
			Ledger:     lID,
			Account:    &aID,
			Value:      database.CurrencyValue(-100),
			Reconciled: false,
		},
	}
	_, err = entry.Insert(DB, rows)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	dv := NewAccountsDetailView(DB, aID)
	tw := tat.NewTestWrapperSpecific(View(dv), meta.NotificationMessageMsg{
		Message: "set reconciled status, updated 2 rows",
	})

	// Initially not reconciled
	tw.Execute(t, func(view View) {
		v := view.(*accountsDetailView)

		require.Len(t, v.viewer.rows, 2)

		assert.False(t, v.viewer.rows[0].Reconciled)
		assert.False(t, v.viewer.rows[1].Reconciled)
	})

	// Toggle reconcile
	tw.Send(meta.ReconcileMsg{}, meta.NavigateMsg{Direction: meta.DOWN}, meta.ReconcileMsg{})

	// Should be reconciled in memory
	tw.Execute(t, func(view View) {
		v := view.(*accountsDetailView)

		require.Len(t, v.viewer.rows, 2)

		assert.True(t, v.viewer.rows[0].Reconciled)
		assert.True(t, v.viewer.rows[1].Reconciled)
	})

	// Commit
	tw.Send(meta.CommitMsg{})

	// Check DB
	storedRows, err := database.SelectRowsByAccount(DB, aID)
	require.NoError(t, err)
	require.Len(t, storedRows, 2)
	assert.True(t, storedRows[0].Reconciled)
}

func TestGenericDetailView_ToggleShowReconciled(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER, IsAccounts: true}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)

	entry := database.Entry{Journal: jID}
	rows := []database.EntryRow{
		{
			Date:       database.Date(time.Now()),
			Ledger:     lID,
			Account:    &aID,
			Value:      database.CurrencyValue(100),
			Reconciled: true,
		},
		{
			Date:       database.Date(time.Now()),
			Ledger:     lID,
			Account:    &aID,
			Value:      database.CurrencyValue(200),
			Reconciled: false,
		},
	}
	_, err = entry.Insert(DB, rows)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	dv := NewAccountsDetailView(DB, aID)
	tw := tat.NewTestWrapperSpecific(View(dv))

	// Default: hide reconciled
	tw.Execute(t, func(view View) {
		v := view.(*accountsDetailView)

		require.Len(t, v.viewer.shownRows, 1)

		assert.Equal(t, v.viewer.shownRows[0].Value, database.CurrencyValue(200))
	})

	// Toggle show reconciled
	tw.Send(meta.ToggleShowReconciledMsg{})

	tw.Execute(t, func(view View) {
		assert.Len(t, view.(*accountsDetailView).viewer.shownRows, 2)
	})
}

func TestGenericDetailView_Navigation(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	ledger := database.Ledger{Name: "Test Ledger", Type: database.ASSETLEDGER}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "Test Account", Type: database.DEBTOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)

	entry := database.Entry{Journal: jID}
	rows := []database.EntryRow{
		{
			Date:    database.Date(time.Now()),
			Ledger:  lID,
			Account: &aID,
			Value:   database.CurrencyValue(100),
		},
		{
			Date:    database.Date(time.Now()),
			Ledger:  lID,
			Account: &aID,
			Value:   database.CurrencyValue(200),
		},
	}
	_, err = entry.Insert(DB, rows)
	require.NoError(t, err)
	require.NoError(t, database.UpdateCache(DB))

	dv := NewAccountsDetailView(DB, aID)
	tw := tat.NewTestWrapperSpecific(View(dv))

	// Initial active row is 0
	tw.Execute(t, func(view View) {
		assert.Equal(t, view.(*accountsDetailView).viewer.activeRow, 0)
	})

	// Move down
	tw.Send(meta.NavigateMsg{Direction: meta.DOWN})

	tw.Execute(t, func(view View) {
		assert.Equal(t, view.(*accountsDetailView).viewer.activeRow, 1)
	})

	// Move down again (should stay at 1)
	tw.Send(meta.NavigateMsg{Direction: meta.DOWN})

	tw.Execute(t, func(view View) {
		assert.Equal(t, view.(*accountsDetailView).viewer.activeRow, 2)
	})

	// Move up
	tw.Send(meta.NavigateMsg{Direction: meta.UP})

	tw.Execute(t, func(view View) {
		assert.Equal(t, view.(*accountsDetailView).viewer.activeRow, 1)
	})
}
