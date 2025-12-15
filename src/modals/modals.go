package modals

import (
	"terminaccounting/meta"
	"terminaccounting/view"

	tea "github.com/charmbracelet/bubbletea"
)

type ModalManager struct {
	Modal view.View
}

func NewModalManager() ModalManager {
	return ModalManager{
		Modal: nil,
	}
}

func (mm *ModalManager) Init() tea.Cmd {
	return nil
}

func (mm *ModalManager) Update(message tea.Msg) (*ModalManager, tea.Cmd) {
	// TODO: handle ReloadViewMsg

	// TODO: Send message to current modal

	return mm, nil
}

func (mm *ModalManager) View() string {
	return meta.ModalStyle.Render(mm.Modal.View())
}

func (mm *ModalManager) CurrentMotionSet() meta.MotionSet {
	return mm.Modal.MotionSet()
}

func (mm *ModalManager) CurrentCommandSet() meta.CommandSet {
	return mm.Modal.CommandSet()
}
