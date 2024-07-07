package meta

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// Does one-time database schema setup
func SetupSchema(db *sqlx.DB, apps []App) error {
	totalChangedCount := 0
	for _, app := range apps {
		changedCount, err := app.SetupSchema(db)
		if err != nil {
			return err
		}
		totalChangedCount += changedCount
	}

	if totalChangedCount > 0 {
		slog.Info(fmt.Sprintf("Finished setting up %d database tables", totalChangedCount))
	}

	return nil
}

func DatabaseTableIsSetUp(db *sqlx.DB, name string) (bool, error) {
	// Kinda hacky, whatever
	result, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name=$1", name)
	if err != nil {
		return false, fmt.Errorf("FAILED TO CHECK IF DATABASE IS NEW: %v", err)
	}
	defer result.Close()
	result.Next()

	name = ""
	err = result.Scan(&name)
	if err == nil {
		return true, nil
	} else {
		return false, nil
	}
}
