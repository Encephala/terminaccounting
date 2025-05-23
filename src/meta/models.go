package meta

import (
	"database/sql/driver"
	"encoding/json"
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

type Notes []string

func (n *Notes) Scan(value any) error {
	if value == nil {
		*n = make([]string, 0)
		return nil
	}

	converted, ok := value.(string)
	if !ok {
		return fmt.Errorf("UNMARSHALLING INVALID NOTES: %v", value)
	}

	return json.Unmarshal([]byte(converted), n)
}

func (n Notes) Value() (driver.Value, error) {
	binary, err := json.Marshal(n)
	result := string(binary)

	return result, err
}
