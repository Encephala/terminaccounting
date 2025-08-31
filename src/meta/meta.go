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
	CurrentCommandSet() *CommandSet

	AcceptedModels() map[ModelType]struct{}

	MakeLoadListCmd() tea.Cmd
	// TODO: I'd like for this to take an int argument, but hard rn because it's only called in
	// generic DetailView's Init, which doesn't know of any id's.
	// Think that's circumventable though? DetailView Init can send a messagecmd to make the app init the view
	// Kinda jank I guess but it works? Asserting view type rn is even more jank imo
	// Actually much simpler: detailview holds the app and the id already. Easy fix then
	MakeLoadRowsCmd(modelId int) tea.Cmd
}
