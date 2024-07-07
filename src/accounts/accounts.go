package accounts

import "github.com/charmbracelet/lipgloss"

type app struct {
}

var Accounts = &app{}

func (a *app) Name() string {
	return "Accounts"
}

func (a *app) Render() string {
	return "TODO accounts"
}

func (a *app) AccentColour() lipgloss.Color {
	return lipgloss.Color("#7BD3EAD0")
}
