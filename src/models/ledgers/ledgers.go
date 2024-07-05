package ledgers

import (
	"context"
	"log/slog"
	"terminaccounting/models/utils"

	"github.com/jmoiron/sqlx"
)

type LedgerType string

const (
	Income    LedgerType = "INCOME"
	Expense   LedgerType = "EXPENSE"
	Asset     LedgerType = "ASSET"
	Liability LedgerType = "LIABILITY"
	Equity    LedgerType = "EQUITY"
)

type Ledger struct {
	id   int    `db:"id"`
	name string `db:"name"`

	// TODO: Can this work with sqlx? I think not, because some type conversion is needed?
	// But then the whole marshalling into struct isn't going to work and I'd have to do either two queries,
	// or just ditch the whole idea altogether and completely lose the convenience.
	// Ah well, we'll see
	ledgerType LedgerType `db:"type"`
	notes      []string   `db:"notes"`
}

func SetupSchema(ctx context.Context, db *sqlx.DB) error {
	isSetUp, err := utils.TableIsSetUp(ctx, db, "ledgers")
	if err != nil {
		return err
	}
	if isSetUp {
		return nil
	}

	slog.Info("Creating `ledgers` table")

	schema := `CREATE TABLE IF NOT EXISTS ledgers(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(69),
		type INTEGER,
		notes JSONB
	);`

	_, err = db.ExecContext(ctx, schema)
	return err
}
