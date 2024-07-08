package meta

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type SetupSchemaMsg struct {
	Db *sqlx.DB
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
