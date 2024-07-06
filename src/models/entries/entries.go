package entries

import (
	"log/slog"
	"terminaccounting/models/utils"

	"github.com/jmoiron/sqlx"
)

type Entry struct {
	id      int      `db:"id"`
	journal int      `db:"journal"`
	notes   []string `db:"notes"`
}

func SetupSchemaEntries(db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(db, "entries")
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
		notes TEXT,
		FOREIGN KEY (journal) REFERENCES journals(id)
	) STRICT;`

	_, err = db.Exec(schema)
	return err
}
