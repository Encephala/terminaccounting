package journals

import (
	"context"
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

func SetupSchema(ctx context.Context, db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(ctx, db, "journals")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `journals` table")

	schema := `CREATE TABLE IF NOT EXISTS journals(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(69),
		type INTEGER,
		notes JSONB
	);`

	_, err = db.ExecContext(ctx, schema)
	return err
}
