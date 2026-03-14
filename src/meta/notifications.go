package meta

import (
	"log/slog"

	"github.com/charmbracelet/lipgloss"
)

type Notification struct {
	Text    string
	IsError bool
}

func (nm Notification) String() string {
	if nm.IsError {
		return lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render(nm.Text)
	}

	return nm.Text
}

func (nm Notification) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("text", nm.Text),
		slog.Bool("isError", nm.IsError),
	)
}
