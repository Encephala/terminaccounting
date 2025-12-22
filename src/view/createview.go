package view

import (
	"cmp"
	"slices"
	"strings"
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type genericCreateView interface {
	View

	title() string

	inputs() []viewable
	inputNames() []string

	getActiveInput() *int

	getColours() meta.AppColours
}

type createViewManager struct {
	specificView genericCreateView
}

func NewCreateViewManager(specificView genericCreateView) *createViewManager {
	return &createViewManager{
		specificView: specificView,
	}
}

func (cvm *createViewManager) Init() tea.Cmd {
	return cvm.specificView.Init()
}

func (cvm *createViewManager) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	newView, cmd := cvm.specificView.Update(message)
	cvm.specificView = newView.(genericCreateView)

	return cvm, cmd
}

func (cvm *createViewManager) View() string {
	var result strings.Builder

	titleStyle := lipgloss.NewStyle().Background(cvm.specificView.getColours().Background).Padding(0, 1).MarginLeft(2)
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

func (cvm *createViewManager) AcceptedModels() map[meta.ModelType]struct{} {
	return cvm.specificView.AcceptedModels()
}

func (cvm *createViewManager) MotionSet() meta.MotionSet {
	return cvm.specificView.MotionSet()
}

func (cvm *createViewManager) CommandSet() meta.CommandSet {
	return cvm.specificView.CommandSet()
}

func (cvm *createViewManager) Reload() View {
	return cvm.specificView.Reload()
}
