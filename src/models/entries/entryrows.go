package entries

import (
	"context"
	"log/slog"
	"terminaccounting/models/utils"

	"github.com/jmoiron/sqlx"
)

type DecimalValue struct {
	whole      int
	fractional int
}

type EntryRow struct {
	id         int          `db:"id"`
	entry      int          `db:"entry"`
	ledger     int          `db:"ledger"`
	account    int          `db:"account"`
	document   string       `db:"document"`
	value      DecimalValue `db:"value"`
	reconciled bool         `db:"reconciled"`
}

func SetupSchemaEntryRows(ctx context.Context, db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(ctx, db, "entryrows")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `entryrows` table")

	schema := `CREATE TABLE IF NOT EXISTS entryrows(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		entry INTEGER,
		ledger INTEGER,
		account INTEGER,
		document,
		FOREIGN KEY (entry) REFERENCES entries(id),
		FOREIGN KEY (ledger) REFERENCES ledgers(id),
		FOREIGN KEY (account) REFERENCES accounts(id)
	);`

	_, err = db.ExecContext(ctx, schema)
	return err
}
