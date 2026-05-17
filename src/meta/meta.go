package meta

import "github.com/charmbracelet/lipgloss"

func RenderBoolean(reconciled bool) string {
	if reconciled {
		// Font Awesome checkbox because it's monospace, standard emoji character is too wide
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00")).Render("")
	}

	return "□"
}
