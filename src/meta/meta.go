package meta

import (
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jmoiron/sqlx"
)

type App interface {
	tea.Model

	Name() string

	SetupSchema(db *sqlx.DB) (int, error)

	AccentColour() lipgloss.Color
	BackgroundColour() lipgloss.Color
	HoverColour() lipgloss.Color
}
