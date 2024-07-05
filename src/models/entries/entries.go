package entries

import (
	"context"
	"log/slog"
	"terminaccounting/models/utils"

	"github.com/jmoiron/sqlx"
)

type Entry struct {
	id      int      `db:"id"`
	journal int      `db:"journal"`
	notes   []string `db:"notes"`
}

func SetupSchemaEntries(ctx context.Context, db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(ctx, db, "entries")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `entries` table")

	schema := `CREATE TABLE IF NOT EXISTS entries(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		journal INTEGER,
		notes JSONB,
		FOREIGN KEY (journal) REFERENCES journals(id)
	);`

	_, err = db.ExecContext(ctx, schema)
	return err
}
