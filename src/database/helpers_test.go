package database_test

import (
	"terminaccounting/database"
	"terminaccounting/meta"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func insertTestLedger(t *testing.T, DB *sqlx.DB) database.Ledger {
	t.Helper()

	ledger := database.Ledger{
		Name:       "test ledger",
		Type:       database.INCOMELEDGER,
		Notes:      meta.Notes{},
		IsAccounts: false,
	}
	id, err := ledger.Insert(DB)
	require.NoError(t, err)
	ledger.Id = id

	return ledger
}

func insertTestAccount(t *testing.T, DB *sqlx.DB) database.Account {
	t.Helper()

	account := database.Account{
		Name:        "test account",
		Type:        database.DEBTOR,
		BankNumbers: meta.Notes{},
		Notes:       meta.Notes{},
	}
	id, err := account.Insert(DB)
	require.NoError(t, err)
	account.Id = id

	return account
}

func insertTestJournal(t *testing.T, DB *sqlx.DB) database.Journal {
	t.Helper()

	journal := database.Journal{
		Name:  "test journal",
		Type:  database.GENERALJOURNAL,
		Notes: meta.Notes{},
	}
	id, err := journal.Insert(DB)
	require.NoError(t, err)
	journal.Id = id

	return journal
}

func insertTestEntry(t *testing.T, DB *sqlx.DB, journalId int, ledgerId int) database.Entry {
	t.Helper()

	entry := database.Entry{
		Journal: journalId,
		Notes:   meta.Notes{},
	}
	date, err := database.ToDate("2024-01-01")
	require.NoError(t, err)

	rows := []database.EntryRow{
		{
			Date:        date,
			Ledger:      ledgerId,
			Account:     nil,
			Description: "test row",
			Document:    nil,
			Value:       1000,
			Reconciled:  false,
		},
	}
	id, err := entry.Insert(DB, rows)
	require.NoError(t, err)
	entry.Id = id

	return entry
}