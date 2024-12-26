// Global types and behaviour that is generic for each App
package meta

import (
	"terminaccounting/styles"

	tea "github.com/charmbracelet/bubbletea"
)

type App interface {
	tea.Model

	Name() string

	Colours() styles.AppColours

	CurrentMotionSet() *MotionSet
}
