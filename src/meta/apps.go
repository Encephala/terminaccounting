// Global types and behaviour that is generic for each App
package meta

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type App interface {
	Init() tea.Cmd
	Update(tea.Msg) (App, tea.Cmd)
	View() string

	Name() string
	Type() AppType

	CurrentTitle() string
	CurrentViewType() ViewType

	Colour() lipgloss.Color

	CurrentMotionSet() Trie[tea.Msg]
	CurrentCommandSet() Trie[tea.Msg]

	CurrentViewAllowsInsertMode() bool
	AcceptedModels() map[ModelType]struct{}

	MakeLoadListCmd() tea.Cmd

	ReloadView() tea.Cmd
}
