package meta

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jmoiron/sqlx"
)

type App interface {
	Name() string

	Render() string

	SetupSchema(db *sqlx.DB) (int, error)

	AccentColour() lipgloss.Color
}
