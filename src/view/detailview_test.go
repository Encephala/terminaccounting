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

func TestJournalsDetailView_DataLoaded(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = jID

	entry1 := database.Entry{Journal: jID, Notes: []string{"First Entry"}}
	_, err = entry1.Insert(DB, []database.EntryRow{})
	require.NoError(t, err)

	entry2 := database.Entry{Journal: jID, Notes: []string{"Second Entry"}}
	_, err = entry2.Insert(DB, []database.EntryRow{})
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	dv := NewJournalsDetailView(DB, journal)
	tw := tat.NewTestWrapperSpecific(View(dv))

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Equal(t, v.modelId, jID)
		assert.Len(t, v.listModel.Items(), 2)
	})
}

func TestJournalsDetailView_Navigation(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = jID

	ledger := database.Ledger{Name: "L", Type: database.EXPENSELEDGER}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "A", Type: database.CREDITOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	rows := []database.EntryRow{
		{
			Date:    database.Date(time.Now()),
			Ledger:  lID,
			Account: &aID,
			Value:   database.CurrencyValue(100),
		},
	}

	// Insert 3 entries
	for i := 0; i < 3; i++ {
		entry := database.Entry{Journal: jID}
		_, err = entry.Insert(DB, rows)
		require.NoError(t, err)
	}
	require.NoError(t, database.UpdateCache(DB))

	dv := NewJournalsDetailView(DB, journal)
	tw := tat.NewTestWrapperSpecific(View(dv))

	// Initial index 0
	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Equal(t, 0, v.listModel.Index())
	})

	// Move down
	tw.Send(meta.NavigateMsg{Direction: meta.DOWN})

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Equal(t, 1, v.listModel.Index())
	})

	// Move down again
	tw.Send(meta.NavigateMsg{Direction: meta.DOWN})

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Equal(t, 2, v.listModel.Index())
	})

	// Move up
	tw.Send(meta.NavigateMsg{Direction: meta.UP})

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Equal(t, 1, v.listModel.Index())
	})
}

func TestJournalsDetailView_Filter(t *testing.T) {
	DB := tat.SetupTestEnv(t)

	journal := database.Journal{Name: "Test Journal", Type: database.GENERALJOURNAL}
	jID, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = jID

	ledger := database.Ledger{Name: "L", Type: database.EXPENSELEDGER, IsAccounts: true}
	lID, err := ledger.Insert(DB)
	require.NoError(t, err)

	account := database.Account{Name: "A", Type: database.CREDITOR}
	aID, err := account.Insert(DB)
	require.NoError(t, err)

	rows := []database.EntryRow{
		{
			Date:    database.Date(time.Now()),
			Ledger:  lID,
			Account: &aID,
			Value:   database.CurrencyValue(100),
		},
	}

	entry1 := database.Entry{Journal: jID, Notes: []string{"Apple"}}
	_, err = entry1.Insert(DB, rows)
	require.NoError(t, err)

	entry2 := database.Entry{Journal: jID, Notes: []string{"Banana"}}
	_, err = entry2.Insert(DB, rows)
	require.NoError(t, err)

	require.NoError(t, database.UpdateCache(DB))

	dv := NewJournalsDetailView(DB, journal)
	tw := tat.NewTestWrapperSpecific(View(dv))

	// Filter "Apple"
	tw.Send(meta.UpdateSearchMsg{Query: "Apple"})

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Len(t, v.listModel.VisibleItems(), 1)
		item := v.listModel.VisibleItems()[0].(database.Entry)
		assert.Contains(t, item.Notes, "Apple")
	})

	// Reset filter
	tw.Send(meta.UpdateSearchMsg{Query: ""})

	tw.Execute(t, func(view View) {
		v := view.(*journalsDetailView)
		assert.Len(t, v.listModel.VisibleItems(), 2)
	})
}
