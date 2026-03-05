package apps_test

import (
	"terminaccounting/apps"
	"terminaccounting/database"
	"terminaccounting/meta"
	tat "terminaccounting/tat"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeLoadListCmd(t *testing.T) {
	testCases := []struct {
		appType     meta.AppType
		modelType   meta.ModelType
		insertItems func(*testing.T, *sqlx.DB) int
		newApp      func(*sqlx.DB) meta.App
	}{
		{
			appType:   meta.LEDGERSAPP,
			modelType: meta.LEDGERMODEL,
			insertItems: func(t *testing.T, DB *sqlx.DB) int {
				t.Helper()
				_, err := (&database.Ledger{Name: "L1", Type: database.EXPENSELEDGER}).Insert(DB)
				require.NoError(t, err)
				_, err = (&database.Ledger{Name: "L2", Type: database.ASSETLEDGER}).Insert(DB)
				require.NoError(t, err)
				return 2
			},
			newApp: apps.NewLedgersApp,
		},
		{
			appType:   meta.ACCOUNTSAPP,
			modelType: meta.ACCOUNTMODEL,
			insertItems: func(t *testing.T, DB *sqlx.DB) int {
				t.Helper()
				_, err := (&database.Account{Name: "A1", Type: database.DEBTOR}).Insert(DB)
				require.NoError(t, err)
				_, err = (&database.Account{Name: "A2", Type: database.CREDITOR}).Insert(DB)
				require.NoError(t, err)
				return 2
			},
			newApp: apps.NewAccountsApp,
		},
		{
			appType:   meta.JOURNALSAPP,
			modelType: meta.JOURNALMODEL,
			insertItems: func(t *testing.T, DB *sqlx.DB) int {
				t.Helper()
				_, err := (&database.Journal{Name: "J1", Type: database.GENERALJOURNAL}).Insert(DB)
				require.NoError(t, err)
				_, err = (&database.Journal{Name: "J2", Type: database.INCOMEJOURNAL}).Insert(DB)
				require.NoError(t, err)
				_, err = (&database.Journal{Name: "J3", Type: database.EXPENSEJOURNAL}).Insert(DB)
				require.NoError(t, err)
				return 3
			},
			newApp: apps.NewJournalsApp,
		},
		{
			appType:   meta.ENTRIESAPP,
			modelType: meta.ENTRYMODEL,
			insertItems: func(t *testing.T, DB *sqlx.DB) int {
				t.Helper()
				journalId, err := (&database.Journal{Name: "J1", Type: database.GENERALJOURNAL}).Insert(DB)
				require.NoError(t, err)
				entry := database.Entry{Journal: journalId}
				_, err = entry.Insert(DB, []database.EntryRow{})
				require.NoError(t, err)
				_, err = entry.Insert(DB, []database.EntryRow{})
				require.NoError(t, err)
				return 2
			},
			newApp: apps.NewEntriesApp,
		},
	}

	for _, tc := range testCases {
		t.Run(string(tc.appType), func(t *testing.T) {
			t.Parallel()

			DB := tat.SetupTestEnv(t)
			expectedCount := tc.insertItems(t, DB)

			app := tc.newApp(DB)
			cmd := app.MakeLoadListCmd()

			result := cmd()
			dataMsg, ok := result.(meta.DataLoadedMsg)
			require.True(t, ok, "expected DataLoadedMsg, got %T", result)

			assert.Equal(t, tc.appType, dataMsg.TargetApp)
			assert.Equal(t, tc.modelType, dataMsg.Model)
			assert.Len(t, dataMsg.Data, expectedCount)
		})
	}
}
