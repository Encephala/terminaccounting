package modals

import (
	"terminaccounting/meta"

	tea "github.com/charmbracelet/bubbletea"
)

type textModal struct {
	message string
}

func NewTextModal(message string) *textModal {
	return &textModal{
		message: message,
	}
}

func (tm *textModal) Init() tea.Cmd {
	return nil
}

func (tm *textModal) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	return tm, nil
}

func (tm *textModal) View() string {
	return meta.ModalStyle.Render(tm.message)
}

func (tm *textModal) AcceptedModels() map[meta.ModelType]struct{} {
	return make(map[meta.ModelType]struct{})
}

func (tm *textModal) MotionSet() meta.MotionSet {
	return meta.MotionSet{}
}

func (tm *textModal) CommandSet() meta.CommandSet {
	return meta.CommandSet{}
}
