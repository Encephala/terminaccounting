package meta

import (
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

type App interface {
	tea.Model

	Name() string

	AccentColour() lipgloss.Color
	BackgroundColour() lipgloss.Color
	HoverColour() lipgloss.Color
}

type FatalErrorMsg struct {
	Error error
}
