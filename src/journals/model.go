package journals

import (
	"log/slog"
	"terminaccounting/meta"

	"github.com/jmoiron/sqlx"
)

type JournalType string

const (
	Debtor   JournalType = "DEBTOR"
	Creditor JournalType = "CREDITOR"
)

type Journal struct {
	id          int         `db:"id"`
	name        string      `db:"name"`
	journalType JournalType `db:"type"`
	notes       []string    `db:"notes"`
}

func (a *app) SetupSchema(db *sqlx.DB) (int, error) {
	isSetUp, err := meta.DatabaseTableIsSetUp(db, "journals")
	if err != nil {
		return 0, err
	}
	if isSetUp {
		return 0, nil
	}

	slog.Info("Creating `journals` table")

	schema := `CREATE TABLE IF NOT EXISTS journals(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type INTEGER NOT NULL,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return 1, err
}
