package entries

import "github.com/charmbracelet/lipgloss"

type app struct{}

var Entries = &app{}

func (a *app) Name() string {
	return "Entries"
}

func (a *app) Render() string {
	return "TODO entries"
}

func (a *app) AccentColour() lipgloss.Color {
	return lipgloss.Color("#F0F1B2D0")
}
