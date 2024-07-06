package journals

import (
	"log/slog"
	"terminaccounting/models/utils"

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

func SetupSchema(db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(db, "journals")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `journals` table")

	schema := `CREATE TABLE IF NOT EXISTS journals(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		type INTEGER,
		notes TEXT
	) STRICT;`

	_, err = db.Exec(schema)
	return err
}
