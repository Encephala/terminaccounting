package view

import (
	"cmp"
	"slices"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type genericMutateView interface {
	View

	title() string

	inputs() []viewable
	inputNames() []string

	getActiveInput() *int

	getColours() meta.AppColours
}

type mutateViewManager struct {
	specificView genericMutateView
}

func NewMutateViewManager(specificView genericMutateView) *mutateViewManager {
	return &mutateViewManager{
		specificView: specificView,
	}
}

func (cvm *mutateViewManager) Init() tea.Cmd {
	return cvm.specificView.Init()
}

func (cvm *mutateViewManager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	newView, cmd := cvm.specificView.Update(message)
	cvm.specificView = newView.(genericMutateView)

	return cvm, cmd
}

func (cvm *mutateViewManager) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(cvm.specificView.getColours().Background).Padding(0, 1)
	result.WriteString(titleStyle.Render(cvm.specificView.title()))

	result.WriteString("\n\n")

	sectionStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		UnsetWidth().
		Align(lipgloss.Left)
	highlightStyle := sectionStyle.Foreground(cvm.specificView.getColours().Foreground)

	names := cvm.specificView.inputNames()
	inputs := cvm.specificView.inputs()

	if len(names) != len(inputs) {
		panic("what in the fuck")
	}

	styles := slices.Repeat([]lipgloss.Style{sectionStyle}, len(names))

	styles[*cvm.specificView.getActiveInput()] = highlightStyle

	// +2 for padding
	maxNameColWidth := len(slices.MaxFunc(names, func(name string, other string) int {
		return cmp.Compare(len(name), len(other))
	})) + 2

	for i := range names {
		result.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionStyle.Width(maxNameColWidth).Render(names[i]),
			" ",
			styles[i].Render(inputs[i].View()),
		))

		result.WriteString("\n")
	}

	return lipgloss.NewStyle().MarginLeft(2).Render(result.String())
}

func (cvm *mutateViewManager) AcceptedModels() map[meta.ModelType]struct{} {
	return cvm.specificView.AcceptedModels()
}

func (cvm *mutateViewManager) MotionSet() meta.MotionSet {
	return cvm.specificView.MotionSet()
}

func (cvm *mutateViewManager) CommandSet() meta.CommandSet {
	return cvm.specificView.CommandSet()
}

func (cvm *mutateViewManager) Reload() View {
	return cvm.specificView.Reload()
}
