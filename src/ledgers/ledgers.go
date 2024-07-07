package ledgers

import "github.com/charmbracelet/lipgloss"

type app struct {
}

var Ledgers = &app{}

func (a *app) Name() string {
	return "Ledgers"
}

func (a *app) Render() string {
	return "TODO ledgers"
}

func (a *app) AccentColour() lipgloss.Color {
	return lipgloss.Color("#A1EEBDD0")
}
