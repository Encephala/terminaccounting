package journals

import "github.com/charmbracelet/lipgloss"

type app struct{}

var Journals = &app{}

func (a *app) Name() string {
	return "Journals"
}

func (a *app) Render() string {
	return "TODO journals"
}

func (a *app) AccentColour() lipgloss.Color {
	return lipgloss.Color("#F6D6D6D0")
}
