package database

import (
	"fmt"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

var DB *sqlx.DB

func DatabaseTableIsSetUp(name string) (bool, error) {
	// Kinda hacky, whatever
	result, err := DB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name=$1", name)
	if err != nil {
		return false, fmt.Errorf("FAILED TO CHECK IF DATABASE IS NEW: %v", err)
	}
	defer result.Close()
	nextRowAvailable := result.Next()

	if nextRowAvailable {
		return true, nil
	}

	return false, nil
}

func InitCaches() tea.Cmd {
	_, err := SelectJournals()
	if err != nil {
		return meta.MessageCmd(err)
	}

	_, err = SelectAccounts()
	if err != nil {
		return meta.MessageCmd(err)
	}

	_, err = SelectJournals()
	if err != nil {
		return meta.MessageCmd(err)
	}

	return nil
}
